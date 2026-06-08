package handlers

import (
	"context"
	"time"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/httpapi/dto"
)

func toPropertyResp(p domain.Property) dto.PropertyResponse {
	r := dto.PropertyResponse{
		ID:         p.ID,
		Slug:       p.Slug,
		Title:      p.Title,
		ShortTitle: p.ShortTitle,
		Address:    p.Address,
	}
	if p.Description != nil {
		r.Description = *p.Description
	}
	if p.Accent != nil {
		r.Accent = *p.Accent
	}
	return r
}

func toKindResp(k domain.RoomKind) dto.RoomKindResponse {
	r := dto.RoomKindResponse{
		ID:         k.ID,
		PropertyID: k.PropertyID,
		Slug:       k.Slug,
		Title:      k.Title,
		BaseRate:   k.BaseRate,
		Capacity:   k.Capacity,
		Area:       k.Area,
		Beds:       k.Beds,
	}
	if k.Description != nil {
		r.Description = *k.Description
	}
	return r
}

func toPhotoResp(p domain.Photo) dto.PhotoResponse {
	return dto.PhotoResponse{
		ID:       p.ID,
		RoomID:   p.RoomID,
		Filename: p.Filename,
		URL:      "/photos/room_" + int64ToStr(p.RoomID) + "/" + p.Filename,
		Position: p.Position,
		IsCover:  p.IsCover,
	}
}

func toRoomResp(r domain.Room, photos []domain.Photo, prop *domain.Property, kind *domain.RoomKind) dto.RoomResponse {
	out := dto.RoomResponse{
		ID:         r.ID,
		PropertyID: r.PropertyID,
		KindID:     r.KindID,
		Label:      r.Label,
		ShortLabel: r.ShortLabel,
		Floor:      r.Floor,
		Photos:     []dto.PhotoResponse{},
	}
	if prop != nil {
		out.PropertyTitle = prop.Title
		out.PropertySlug = prop.Slug
	}
	if kind != nil {
		out.KindTitle = kind.Title
		out.KindSlug = kind.Slug
	}
	if r.Side != nil {
		out.Side = *r.Side
	}
	if r.Area != nil {
		out.Area = *r.Area
	}
	if r.Orientation != nil {
		out.Orientation = *r.Orientation
	}
	for _, p := range photos {
		out.Photos = append(out.Photos, toPhotoResp(p))
	}
	return out
}

func toGuestResp(g domain.Guest) dto.GuestResponse {
	r := dto.GuestResponse{ID: g.ID, FullName: g.FullName, CreatedAt: g.CreatedAt}
	if g.Phone != nil {
		r.Phone = *g.Phone
	}
	if g.Email != nil {
		r.Email = *g.Email
	}
	if g.DocType != nil {
		r.DocType = *g.DocType
	}
	if g.DocNumber != nil {
		r.DocNumber = *g.DocNumber
	}
	if g.Notes != nil {
		r.Notes = *g.Notes
	}
	return r
}

func toBookingResp(b domain.Booking, room *domain.Room, prop *domain.Property, guest *domain.Guest, history []domain.BookingStatusEvent) dto.BookingResponse {
	r := dto.BookingResponse{
		ID:           b.ID,
		Code:         b.Code,
		RoomID:       b.RoomID,
		GuestID:      b.GuestID,
		CheckIn:      b.CheckIn.Format("2006-01-02"),
		CheckOut:     b.CheckOut.Format("2006-01-02"),
		CheckInTime:  normalizeHM(b.CheckInTime, "14:00:00"),
		CheckOutTime: normalizeHM(b.CheckOutTime, "12:00:00"),
		Nights:       nightsBetween(b.CheckIn, b.CheckOut),
		Adults:       b.Adults,
		Status:       b.Status,
		Source:       b.Source,
		TotalAmount:  b.TotalAmount,
		Prepayment:   b.Prepayment,
		CreatedBy:    b.CreatedBy,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
	if room != nil {
		r.RoomLabel = room.Label
		r.PropertyID = room.PropertyID
	}
	if prop != nil {
		r.PropertyTitle = prop.Title
	}
	if guest != nil {
		g := toGuestResp(*guest)
		r.Guest = &g
	}
	if b.Notes != nil {
		r.Notes = *b.Notes
	}
	for _, ev := range history {
		ser := dto.StatusEventResponse{
			ToStatus:  ev.ToStatus,
			ChangedBy: ev.ChangedBy,
			ChangedAt: ev.ChangedAt,
		}
		if ev.FromStatus != nil {
			ser.FromStatus = *ev.FromStatus
		}
		if ev.Reason != nil {
			ser.Reason = *ev.Reason
		}
		r.History = append(r.History, ser)
	}
	return r
}

// toChessboardBar собирает полосу для шахматки: склеивает date + time → ISO datetime.
func toChessboardBar(b domain.Booking, guestName string) dto.ChessboardBar {
	checkIn := b.CheckIn.Format("2006-01-02")
	checkOut := b.CheckOut.Format("2006-01-02")
	startISO := checkIn + "T" + normalizeHM(b.CheckInTime, "14:00:00")
	endISO := checkOut + "T" + normalizeHM(b.CheckOutTime, "12:00:00")
	return dto.ChessboardBar{
		BookingID:   b.ID,
		RoomID:      b.RoomID,
		Code:        b.Code,
		StartISO:    startISO,
		EndISO:      endISO,
		Nights:      nightsBetween(b.CheckIn, b.CheckOut),
		Status:      b.Status,
		GuestName:   guestName,
		Adults:      b.Adults,
		Source:      b.Source,
		TotalAmount: b.TotalAmount,
	}
}

// normalizeHM приводит "HH:MM" или "HH:MM:SS" к "HH:MM" для UI.
func normalizeHM(v, def string) string {
	if v == "" {
		return trimSec(def)
	}
	return trimSec(v)
}

func trimSec(v string) string {
	if len(v) >= 5 {
		return v[:5]
	}
	return v
}

// nightsBetween считает ночи между двумя датами (date-only, без времени).
func nightsBetween(checkIn, checkOut time.Time) int {
	d := int(checkOut.Sub(checkIn).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}


// loadPropsAndKinds подгружает properties и room_kinds по id для списка номеров.
// Используется хендлерами, чтобы прокинуть названия в RoomResponse.
func loadPropsAndKinds(ctx context.Context, props propertyGetter, kinds kindGetter, rooms []domain.Room) (map[int64]*domain.Property, map[int64]*domain.RoomKind) {
	propIDs := make(map[int64]struct{}, 8)
	kindIDs := make(map[int64]struct{}, 8)
	for _, r := range rooms {
		propIDs[r.PropertyID] = struct{}{}
		kindIDs[r.KindID] = struct{}{}
	}
	propsByID := make(map[int64]*domain.Property, len(propIDs))
	kindsByID := make(map[int64]*domain.RoomKind, len(kindIDs))
	for id := range propIDs {
		if p, err := props.GetByID(ctx, id); err == nil {
			propsByID[id] = p
		}
	}
	for id := range kindIDs {
		if k, err := kinds.GetByID(ctx, id); err == nil {
			kindsByID[id] = k
		}
	}
	return propsByID, kindsByID
}

// Мини-интерфейсы, чтобы не тянуть gin в helper.
type propertyGetter interface {
	GetByID(ctx context.Context, id int64) (*domain.Property, error)
}

type kindGetter interface {
	GetByID(ctx context.Context, id int64) (*domain.RoomKind, error)
}

func toRateResp(x domain.Rate) dto.RateResponse {
	return dto.RateResponse{
		ID:          x.ID,
		KindID:      x.KindID,
		DateFrom:    x.DateFrom.Format("2006-01-02"),
		DateTo:      x.DateTo.Format("2006-01-02"),
		WeekdayRate: x.WeekdayRate,
		WeekendRate: x.WeekendRate,
	}
}

func int64ToStr(v int64) string {
	// быстрая конвертация без strconv (избегаем лишний импорт)
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
