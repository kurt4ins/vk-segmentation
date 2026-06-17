package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type HistoryRepo struct {
	base
}

func NewHistoryRepo(pool *pgxpool.Pool) *HistoryRepo {
	return &HistoryRepo{base{pool: pool}}
}

func (r *HistoryRepo) BatchInsert(ctx context.Context, records []domain.HistoryRecord) error {
	if len(records) == 0 {
		return nil
	}

	userIDs := make([]int64, len(records))
	slugs := make([]string, len(records))
	ops := make([]string, len(records))
	for i, rec := range records {
		userIDs[i] = rec.UserID
		slugs[i] = rec.Slug
		ops[i] = string(rec.Operation)
	}

	const q = `
		INSERT INTO segment_history (user_id, slug, operation)
		SELECT * FROM unnest($1::bigint[], $2::text[], $3::text[])`

	if _, err := r.querier(ctx).Exec(ctx, q, userIDs, slugs, ops); err != nil {
		return fmt.Errorf("postgres: batch insert history: %w", err)
	}
	return nil
}
