package dto

import "time"

// Auth

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	FullName string `json:"fullName"`
}

// Property / kind / room (responses)

type PropertyResponse struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	ShortTitle  string `json:"shortTitle"`
	Address     string `json:"address"`
	Description string `json:"description,omitempty"`
	Accent      string `json:"accent,omitempty"`
}

type RoomKindResponse struct {
	ID          int64   `json:"id"`
	PropertyID  int64   `json:"propertyId"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	BaseRate    float64 `json:"baseRate"`
	Capacity    int     `json:"capacity"`
	Area        float64 `json:"area"`
	Beds        string  `json:"beds"`
}

type PhotoResponse struct {
	ID       int64  `json:"id"`
	RoomID   int64  `json:"roomId"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	IsCover  bool   `json:"isCover"`
}

type RoomResponse struct {
	ID            int64           `json:"id"`
	PropertyID    int64           `json:"propertyId"`
	PropertyTitle string          `json:"propertyTitle,omitempty"`
	PropertySlug  string          `json:"propertySlug,omitempty"`
	KindID        int64           `json:"kindId"`
	KindTitle     string          `json:"kindTitle,omitempty"`
	KindSlug      string          `json:"kindSlug,omitempty"`
	Label         string          `json:"label"`
	ShortLabel    string          `json:"shortLabel"`
	Floor         int             `json:"floor"`
	Side          string          `json:"side,omitempty"`
	Area          float64         `json:"area,omitempty"`
	Orientation   string          `json:"orientation,omitempty"`
	Photos        []PhotoResponse `json:"photos"`
}

// Guest

type GuestRequest struct {
	FullName  string  `json:"fullName" validate:"required"`
	Phone     *string `json:"phone,omitempty"`
	Email     *string `json:"email,omitempty"`
	DocType   *string `json:"docType,omitempty"`
	DocNumber *string `json:"docNumber,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type GuestResponse struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"fullName"`
	Phone     string    `json:"phone,omitempty"`
	Email     string    `json:"email,omitempty"`
	DocType   string    `json:"docType,omitempty"`
	DocNumber string    `json:"docNumber,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Booking

type CreateBookingRequest struct {
	RoomID       int64         `json:"roomId" validate:"required"`
	CheckIn      string        `json:"checkIn" validate:"required"`  // YYYY-MM-DD
	CheckOut     string        `json:"checkOut" validate:"required"` // YYYY-MM-DD
	CheckInTime  string        `json:"checkInTime,omitempty"`       // HH:MM, дефолт 14:00
	CheckOutTime string        `json:"checkOutTime,omitempty"`      // HH:MM, дефолт 12:00
	Adults       int           `json:"adults" validate:"required,min=1"`
	Source       string        `json:"source,omitempty"`
	GuestID      *int64        `json:"guestId,omitempty"`
	Guest        *GuestRequest `json:"guest,omitempty"`
	Total        *float64      `json:"totalAmount,omitempty"`
	Prepay       float64       `json:"prepayment,omitempty"`
	Notes        *string       `json:"notes,omitempty"`
}

type UpdateBookingRequest struct {
	CheckIn      *string  `json:"checkIn,omitempty"`
	CheckOut     *string  `json:"checkOut,omitempty"`
	CheckInTime  *string  `json:"checkInTime,omitempty"`
	CheckOutTime *string  `json:"checkOutTime,omitempty"`
	Adults       *int     `json:"adults,omitempty"`
	GuestID      *int64   `json:"guestId,omitempty"`
	Total        *float64 `json:"totalAmount,omitempty"`
	Prepayment   *float64 `json:"prepayment,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

type ChangeStatusRequest struct {
	To     string  `json:"to" validate:"required"`
	Reason *string `json:"reason,omitempty"`
}

type BookingResponse struct {
	ID           int64                 `json:"id"`
	Code         string                `json:"code"`
	RoomID       int64                 `json:"roomId"`
	RoomLabel    string                `json:"roomLabel,omitempty"`
	PropertyID   int64                 `json:"propertyId"`
	PropertyTitle string               `json:"propertyTitle,omitempty"`
	GuestID      *int64                `json:"guestId,omitempty"`
	Guest        *GuestResponse        `json:"guest,omitempty"`
	CheckIn      string                `json:"checkIn"`
	CheckOut     string                `json:"checkOut"`
	CheckInTime  string                `json:"checkInTime,omitempty"`
	CheckOutTime string                `json:"checkOutTime,omitempty"`
	Nights       int                   `json:"nights"`
	Adults       int                   `json:"adults"`
	Status       string                `json:"status"`
	Source       string                `json:"source"`
	TotalAmount  float64               `json:"totalAmount"`
	Prepayment   float64               `json:"prepayment"`
	Notes        string                `json:"notes,omitempty"`
	CreatedBy    *int64                `json:"createdBy,omitempty"`
	CreatedAt    time.Time             `json:"createdAt"`
	UpdatedAt    time.Time             `json:"updatedAt"`
	History      []StatusEventResponse `json:"history,omitempty"`
}

type StatusEventResponse struct {
	FromStatus string    `json:"fromStatus,omitempty"`
	ToStatus   string    `json:"toStatus"`
	Reason     string    `json:"reason,omitempty"`
	ChangedBy  *int64    `json:"changedBy,omitempty"`
	ChangedAt  time.Time `json:"changedAt"`
}

// Chessboard — полосы броней с почасовой точностью поверх дневной сетки.
type ChessboardBar struct {
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

type ChessboardResult struct {
	Rooms    []RoomResponse  `json:"rooms"`
	Days     []string        `json:"days"`     // YYYY-MM-DD
	Bookings []ChessboardBar `json:"bookings"` // активные: new, confirmed, checked_in
}

// Rate

type RateRequest struct {
	KindID      int64   `json:"kindId" validate:"required"`
	DateFrom    string  `json:"dateFrom" validate:"required"`
	DateTo      string  `json:"dateTo" validate:"required"`
	WeekdayRate float64 `json:"weekdayRate" validate:"required,min=0"`
	WeekendRate float64 `json:"weekendRate" validate:"required,min=0"`
}

type RateResponse struct {
	ID          int64  `json:"id"`
	KindID      int64  `json:"kindId"`
	DateFrom    string `json:"dateFrom"`
	DateTo      string `json:"dateTo"`
	WeekdayRate float64 `json:"weekdayRate"`
	WeekendRate float64 `json:"weekendRate"`
}

// User (admin)

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	Role      string `json:"role" validate:"required,oneof=admin manager receptionist"`
	FullName  string `json:"fullName" validate:"required"`
}

type UpdateUserRequest struct {
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=admin manager receptionist"`
	FullName *string `json:"fullName,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6"`
}
