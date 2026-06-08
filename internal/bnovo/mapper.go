package bnovo

import (
	"strings"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

// MapStatus переводит имя статуса из Bnovo в наш enum.
// Реальные имена (из живого ответа Bnovo): "Новое", "Проверено", "Заселен", "Отменен".
// Маппим по подстроке. Если не распознали — возвращаем "new" + false.
func MapStatus(bnovoName string) (domainStatus string, recognized bool) {
	s := strings.ToLower(strings.TrimSpace(bnovoName))
	switch {
	case strings.Contains(s, "отмен"):
		return domain.BookingStatusCancelled, true
	case strings.Contains(s, "неяв"), strings.Contains(s, "no_show"), strings.Contains(s, "no show"):
		return domain.BookingStatusNoShow, true
	case strings.Contains(s, "высел"), strings.Contains(s, "выезд"), strings.Contains(s, "checked_out"), strings.Contains(s, "выписан"), strings.Contains(s, "выехал"):
		return domain.BookingStatusCheckedOut, true
	case strings.Contains(s, "засел"), strings.Contains(s, "прожив"), strings.Contains(s, "in_house"), strings.Contains(s, "checked_in"), strings.Contains(s, "заехал"):
		return domain.BookingStatusCheckedIn, true
	case strings.Contains(s, "провер"), strings.Contains(s, "подтверж"), strings.Contains(s, "confirmed"), strings.Contains(s, "гарант"), strings.Contains(s, "verified"):
		return domain.BookingStatusConfirmed, true
	case strings.Contains(s, "новое"), strings.Contains(s, "новая"), strings.Contains(s, "new"), strings.Contains(s, "ожидан"), strings.Contains(s, "брон"):
		return domain.BookingStatusNew, true
	}
	return domain.BookingStatusNew, false
}

// MapSource переводит имя источника из Bnovo в наш enum.
// Реальные имена (из живого ответа Bnovo): "Avito", "Kvartirka", "Островок!",
// "Прямое", "Суточно.ру".
func MapSource(bnovoName string) (domainStatus string, recognized bool) {
	s := strings.ToLower(strings.TrimSpace(bnovoName))
	switch {
	case s == "":
		return domain.BookingSourceSite, false
	case strings.Contains(s, "max"):
		return domain.BookingSourceMax, true
	case strings.Contains(s, "суточно"), strings.Contains(s, "sutochno"),
		strings.Contains(s, "avito"), strings.Contains(s, "авито"),
		strings.Contains(s, "островок"), strings.Contains(s, "ostrovok"),
		strings.Contains(s, "kvartirka"), strings.Contains(s, "квартирка"),
		strings.Contains(s, "твил"), strings.Contains(s, "tvl"),
		strings.Contains(s, "booking.com"), strings.Contains(s, "booking"),
		strings.Contains(s, "ozon"), strings.Contains(s, "озон"),
		strings.Contains(s, "airbnb"),
		strings.Contains(s, "101hotels"),
		strings.Contains(s, "ota"), strings.Contains(s, "канал"),
		strings.Contains(s, "bronevik"), strings.Contains(s, "броневик"):
		return domain.BookingSourceOTA, true
	case strings.Contains(s, "прямое"), strings.Contains(s, "прям"):
		return domain.BookingSourceDirect, true
	case strings.Contains(s, "сайт"), strings.Contains(s, "site"), strings.Contains(s, "website"), strings.Contains(s, "веб"):
		return domain.BookingSourceSite, true
	case strings.Contains(s, "телефон"), strings.Contains(s, "phone"), strings.Contains(s, "whatsapp"), strings.Contains(s, "telegram"):
		return domain.BookingSourcePhone, true
	case strings.Contains(s, "лично"), strings.Contains(s, "walk-in"), strings.Contains(s, "walk in"),
		strings.Contains(s, "direct"), strings.Contains(s, "ресепшн"):
		return domain.BookingSourceDirect, true
	}
	return domain.BookingSourceSite, false
}

// ExtractRoomIDs возвращает уникальные room_id из prices[] в порядке первого появления.
func ExtractRoomIDs(prices []RawPrice) []int64 {
	seen := make(map[int64]struct{}, 4)
	out := make([]int64, 0, 4)
	for _, p := range prices {
		if _, ok := seen[p.RoomID]; ok {
			continue
		}
		seen[p.RoomID] = struct{}{}
		out = append(out, p.RoomID)
	}
	return out
}

// FullName склеивает имя и фамилию гостя из Bnovo.
// Если surname == "—", его отбрасываем. Если оба пустые — "—".
func FullName(c RawCustomer) string {
	parts := []string{}
	if c.Name != "" && c.Name != "—" {
		parts = append(parts, c.Name)
	}
	if c.Surname != "" && c.Surname != "—" {
		parts = append(parts, c.Surname)
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " ")
}

// Adults достаёт количество гостей из booking.extra.adults.
// Если поля нет — возвращает 1 (минимум, требуемый валидацией).
func Adults(b RawBooking) int {
	if b.Extra != nil && b.Extra.Adults > 0 {
		return b.Extra.Adults
	}
	return 1
}

// ExtractPropertyPrefix выдёргивает префикс адреса из room_name.
// "Пионерская 63 - О9" → "Пионерская 63"
// "Аллея Труда 21 - Номер 5" → "Аллея Труда 21"
// "Дмитровское шоссе д.107А корпус 5 - 27/3" → "Дмитровское шоссе д.107А корпус 5"
func ExtractPropertyPrefix(roomName string) string {
	// Bnovo использует " - " как разделитель между адресом и номером.
	if i := strings.Index(roomName, " - "); i > 0 {
		return strings.TrimSpace(roomName[:i])
	}
	// Если разделителя нет — возвращаем всё.
	return strings.TrimSpace(roomName)
}
