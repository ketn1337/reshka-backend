// Package bnovo — минимальный клиент Bnovo PMS API v1.
// Используется CLI-утилитой cmd/import-bnovo для разового импорта броней.
//
// Базовый URL: https://api.pms.bnovo.ru
// Методы, доступные в бесплатной версии: /api/v1/auth, /api/v1/bookings, /api/v1/bookings/{id}.
package bnovo

import "time"

// =========================
// Запросы/ответы
// =========================

type authRequest struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	// Bnovo в нашем аккаунте оборачивает ответ в {"data": {...}} — обёртка
	// разворачивается в client.Auth.
	Data *authResponse `json:"data,omitempty"`
}

// BookingsListResponse — ответ на GET /api/v1/bookings.
// Реальная структура (виден в swagger + живой ответ):
//   { "data": { "bookings": [...], "meta": {...} }, "meta": {...} }
//
// «Внешний» meta — общая пагинация, «внутренний» data.meta — то же самое
// (Bnovo дублирует). Поддерживаем оба уровня.
type BookingsListResponse struct {
	Data *BookingsData `json:"data"`
	Meta *PageMeta     `json:"meta,omitempty"`
}

type BookingsData struct {
	Bookings []RawBooking `json:"bookings"`
	Meta     *PageMeta    `json:"meta,omitempty"`
}

type PageMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// =========================
// Доменные структуры Bnovo
// =========================

type RawBooking struct {
	ID         int64       `json:"id"`
	Number     string      `json:"number"`
	Amount     float64     `json:"amount"`
	HotelID    int64       `json:"hotel_id"`
	RoomName   string      `json:"room_name"` // top-level, напр. "Пионерская 63 - О9"
	PlanName   string      `json:"plan_name"`
	Source     RawSource   `json:"source"`
	Status     RawStatus   `json:"status"`
	Customer   RawCustomer `json:"customer"`
	Dates      RawDates    `json:"dates"`
	Discount   *RawDiscount `json:"discount,omitempty"`
	Extra      *RawExtra   `json:"extra,omitempty"` // {"adults":N, "children":N}
	RoomID     int64       `json:"room_id"`
	RoomTypeID int64       `json:"room_type_id"`
	// Prices — посуточная разбивка. Один элемент = одна занятая ночь в одной комнате.
	// У мультирумной брони здесь несколько разных room_id.
	Prices []RawPrice `json:"prices"`
}

type RawStatus struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type RawSource struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	Commission float64 `json:"commission,omitempty"`
}

type RawCustomer struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Notes   string `json:"notes,omitempty"`
}

type RawDates struct {
	CreateDate        string `json:"create_date"`
	Arrival           string `json:"arrival"`     // "2026-06-15 14:00:00+03"
	Departure         string `json:"departure"`   // "2026-06-17 12:00:00+03"
	RealArrival       string `json:"real_arrival,omitempty"`
	RealDeparture     string `json:"real_departure,omitempty"`
	OriginalArrival   string `json:"original_arrival,omitempty"`
	OriginalDeparture string `json:"original_departure,omitempty"`
	CancelDate        string `json:"cancel_date,omitempty"`
	UpdateDate        string `json:"update_date,omitempty"`
}

type RawDiscount struct {
	Type     int    `json:"type"`
	Amount   int    `json:"amount"`
	ReasonID int    `json:"reason_id"`
	Reason   string `json:"reason"`
}

type RawExtra struct {
	Adults   int `json:"adults"`
	Children int `json:"children"`
}

type RawPrice struct {
	Date           string  `json:"date"`         // "YYYY-MM-DD"
	Price          float64 `json:"price"`
	Policy         string  `json:"policy,omitempty"`
	RoomID         int64   `json:"room_id"`
	RoomTypeID     int64   `json:"room_type_id"`
	RoomTypeName   string  `json:"room_type_name"`
	RealRoomTypeID int64   `json:"real_room_type_id,omitempty"`
	RoomName       string  `json:"room_name,omitempty"`
	PlanName       string  `json:"plan_name,omitempty"`
}

// =========================
// Конфигурация
// =========================

// Config — параметры доступа. AccountID и APIKey юзер вписывает в bnovo-rooms.json.
type Config struct {
	BaseURL    string // дефолт https://api.pms.bnovo.ru
	AccountID  int64
	APIKey     string
	HTTPClient HTTPClient // переопределяется для тестов; nil → дефолтный *http.Client
}

// =========================
// Внутреннее
// =========================

// cachedToken хранится в клиенте до истечения exp.
type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}
