package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type PropertyRepo struct{ db *sqlx.DB }

func NewPropertyRepo(db *sqlx.DB) *PropertyRepo { return &PropertyRepo{db: db} }

func (r *PropertyRepo) Upsert(ctx context.Context, p domain.Property) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO properties(slug, title, short_title, address, description, accent)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO UPDATE
		SET title       = EXCLUDED.title,
		    short_title = EXCLUDED.short_title,
		    address     = EXCLUDED.address,
		    description = EXCLUDED.description,
		    accent      = EXCLUDED.accent
		RETURNING id
	`, p.Slug, p.Title, p.ShortTitle, p.Address, p.Description, p.Accent).Scan(&id)
	return id, err
}

func (r *PropertyRepo) List(ctx context.Context) ([]domain.Property, error) {
	var out []domain.Property
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, slug, title, short_title, address, description, accent, created_at
		FROM properties ORDER BY id
	`)
	return out, err
}

func (r *PropertyRepo) GetBySlug(ctx context.Context, slug string) (*domain.Property, error) {
	var p domain.Property
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, slug, title, short_title, address, description, accent, created_at
		FROM properties WHERE slug = $1
	`, slug).Scan(&p.ID, &p.Slug, &p.Title, &p.ShortTitle, &p.Address, &p.Description, &p.Accent, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *PropertyRepo) GetByID(ctx context.Context, id int64) (*domain.Property, error) {
	var p domain.Property
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, slug, title, short_title, address, description, accent, created_at
		FROM properties WHERE id = $1
	`, id).Scan(&p.ID, &p.Slug, &p.Title, &p.ShortTitle, &p.Address, &p.Description, &p.Accent, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}
