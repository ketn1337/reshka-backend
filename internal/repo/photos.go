package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/ketn1337/reshka-backend/internal/domain"
)

type PhotoRepo struct{ db *sqlx.DB }

func NewPhotoRepo(db *sqlx.DB) *PhotoRepo { return &PhotoRepo{db: db} }

func (r *PhotoRepo) Insert(ctx context.Context, roomID int64, filename string, position int, isCover bool) (int64, error) {
	var id int64
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO photos(room_id, filename, position, is_cover)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id, filename) DO UPDATE
		SET position = EXCLUDED.position,
		    is_cover = EXCLUDED.is_cover
		RETURNING id
	`, roomID, filename, position, isCover).Scan(&id)
	return id, err
}

func (r *PhotoRepo) ListByRoom(ctx context.Context, roomID int64) ([]domain.Photo, error) {
	var out []domain.Photo
	err := r.db.SelectContext(ctx, &out, `
		SELECT id, room_id, filename, position, is_cover, created_at
		FROM photos WHERE room_id = $1 ORDER BY position, id
	`, roomID)
	return out, err
}

func (r *PhotoRepo) ListByRoomIDs(ctx context.Context, roomIDs []int64) (map[int64][]domain.Photo, error) {
	if len(roomIDs) == 0 {
		return map[int64][]domain.Photo{}, nil
	}
	q, args, err := sqlx.In(`
		SELECT id, room_id, filename, position, is_cover, created_at
		FROM photos WHERE room_id IN (?) ORDER BY room_id, position, id
	`, roomIDs)
	if err != nil {
		return nil, err
	}
	q = r.db.Rebind(q)
	var rows []domain.Photo
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make(map[int64][]domain.Photo, len(roomIDs))
	for _, p := range rows {
		out[p.RoomID] = append(out[p.RoomID], p)
	}
	return out, nil
}

func (r *PhotoRepo) GetByID(ctx context.Context, id int64) (*domain.Photo, error) {
	var p domain.Photo
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, room_id, filename, position, is_cover, created_at
		FROM photos WHERE id = $1
	`, id).Scan(&p.ID, &p.RoomID, &p.Filename, &p.Position, &p.IsCover, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *PhotoRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM photos WHERE id = $1`, id)
	return err
}

func (r *PhotoRepo) Reorder(ctx context.Context, roomID int64, ids []int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE photos SET position = $1 WHERE id = $2 AND room_id = $3`,
			i, id, roomID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
