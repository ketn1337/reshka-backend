package domain

import "time"

// =========================
// Users
// =========================

const (
	RoleAdmin       = "admin"
	RoleManager     = "manager"
	RoleReceptionist = "receptionist"
)

type User struct {
	ID        int64     `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Role      string    `db:"role" json:"role"`
	FullName  string    `db:"full_name" json:"fullName"`
	IsActive  bool      `db:"is_active" json:"isActive"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// =========================
// Property
// =========================

type Property struct {
	ID          int64     `db:"id" json:"id"`
	Slug        string    `db:"slug" json:"slug"`
	Title       string    `db:"title" json:"title"`
	ShortTitle  string    `db:"short_title" json:"shortTitle"`
	Address     string    `db:"address" json:"address"`
	Description *string   `db:"description" json:"description,omitempty"`
	Accent      *string   `db:"accent" json:"accent,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// =========================
// RoomKind
// =========================

type RoomKind struct {
	ID          int64   `db:"id" json:"id"`
	PropertyID  int64   `db:"property_id" json:"propertyId"`
	Slug        string  `db:"slug" json:"slug"`
	Title       string  `db:"title" json:"title"`
	Description *string `db:"description" json:"description,omitempty"`
	BaseRate    float64 `db:"base_rate" json:"baseRate"`
	Capacity    int     `db:"capacity" json:"capacity"`
	Area        float64 `db:"area" json:"area"`
	Beds        string  `db:"beds" json:"beds"`
}

// =========================
// Room
// =========================

const (
	OrientationInner    = "inner"
	OrientationStreet   = "street"
	OrientationCourtyard = "courtyard"
)

type Room struct {
	ID          int64     `db:"id" json:"id"`
	PropertyID  int64     `db:"property_id" json:"propertyId"`
	KindID      int64     `db:"kind_id" json:"kindId"`
	Label       string    `db:"label" json:"label"`
	ShortLabel  string    `db:"short_label" json:"shortLabel"`
	Floor       int       `db:"floor" json:"floor"`
	Side        *string   `db:"side" json:"side,omitempty"`
	Area        *float64  `db:"area" json:"area,omitempty"`
	Orientation *string   `db:"orientation" json:"orientation,omitempty"`
	IsActive    bool      `db:"is_active" json:"isActive"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// =========================
// Photo
// =========================

type Photo struct {
	ID        int64     `db:"id" json:"id"`
	RoomID    int64     `db:"room_id" json:"roomId"`
	Filename  string    `db:"filename" json:"filename"`
	Position  int       `db:"position" json:"position"`
	IsCover   bool      `db:"is_cover" json:"isCover"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

// =========================
// Guest
// =========================

type Guest struct {
	ID        int64     `db:"id" json:"id"`
	FullName  string    `db:"full_name" json:"fullName"`
	Phone     *string   `db:"phone" json:"phone,omitempty"`
	Email     *string   `db:"email" json:"email,omitempty"`
	DocType   *string   `db:"doc_type" json:"docType,omitempty"`
	DocNumber *string   `db:"doc_number" json:"docNumber,omitempty"`
	Notes     *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

// =========================
// Booking
// =========================

const (
	BookingStatusNew         = "new"
	BookingStatusConfirmed   = "confirmed"
	BookingStatusCheckedIn   = "checked_in"
	BookingStatusCheckedOut  = "checked_out"
	BookingStatusCancelled   = "cancelled"
	BookingStatusNoShow      = "no_show"
)

const (
	BookingSourceDirect = "direct"
	BookingSourceSite   = "site"
	BookingSourceOTA    = "ota"
	BookingSourcePhone  = "phone"
	BookingSourceMax    = "max"
)

type Booking struct {
	ID           int64     `db:"id" json:"id"`
	Code         string    `db:"code" json:"code"`
	RoomID       int64     `db:"room_id" json:"roomId"`
	GuestID      *int64    `db:"guest_id" json:"guestId,omitempty"`
	CheckIn      time.Time `db:"check_in" json:"checkIn"`
	CheckOut     time.Time `db:"check_out" json:"checkOut"`
	CheckInTime  string    `db:"check_in_time" json:"checkInTime"`   // "HH:MM:SS"
	CheckOutTime string    `db:"check_out_time" json:"checkOutTime"` // "HH:MM:SS"
	Adults       int       `db:"adults" json:"adults"`
	Status       string    `db:"status" json:"status"`
	Source       string    `db:"source" json:"source"`
	TotalAmount  float64   `db:"total_amount" json:"totalAmount"`
	Prepayment   float64   `db:"prepayment" json:"prepayment"`
	Notes        *string   `db:"notes" json:"notes,omitempty"`
	CreatedBy    *int64    `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

type BookingStatusEvent struct {
	ID         int64     `db:"id" json:"id"`
	BookingID  int64     `db:"booking_id" json:"bookingId"`
	FromStatus *string   `db:"from_status" json:"fromStatus,omitempty"`
	ToStatus   string    `db:"to_status" json:"toStatus"`
	ChangedBy  *int64    `db:"changed_by" json:"changedBy,omitempty"`
	Reason     *string   `db:"reason" json:"reason,omitempty"`
	ChangedAt  time.Time `db:"changed_at" json:"changedAt"`
}

// =========================
// Rate
// =========================

type Rate struct {
	ID          int64     `db:"id" json:"id"`
	KindID      int64     `db:"kind_id" json:"kindId"`
	DateFrom    time.Time `db:"date_from" json:"dateFrom"`
	DateTo      time.Time `db:"date_to" json:"dateTo"`
	WeekdayRate float64   `db:"weekday_rate" json:"weekdayRate"`
	WeekendRate float64   `db:"weekend_rate" json:"weekendRate"`
}
