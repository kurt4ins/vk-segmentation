package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

	userIDs := make([]uuid.UUID, len(records))
	slugs := make([]string, len(records))
	ops := make([]string, len(records))
	for i, rec := range records {
		userIDs[i] = rec.UserID
		slugs[i] = rec.Slug
		ops[i] = string(rec.Operation)
	}

	const q = `
		INSERT INTO segment_history (user_id, slug, operation)
		SELECT * FROM unnest($1::uuid[], $2::text[], $3::text[])`

	if _, err := r.querier(ctx).Exec(ctx, q, userIDs, slugs, ops); err != nil {
		return fmt.Errorf("postgres: batch insert history: %w", err)
	}
	return nil
}

func (r *HistoryRepo) ListByUserAndRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.HistoryRecord, error) {
	const q = `
		SELECT id, user_id, slug, operation, created_at
		FROM segment_history
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at, id`

	rows, err := r.querier(ctx).Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("postgres: list history: %w", err)
	}
	defer rows.Close()

	records := make([]domain.HistoryRecord, 0)
	for rows.Next() {
		var rec domain.HistoryRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Slug, &rec.Operation, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan history record: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list history rows: %w", err)
	}
	return records, nil
}
