package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, email, hash, role, fullName string) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO users(email, password_hash, role, full_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    role          = EXCLUDED.role,
		    full_name     = EXCLUDED.full_name,
		    is_active     = true,
		    updated_at    = now()
		RETURNING id
	`, email, hash, role, fullName).Scan(&id)
	return id, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, string, error) {
	var u domain.User
	var hash string
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, email, password_hash, role, full_name, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &hash, &u.Role, &u.FullName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, email, role, full_name, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	var out []domain.User
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, email, role, full_name, is_active, created_at, updated_at
		FROM users ORDER BY id
	`)
	return out, err
}

func (r *UserRepo) Update(ctx context.Context, id int64, role, fullName *string, isActive *bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET role      = COALESCE($2, role),
		    full_name = COALESCE($3, full_name),
		    is_active = COALESCE($4, is_active),
		    updated_at = now()
		WHERE id = $1
	`, id, role, fullName, isActive)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1
	`, id, hash)
	return err
}

// touchUpdatedAt используется в других репо при необходимости.
func touchUpdatedAt(_ time.Time) {}
