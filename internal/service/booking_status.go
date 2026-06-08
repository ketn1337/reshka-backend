package service

import (
	"context"
	"fmt"

	"github.com/ketn1337/reshka-backend/internal/domain"
	"github.com/ketn1337/reshka-backend/internal/repo"
)

// statusTransitions определяет допустимые переходы FSM.
var statusTransitions = map[string][]string{
	domain.BookingStatusNew:        {domain.BookingStatusConfirmed, domain.BookingStatusCancelled},
	domain.BookingStatusConfirmed:  {domain.BookingStatusCheckedIn, domain.BookingStatusCancelled, domain.BookingStatusNoShow},
	domain.BookingStatusCheckedIn:  {domain.BookingStatusCheckedOut, domain.BookingStatusCancelled},
	domain.BookingStatusCheckedOut: {},
	domain.BookingStatusCancelled:  {},
	domain.BookingStatusNoShow:     {},
}

// CanTransition проверяет, можно ли перевести бронь из from -> to.
func CanTransition(from, to string) bool {
	for _, allowed := range statusTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

type BookingStatusService struct {
	bookings *repo.BookingRepo
}

func NewBookingStatusService(b *repo.BookingRepo) *BookingStatusService {
	return &BookingStatusService{bookings: b}
}

func (s *BookingStatusService) Change(ctx context.Context, bookingID int64, to string, actorID int64, reason *string) error {
	b, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return err
	}
	if b.Status == to {
		return nil
	}
	if !CanTransition(b.Status, to) {
		return fmt.Errorf("%w: %s -> %s", domain.ErrBadStatus, b.Status, to)
	}
	if err := s.bookings.UpdateStatus(ctx, bookingID, to); err != nil {
		return err
	}
	from := b.Status
	return s.bookings.InsertHistory(ctx, domain.BookingStatusEvent{
		BookingID:  bookingID,
		FromStatus: &from,
		ToStatus:   to,
		ChangedBy:  &actorID,
		Reason:     reason,
	})
}
