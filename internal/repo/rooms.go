package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type RoomKindRepo struct{ db *sqlx.DB }

func NewRoomKindRepo(db *sqlx.DB) *RoomKindRepo { return &RoomKindRepo{db: db} }

func (r *RoomKindRepo) Upsert(ctx context.Context, k domain.RoomKind) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO room_kinds(property_id, slug, title, description, base_rate, capacity, area, beds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (property_id, slug) DO UPDATE
		SET title       = EXCLUDED.title,
		    description = EXCLUDED.description,
		    base_rate   = EXCLUDED.base_rate,
		    capacity    = EXCLUDED.capacity,
		    area        = EXCLUDED.area,
		    beds        = EXCLUDED.beds
		RETURNING id
	`, k.PropertyID, k.Slug, k.Title, k.Description, k.BaseRate, k.Capacity, k.Area, k.Beds).Scan(&id)
	return id, err
}

func (r *RoomKindRepo) GetByPropertySlug(ctx context.Context, propSlug, kindSlug string) (*domain.RoomKind, error) {
	var k domain.RoomKind
	err := r.db.QueryRowxContext(ctx, `
		SELECT k.id, k.property_id, k.slug, k.title, k.description,
		       k.base_rate, k.capacity, k.area, k.beds
		FROM room_kinds k
		JOIN properties p ON p.id = k.property_id
		WHERE p.slug = $1 AND k.slug = $2
	`, propSlug, kindSlug).Scan(
		&k.ID, &k.PropertyID, &k.Slug, &k.Title, &k.Description,
		&k.BaseRate, &k.Capacity, &k.Area, &k.Beds,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &k, err
}

func (r *RoomKindRepo) ListByProperty(ctx context.Context, propertyID int64) ([]domain.RoomKind, error) {
	var out []domain.RoomKind
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, property_id, slug, title, description,
		       base_rate, capacity, area, beds
		FROM room_kinds WHERE property_id = $1 ORDER BY id
	`, propertyID)
	return out, err
}

func (r *RoomKindRepo) GetByID(ctx context.Context, id int64) (*domain.RoomKind, error) {
	var k domain.RoomKind
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, property_id, slug, title, description,
		       base_rate, capacity, area, beds
		FROM room_kinds WHERE id = $1
	`, id).Scan(&k.ID, &k.PropertyID, &k.Slug, &k.Title, &k.Description, &k.BaseRate, &k.Capacity, &k.Area, &k.Beds)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &k, err
}

// =========================
// Rooms
// =========================

type RoomRepo struct{ db *sqlx.DB }

func NewRoomRepo(db *sqlx.DB) *RoomRepo { return &RoomRepo{db: db} }

func (r *RoomRepo) Upsert(ctx context.Context, room domain.Room) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO rooms(property_id, kind_id, label, short_label, floor, side, area, orientation, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (property_id, short_label) DO UPDATE
		SET kind_id     = EXCLUDED.kind_id,
		    label       = EXCLUDED.label,
		    floor       = EXCLUDED.floor,
		    side        = EXCLUDED.side,
		    area        = EXCLUDED.area,
		    orientation = EXCLUDED.orientation,
		    is_active   = EXCLUDED.is_active
		RETURNING id
	`, room.PropertyID, room.KindID, room.Label, room.ShortLabel, room.Floor,
		room.Side, room.Area, room.Orientation, room.IsActive).Scan(&id)
	return id, err
}

func (r *RoomRepo) GetByID(ctx context.Context, id int64) (*domain.Room, error) {
	var room domain.Room
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, property_id, kind_id, label, short_label, floor, side, area, orientation, is_active, created_at
		FROM rooms WHERE id = $1
	`, id).Scan(&room.ID, &room.PropertyID, &room.KindID, &room.Label, &room.ShortLabel,
		&room.Floor, &room.Side, &room.Area, &room.Orientation, &room.IsActive, &room.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &room, err
}

func (r *RoomRepo) List(ctx context.Context, propertyID *int64, kindID *int64, floor *int) ([]domain.Room, error) {
	q := `SELECT id, property_id, kind_id, label, short_label, floor, side, area, orientation, is_active, created_at
	      FROM rooms WHERE is_active = true`
	args := []any{}
	idx := 1
	if propertyID != nil {
		q += " AND property_id = $" + itoa(idx)
		args = append(args, *propertyID)
		idx++
	}
	if kindID != nil {
		q += " AND kind_id = $" + itoa(idx)
		args = append(args, *kindID)
		idx++
	}
	if floor != nil {
		q += " AND floor = $" + itoa(idx)
		args = append(args, *floor)
		idx++
	}
	q += " ORDER BY property_id, floor, short_label::int, short_label"

	var out []domain.Room
	err := r.db.SelectContext(ctx, &out, q, args...)
	return out, err
}

func (r *RoomRepo) Update(ctx context.Context, id int64, label, shortLabel *string, floor *int, side, orientation **string, isActive *bool, area *float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rooms
		SET label       = COALESCE($2, label),
		    short_label = COALESCE($3, short_label),
		    floor       = COALESCE($4, floor),
		    side        = COALESCE($5, side),
		    area        = COALESCE($6, area),
		    orientation = COALESCE($7, orientation),
		    is_active   = COALESCE($8, is_active)
		WHERE id = $1
	`, id, label, shortLabel, floor, side, area, orientation, isActive)
	return err
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	buf := make([]byte, 0, 4)
	for i > 0 {
		buf = append([]byte{digits[i%10]}, buf...)
		i /= 10
	}
	return string(buf)
}
