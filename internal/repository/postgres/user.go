package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserRepo struct {
	base
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{base{pool: pool}}
}

func (r *UserRepo) Create(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const q = `INSERT INTO users (user_id) VALUES ($1) RETURNING user_id, created_at`

	var u domain.User
	if err := r.querier(ctx).QueryRow(ctx, q, id).Scan(&u.ID, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("postgres: create user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`

	var exists bool
	if err := r.querier(ctx).QueryRow(ctx, q, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("postgres: user exists: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	const q = `SELECT count(*) FROM users`

	var n int64
	if err := r.querier(ctx).QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count users: %w", err)
	}
	return n, nil
}

func (r *UserRepo) ListNonMembers(ctx context.Context, segmentID int64, limit int) ([]uuid.UUID, error) {
	const q = `
		SELECT u.user_id
		FROM users u
		WHERE NOT EXISTS (
			SELECT 1 FROM user_segments us
			WHERE us.user_id = u.user_id AND us.segment_id = $1
		)
		ORDER BY random()
		LIMIT $2`

	rows, err := r.querier(ctx).Query(ctx, q, segmentID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: list non-members: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: scan non-member id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list non-members rows: %w", err)
	}
	return ids, nil
}
