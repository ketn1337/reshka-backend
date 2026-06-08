package repo

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type GuestRepo struct{ db *sqlx.DB }

func NewGuestRepo(db *sqlx.DB) *GuestRepo { return &GuestRepo{db: db} }

func (r *GuestRepo) Create(ctx context.Context, g domain.Guest) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO guests(full_name, phone, email, doc_type, doc_number, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, g.FullName, g.Phone, g.Email, g.DocType, g.DocNumber, g.Notes).Scan(&id)
	return id, err
}

func (r *GuestRepo) GetByID(ctx context.Context, id int64) (*domain.Guest, error) {
	var g domain.Guest
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, full_name, phone, email, doc_type, doc_number, notes, created_at
		FROM guests WHERE id = $1
	`, id).Scan(&g.ID, &g.FullName, &g.Phone, &g.Email, &g.DocType, &g.DocNumber, &g.Notes, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &g, err
}

func (r *GuestRepo) Update(ctx context.Context, id int64, g domain.Guest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE guests
		SET full_name  = $2,
		    phone      = $3,
		    email      = $4,
		    doc_type   = $5,
		    doc_number = $6,
		    notes      = $7
		WHERE id = $1
	`, id, g.FullName, g.Phone, g.Email, g.DocType, g.DocNumber, g.Notes)
	return err
}

func (r *GuestRepo) ListByIDs(ctx context.Context, ids []int64) ([]domain.Guest, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q, args, err := sqlx.In(`
		SELECT id, full_name, phone, email, doc_type, doc_number, notes, created_at
		FROM guests WHERE id IN (?)
	`, ids)
	if err != nil {
		return nil, err
	}
	q = r.db.Rebind(q)
	var out []domain.Guest
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *GuestRepo) Search(ctx context.Context, q string, limit int) ([]domain.Guest, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	pattern := strings.ToLower(strings.TrimSpace(q)) + "%"
	var out []domain.Guest
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, full_name, phone, email, doc_type, doc_number, notes, created_at
		FROM guests
		WHERE LOWER(full_name) LIKE $1
		   OR phone LIKE $2
		   OR LOWER(email) LIKE $1
		ORDER BY full_name
		LIMIT $3
	`, pattern, q+"%", limit)
	return out, err
}
