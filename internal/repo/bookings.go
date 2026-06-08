package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type BookingRepo struct{ db *sqlx.DB }

func NewBookingRepo(db *sqlx.DB) *BookingRepo { return &BookingRepo{db: db} }

func (r *BookingRepo) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	var b domain.Booking
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, code, room_id, guest_id, check_in, check_out,
		       check_in_time::text, check_out_time::text,
		       adults, status, source,
		       total_amount, prepayment, notes, created_by, created_at, updated_at
		FROM bookings WHERE id = $1
	`, id).Scan(&b.ID, &b.Code, &b.RoomID, &b.GuestID, &b.CheckIn, &b.CheckOut,
		&b.CheckInTime, &b.CheckOutTime,
		&b.Adults, &b.Status, &b.Source,
		&b.TotalAmount, &b.Prepayment, &b.Notes, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &b, err
}

func (r *BookingRepo) GetByCode(ctx context.Context, code string) (*domain.Booking, error) {
	var b domain.Booking
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, code, room_id, guest_id, check_in, check_out,
		       check_in_time::text, check_out_time::text,
		       adults, status, source,
		       total_amount, prepayment, notes, created_by, created_at, updated_at
		FROM bookings WHERE code = $1
	`, code).Scan(&b.ID, &b.Code, &b.RoomID, &b.GuestID, &b.CheckIn, &b.CheckOut,
		&b.CheckInTime, &b.CheckOutTime,
		&b.Adults, &b.Status, &b.Source,
		&b.TotalAmount, &b.Prepayment, &b.Notes, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &b, err
}

func (r *BookingRepo) List(ctx context.Context, from, to *time.Time, propertyID, kindID *int64, status, q *string) ([]domain.Booking, error) {
	query := `SELECT b.id, b.code, b.room_id, b.guest_id, b.check_in, b.check_out,
	          b.check_in_time::text, b.check_out_time::text,
	          b.adults, b.status, b.source,
	          b.total_amount, b.prepayment, b.notes, b.created_by, b.created_at, b.updated_at
	          FROM bookings b
	          JOIN rooms r ON r.id = b.room_id`
	args := []any{}
	where := []string{}
	idx := 1
	if from != nil {
		where = append(where, "b.check_out > $"+itoa(idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		where = append(where, "b.check_in < $"+itoa(idx))
		args = append(args, *to)
		idx++
	}
	if propertyID != nil {
		where = append(where, "r.property_id = $"+itoa(idx))
		args = append(args, *propertyID)
		idx++
	}
	if kindID != nil {
		where = append(where, "r.kind_id = $"+itoa(idx))
		args = append(args, *kindID)
		idx++
	}
	if status != nil {
		where = append(where, "b.status = $"+itoa(idx))
		args = append(args, *status)
		idx++
	}
	if q != nil && *q != "" {
		where = append(where, "(b.code ILIKE $"+itoa(idx)+" OR b.notes ILIKE $"+itoa(idx)+")")
		args = append(args, "%"+*q+"%")
		idx++
	}
	if len(where) > 0 {
		query += " WHERE " + joinAnd(where)
	}
	query += " ORDER BY b.check_in DESC, b.id DESC LIMIT 500"

	var out []domain.Booking
	err := r.db.SelectContext(ctx, &out, query, args...)
	return out, err
}

// OverlappingActive возвращает активные брони на комнату в диапазоне.
// Используется для шахматки.
func (r *BookingRepo) OverlappingActive(ctx context.Context, roomIDs []int64, from, to time.Time) ([]domain.Booking, error) {
	if len(roomIDs) == 0 {
		return nil, nil
	}
	q, args, err := sqlx.In(`
		SELECT id, code, room_id, guest_id, check_in, check_out,
		       check_in_time::text, check_out_time::text,
		       adults, status, source,
		       total_amount, prepayment, notes, created_by, created_at, updated_at
		FROM bookings
		WHERE room_id IN (?) AND status IN ('new','confirmed','checked_in')
		  AND check_out > ? AND check_in < ?
		ORDER BY check_in
	`, roomIDs, from, to)
	if err != nil {
		return nil, err
	}
	q = r.db.Rebind(q)
	var out []domain.Booking
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *BookingRepo) Insert(ctx context.Context, b domain.Booking) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO bookings(code, room_id, guest_id, check_in, check_out,
		                     check_in_time, check_out_time,
		                     adults, status, source,
		                     total_amount, prepayment, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`, b.Code, b.RoomID, b.GuestID, b.CheckIn, b.CheckOut,
		b.CheckInTime, b.CheckOutTime,
		b.Adults, b.Status, b.Source,
		b.TotalAmount, b.Prepayment, b.Notes, b.CreatedBy).Scan(&id)
	return id, err
}

func (r *BookingRepo) UpdateFields(ctx context.Context, id int64, checkIn, checkOut *time.Time, checkInTime, checkOutTime *string, adults *int, guestID *int64, total *float64, prepayment *float64, notes *string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE bookings
		SET check_in       = COALESCE($2, check_in),
		    check_out      = COALESCE($3, check_out),
		    check_in_time  = COALESCE($4, check_in_time),
		    check_out_time = COALESCE($5, check_out_time),
		    adults         = COALESCE($6, adults),
		    guest_id       = COALESCE($7, guest_id),
		    total_amount   = COALESCE($8, total_amount),
		    prepayment     = COALESCE($9, prepayment),
		    notes          = COALESCE($10, notes),
		    updated_at     = now()
		WHERE id = $1
	`, id, checkIn, checkOut, checkInTime, checkOutTime, adults, guestID, total, prepayment, notes)
	return err
}

func (r *BookingRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE bookings SET status = $2, updated_at = now() WHERE id = $1
	`, id, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BookingRepo) SoftCancel(ctx context.Context, id int64) error {
	return r.UpdateStatus(ctx, id, domain.BookingStatusCancelled)
}

func (r *BookingRepo) History(ctx context.Context, bookingID int64) ([]domain.BookingStatusEvent, error) {
	var out []domain.BookingStatusEvent
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, booking_id, from_status, to_status, changed_by, reason, changed_at
		FROM booking_status_history WHERE booking_id = $1 ORDER BY changed_at, id
	`, bookingID)
	return out, err
}

func (r *BookingRepo) InsertHistory(ctx context.Context, ev domain.BookingStatusEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO booking_status_history(booking_id, from_status, to_status, changed_by, reason)
		VALUES ($1, $2, $3, $4, $5)
	`, ev.BookingID, ev.FromStatus, ev.ToStatus, ev.ChangedBy, ev.Reason)
	return err
}

// NextCodeSeq возвращает следующий seq в году, для генерации BK-YYYY-XXXX.
func (r *BookingRepo) NextCodeSeq(ctx context.Context, year int) (int, error) {
	var maxSeq sql.NullInt64
	prefix := "BK-"
	pattern := prefix + itoa(year) + "-%"
	err := r.db.QueryRowxContext(ctx, `
		SELECT MAX( CAST( SPLIT_PART(code, '-', 3) AS INT) ) FROM bookings
		WHERE code LIKE $1
	`, pattern).Scan(&maxSeq)
	if err != nil {
		return 0, err
	}
	if !maxSeq.Valid {
		return 1, nil
	}
	return int(maxSeq.Int64) + 1, nil
}

func joinAnd(s []string) string {
	out := s[0]
	for i := 1; i < len(s); i++ {
		out += " AND " + s[i]
	}
	return out
}
