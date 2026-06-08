package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type RateRepo struct{ db *sqlx.DB }

func NewRateRepo(db *sqlx.DB) *RateRepo { return &RateRepo{db: db} }

func (r *RateRepo) ListByKind(ctx context.Context, kindID int64) ([]domain.Rate, error) {
	var out []domain.Rate
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, kind_id, date_from, date_to, weekday_rate, weekend_rate
		FROM rates WHERE kind_id = $1
		ORDER BY date_from
	`, kindID)
	return out, err
}

func (r *RateRepo) GetByID(ctx context.Context, id int64) (*domain.Rate, error) {
	var x domain.Rate
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, kind_id, date_from, date_to, weekday_rate, weekend_rate
		FROM rates WHERE id = $1
	`, id).Scan(&x.ID, &x.KindID, &x.DateFrom, &x.DateTo, &x.WeekdayRate, &x.WeekendRate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &x, err
}

func (r *RateRepo) Create(ctx context.Context, x domain.Rate) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO rates(kind_id, date_from, date_to, weekday_rate, weekend_rate)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, x.KindID, x.DateFrom, x.DateTo, x.WeekdayRate, x.WeekendRate).Scan(&id)
	return id, err
}

func (r *RateRepo) Update(ctx context.Context, x domain.Rate) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rates SET date_from = $2, date_to = $3, weekday_rate = $4, weekend_rate = $5
		WHERE id = $1
	`, x.ID, x.DateFrom, x.DateTo, x.WeekdayRate, x.WeekendRate)
	return err
}

func (r *RateRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM rates WHERE id = $1`, id)
	return err
}

// RateForDate возвращает ставку для kind на конкретную дату.
// Если ничего не нашли — fallback на base_rate, переданный аргументом.
func (r *RateRepo) RateForDate(ctx context.Context, kindID int64, date sql.NullTime, baseRate float64) (float64, error) {
	var rate float64
	err := r.db.QueryRowxContext(ctx, `
		SELECT COALESCE(
			(CASE WHEN EXTRACT(DOW FROM $2::date) IN (0, 6)
			      THEN weekend_rate ELSE weekday_rate END),
			$3
		) FROM rates
		WHERE kind_id = $1 AND $2::date BETWEEN date_from AND date_to
		ORDER BY date_from DESC LIMIT 1
	`, kindID, date, baseRate).Scan(&rate)
	if errors.Is(err, sql.ErrNoRows) {
		return baseRate, nil
	}
	return rate, err
}
