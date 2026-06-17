package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserRepo struct {
	base
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{base{pool: pool}}
}

func (r *UserRepo) Create(ctx context.Context) (domain.User, error) {
	const q = `INSERT INTO users DEFAULT VALUES RETURNING user_id, created_at`

	var u domain.User
	if err := r.querier(ctx).QueryRow(ctx, q).Scan(&u.ID, &u.CreatedAt); err != nil {
		return domain.User{}, fmt.Errorf("postgres: create user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) Exists(ctx context.Context, userID int64) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`

	var exists bool
	if err := r.querier(ctx).QueryRow(ctx, q, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("postgres: user exists: %w", err)
	}
	return exists, nil
}
