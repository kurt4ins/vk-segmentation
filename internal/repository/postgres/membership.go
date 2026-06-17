package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type MembershipRepo struct {
	base
}

func NewMembershipRepo(pool *pgxpool.Pool) *MembershipRepo {
	return &MembershipRepo{base{pool: pool}}
}

func (r *MembershipRepo) BatchInsert(ctx context.Context, userID int64, segmentIDs []int64, expiresAt *time.Time) ([]int64, error) {
	if len(segmentIDs) == 0 {
		return nil, nil
	}

	const q = `
		INSERT INTO user_segments (user_id, segment_id, expires_at)
		SELECT $1, sid, $3
		FROM unnest($2::bigint[]) AS sid
		ON CONFLICT (user_id, segment_id) DO NOTHING
		RETURNING segment_id`

	return r.queryIDs(ctx, q, userID, segmentIDs, expiresAt)
}

func (r *MembershipRepo) BatchDelete(ctx context.Context, userID int64, segmentIDs []int64) ([]int64, error) {
	if len(segmentIDs) == 0 {
		return nil, nil
	}

	const q = `
		DELETE FROM user_segments
		WHERE user_id = $1 AND segment_id = ANY($2::bigint[])
		RETURNING segment_id`

	return r.queryIDs(ctx, q, userID, segmentIDs)
}

func (r *MembershipRepo) ListActive(ctx context.Context, userID int64) ([]domain.ActiveSegment, error) {
	const q = `
		SELECT s.slug, us.expires_at
		FROM user_segments us
		JOIN segments s ON s.id = us.segment_id
		WHERE us.user_id = $1
		  AND s.deleted_at IS NULL
		  AND (us.expires_at IS NULL OR us.expires_at > now())
		ORDER BY s.slug`

	rows, err := r.querier(ctx).Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list active segments: %w", err)
	}
	defer rows.Close()

	active := make([]domain.ActiveSegment, 0)
	for rows.Next() {
		var a domain.ActiveSegment
		if err := rows.Scan(&a.Slug, &a.ExpiresAt); err != nil {
			return nil, fmt.Errorf("postgres: scan active segment: %w", err)
		}
		active = append(active, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list active segments rows: %w", err)
	}
	return active, nil
}

func (r *MembershipRepo) queryIDs(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := r.querier(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: membership mutation: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: scan segment id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: membership mutation rows: %w", err)
	}
	return ids, nil
}
