package service

import (
	"testing"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{domain.BookingStatusNew, domain.BookingStatusConfirmed, true},
		{domain.BookingStatusNew, domain.BookingStatusCheckedIn, false},
		{domain.BookingStatusNew, domain.BookingStatusCheckedOut, false},
		{domain.BookingStatusNew, domain.BookingStatusCancelled, true},

		{domain.BookingStatusConfirmed, domain.BookingStatusCheckedIn, true},
		{domain.BookingStatusConfirmed, domain.BookingStatusNoShow, true},
		{domain.BookingStatusConfirmed, domain.BookingStatusNew, false},

		{domain.BookingStatusCheckedIn, domain.BookingStatusCheckedOut, true},
		{domain.BookingStatusCheckedIn, domain.BookingStatusCancelled, true},
		{domain.BookingStatusCheckedIn, domain.BookingStatusNew, false},

		{domain.BookingStatusCheckedOut, domain.BookingStatusCheckedIn, false},
		{domain.BookingStatusCheckedOut, domain.BookingStatusNew, false},

		{domain.BookingStatusCancelled, domain.BookingStatusConfirmed, false},
		{domain.BookingStatusNoShow, domain.BookingStatusConfirmed, false},
	}
	for _, c := range cases {
		got := CanTransition(c.from, c.to)
		if got != c.ok {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestCanTransitionSelf(t *testing.T) {
	// Self-transition недопустим в CanTransition; no-op обрабатывается
	// выше в Change() через проверку b.Status == to.
	for s := range statusTransitions {
		if CanTransition(s, s) {
			t.Errorf("expected self-transition for %s to be denied by CanTransition", s)
		}
	}
}
