package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

type BookingService struct {
	bookings  *repo.BookingRepo
	guests    *repo.GuestRepo
	rooms     *repo.RoomRepo
	rates     *repo.RateRepo
	roomKinds *repo.RoomKindRepo
}

func NewBookingService(b *repo.BookingRepo, g *repo.GuestRepo, r *repo.RoomRepo, rt *repo.RateRepo, rk *repo.RoomKindRepo) *BookingService {
	return &BookingService{bookings: b, guests: g, rooms: r, rates: rt, roomKinds: rk}
}

type GuestInput struct {
	FullName  string  `json:"fullName" validate:"required"`
	Phone     *string `json:"phone,omitempty"`
	Email     *string `json:"email,omitempty"`
	DocType   *string `json:"docType,omitempty"`
	DocNumber *string `json:"docNumber,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type CreateBookingInput struct {
	RoomID       int64
	CheckIn      time.Time
	CheckOut     time.Time
	CheckInTime  string // "HH:MM:SS"
	CheckOutTime string // "HH:MM:SS"
	Adults       int
	Source       string
	GuestID      *int64
	Guest        *GuestInput
	Total        *float64
	Prepay       float64
	Notes        *string
	CreatedBy    int64
}

func (in CreateBookingInput) Validate() error {
	if in.RoomID == 0 {
		return fmt.Errorf("%w: roomId required", domain.ErrValidation)
	}
	if !in.CheckOut.After(in.CheckIn) {
		return fmt.Errorf("%w: checkOut must be after checkIn", domain.ErrValidation)
	}
	if in.Adults < 1 {
		return fmt.Errorf("%w: adults >= 1", domain.ErrValidation)
	}
	if in.GuestID == nil && (in.Guest == nil || in.Guest.FullName == "") {
		return fmt.Errorf("%w: guest or guestId required", domain.ErrValidation)
	}
	return nil
}

func (s *BookingService) Create(ctx context.Context, in CreateBookingInput) (*domain.Booking, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	var guestID *int64
	if in.GuestID != nil {
		if _, err := s.guests.GetByID(ctx, *in.GuestID); err != nil {
			return nil, err
		}
		guestID = in.GuestID
	} else {
		g := domain.Guest{
			FullName:  in.Guest.FullName,
			Phone:     in.Guest.Phone,
			Email:     in.Guest.Email,
			DocType:   in.Guest.DocType,
			DocNumber: in.Guest.DocNumber,
			Notes:     in.Guest.Notes,
		}
		id, err := s.guests.Create(ctx, g)
		if err != nil {
			return nil, fmt.Errorf("create guest: %w", err)
		}
		guestID = &id
	}

	room, err := s.rooms.GetByID(ctx, in.RoomID)
	if err != nil {
		return nil, err
	}

	var total float64
	if in.Total != nil {
		total = *in.Total
	} else {
		kind, err := s.roomKinds.GetByID(ctx, room.KindID)
		if err != nil {
			return nil, err
		}
		total, err = s.calcTotal(ctx, kind, in.CheckIn, in.CheckOut)
		if err != nil {
			return nil, err
		}
	}

	year := in.CheckIn.Year()
	seq, err := s.bookings.NextCodeSeq(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("next seq: %w", err)
	}
	code := fmt.Sprintf("BK-%d-%04d", year, seq)

	b := domain.Booking{
		Code:         code,
		RoomID:       in.RoomID,
		GuestID:      guestID,
		CheckIn:      in.CheckIn,
		CheckOut:     in.CheckOut,
		CheckInTime:  firstNonEmpty(in.CheckInTime, "14:00:00"),
		CheckOutTime: firstNonEmpty(in.CheckOutTime, "12:00:00"),
		Adults:       in.Adults,
		Status:       domain.BookingStatusNew,
		Source:       firstNonEmpty(in.Source, domain.BookingSourceSite),
		TotalAmount:  total,
		Prepayment:   in.Prepay,
		Notes:        in.Notes,
		CreatedBy:    &in.CreatedBy,
	}

	id, err := s.bookings.Insert(ctx, b)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23P01" {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("insert booking: %w", err)
	}

	if err := s.bookings.InsertHistory(ctx, domain.BookingStatusEvent{
		BookingID: id,
		ToStatus:  domain.BookingStatusNew,
		ChangedBy: &in.CreatedBy,
	}); err != nil {
		log.Printf("history insert: %v", err)
	}

	return s.bookings.GetByID(ctx, id)
}

func (s *BookingService) calcTotal(ctx context.Context, kind *domain.RoomKind, checkIn, checkOut time.Time) (float64, error) {
	var total float64
	for d := checkIn; d.Before(checkOut); d = d.AddDate(0, 0, 1) {
		rate, err := s.rates.RateForDate(ctx, kind.ID, sql.NullTime{Time: d, Valid: true}, kind.BaseRate)
		if err != nil {
			return 0, err
		}
		total += rate
	}
	return total, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
