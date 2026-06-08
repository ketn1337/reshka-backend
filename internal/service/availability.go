package service

import (
	"context"
	"time"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type AvailabilityService struct {
	rooms     *repo.RoomRepo
	bookings  *repo.BookingRepo
	roomKinds *repo.RoomKindRepo
	props     *repo.PropertyRepo
	guests    *repo.GuestRepo
}

func NewAvailabilityService(r *repo.RoomRepo, b *repo.BookingRepo, rk *repo.RoomKindRepo, p *repo.PropertyRepo, g *repo.GuestRepo) *AvailabilityService {
	return &AvailabilityService{rooms: r, bookings: b, roomKinds: rk, props: p, guests: g}
}

// ChessboardResult — ответ на /api/admin/chessboard.
// Rooms несут propertyTitle/kindTitle (см. mappers.toRoomResp в DTO-слое).
type ChessboardResult struct {
	Rooms    []domain.Room     `json:"rooms"`
	Days     []string          `json:"days"`
	Bookings []ChessboardBooking `json:"bookings"`
}

type ChessboardBooking struct {
	BookingID   int64   `json:"bookingId"`
	RoomID      int64   `json:"roomId"`
	Code        string  `json:"code"`
	StartISO    string  `json:"startISO"` // "2026-06-15T14:00:00+10:00"
	EndISO      string  `json:"endISO"`   // "2026-06-17T12:00:00+10:00"
	Nights      int     `json:"nights"`
	Status      string  `json:"status"`
	GuestName   string  `json:"guestName"`
	Adults      int     `json:"adults"`
	Source      string  `json:"source"`
	TotalAmount float64 `json:"totalAmount"`
}

// Chessboard возвращает все активные номера обоих объектов и активные брони (new/confirmed/checked_in)
// в виде полос с timestamp-границами. Активные статусы: new, confirmed, checked_in.
// checked_out, cancelled, no_show — НЕ показываются (решение пользователя).
func (s *AvailabilityService) Chessboard(ctx context.Context, from time.Time, days int) (*ChessboardResult, error) {
	if days <= 0 || days > 60 {
		days = 14
	}
	// Берём ВСЕ номера обоих объектов.
	rooms, err := s.rooms.List(ctx, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	// Сортировка: сначала по property_id, потом по label (Номер 1, 2, 3…).
	// rooms.List уже сортирует (property_id, short_label) — проверим, если нет, доп. сортировка ниже.
	ids := make([]int64, 0, len(rooms))
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}

	dayList := make([]string, days)
	for i := 0; i < days; i++ {
		dayList[i] = from.AddDate(0, 0, i).Format("2006-01-02")
	}

	bs, err := s.bookings.OverlappingActive(ctx, ids, from, from.AddDate(0, 0, days))
	if err != nil {
		return nil, err
	}

	// Гости: батч по id, чтобы достать имя для каждой брони.
	guestIDs := make([]int64, 0, len(bs))
	for _, b := range bs {
		if b.GuestID != nil {
			guestIDs = append(guestIDs, *b.GuestID)
		}
	}
	guestsByID := make(map[int64]string, len(guestIDs))
	if len(guestIDs) > 0 {
		gs, _ := s.guests.ListByIDs(ctx, guestIDs)
		for _, g := range gs {
			guestsByID[g.ID] = g.FullName
		}
	}

	bookings := make([]ChessboardBooking, 0, len(bs))
	for _, b := range bs {
		startISO := b.CheckIn.Format("2006-01-02") + "T" + normalizeTime(b.CheckInTime, "14:00:00")
		endISO := b.CheckOut.Format("2006-01-02") + "T" + normalizeTime(b.CheckOutTime, "12:00:00")
		guest := ""
		if b.GuestID != nil {
			guest = guestsByID[*b.GuestID]
		}
		bookings = append(bookings, ChessboardBooking{
			BookingID:   b.ID,
			RoomID:      b.RoomID,
			Code:        b.Code,
			StartISO:    startISO,
			EndISO:      endISO,
			Nights:      nightsBetween(b.CheckIn, b.CheckOut),
			Status:      b.Status,
			GuestName:   guest,
			Adults:      b.Adults,
			Source:      b.Source,
			TotalAmount: b.TotalAmount,
		})
	}

	return &ChessboardResult{
		Rooms:    rooms,
		Days:     dayList,
		Bookings: bookings,
	}, nil
}

func normalizeTime(v, def string) string {
	if v == "" {
		return def
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

type AvailabilityRoom struct {
	Room      domain.Room    `json:"room"`
	Kind      domain.RoomKind `json:"kind"`
	Available bool           `json:"available"`
	Total     float64        `json:"total"`
}

func (s *AvailabilityService) Search(ctx context.Context, propertyID, kindID int64, checkIn, checkOut time.Time) ([]AvailabilityRoom, error) {
	return s.searchInternal(ctx, propertyID, &kindID, checkIn, checkOut)
}

// SearchAll — все типы номеров на объекте.
func (s *AvailabilityService) SearchAll(ctx context.Context, propertyID int64, checkIn, checkOut time.Time) ([]AvailabilityRoom, error) {
	return s.searchInternal(ctx, propertyID, nil, checkIn, checkOut)
}

func (s *AvailabilityService) searchInternal(ctx context.Context, propertyID int64, kindID *int64, checkIn, checkOut time.Time) ([]AvailabilityRoom, error) {
	rooms, err := s.rooms.List(ctx, &propertyID, kindID, nil)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rooms))
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}
	bs, err := s.bookings.OverlappingActive(ctx, ids, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	busy := make(map[int64]struct{}, len(bs))
	for _, b := range bs {
		busy[b.RoomID] = struct{}{}
	}

	kinds, err := s.roomKinds.ListByProperty(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	kindsByID := make(map[int64]domain.RoomKind, len(kinds))
	for _, k := range kinds {
		kindsByID[k.ID] = k
	}

	out := make([]AvailabilityRoom, 0, len(rooms))
	for _, r := range rooms {
		_, isBusy := busy[r.ID]
		out = append(out, AvailabilityRoom{
			Room:      r,
			Kind:      kindsByID[r.KindID],
			Available: !isBusy,
		})
	}
	return out, nil
}
