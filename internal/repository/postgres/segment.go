package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type SegmentRepo struct {
	base
}

func NewSegmentRepo(pool *pgxpool.Pool) *SegmentRepo {
	return &SegmentRepo{base{pool: pool}}
}

func (r *SegmentRepo) Create(ctx context.Context, slug string, autoPercent *int) (domain.Segment, error) {
	status := domain.StatusApplied
	if autoPercent != nil {
		status = domain.StatusPending
	}

	const q = `
		INSERT INTO segments (slug, auto_assign_percent, status)
		VALUES ($1, $2, $3)
		RETURNING id, slug, auto_assign_percent, status, created_at, deleted_at`

	var seg domain.Segment
	err := r.querier(ctx).QueryRow(ctx, q, slug, autoPercent, string(status)).
		Scan(&seg.ID, &seg.Slug, &seg.AutoAssignPercent, &seg.Status, &seg.CreatedAt, &seg.DeletedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Segment{}, domain.ErrSegmentAlreadyExists
		}
		return domain.Segment{}, fmt.Errorf("postgres: create segment: %w", err)
	}
	return seg, nil
}

func (r *SegmentRepo) GetBySlug(ctx context.Context, slug string) (domain.Segment, error) {
	const q = `
		SELECT id, slug, auto_assign_percent, status, created_at, deleted_at
		FROM segments
		WHERE slug = $1 AND deleted_at IS NULL`

	var seg domain.Segment
	err := r.querier(ctx).QueryRow(ctx, q, slug).
		Scan(&seg.ID, &seg.Slug, &seg.AutoAssignPercent, &seg.Status, &seg.CreatedAt, &seg.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Segment{}, domain.ErrSegmentNotFound
		}
		return domain.Segment{}, fmt.Errorf("postgres: get segment by slug: %w", err)
	}
	return seg, nil
}

func (r *SegmentRepo) List(ctx context.Context) ([]domain.Segment, error) {
	const q = `
		SELECT id, slug, auto_assign_percent, status, created_at, deleted_at
		FROM segments
		WHERE deleted_at IS NULL
		ORDER BY id`

	rows, err := r.querier(ctx).Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: list segments: %w", err)
	}
	return scanSegments(rows)
}

func (r *SegmentRepo) ListBySlugs(ctx context.Context, slugs []string) ([]domain.Segment, error) {
	if len(slugs) == 0 {
		return nil, nil
	}

	const q = `
		SELECT id, slug, auto_assign_percent, status, created_at, deleted_at
		FROM segments
		WHERE slug = ANY($1::text[]) AND deleted_at IS NULL`

	rows, err := r.querier(ctx).Query(ctx, q, slugs)
	if err != nil {
		return nil, fmt.Errorf("postgres: list segments by slugs: %w", err)
	}
	return scanSegments(rows)
}

func (r *SegmentRepo) ListPercentSegments(ctx context.Context) ([]domain.Segment, error) {
	const q = `
		SELECT id, slug, auto_assign_percent, status, created_at, deleted_at
		FROM segments
		WHERE auto_assign_percent IS NOT NULL AND deleted_at IS NULL
		ORDER BY id`

	rows, err := r.querier(ctx).Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: list percent segments: %w", err)
	}
	return scanSegments(rows)
}

func scanSegments(rows pgx.Rows) ([]domain.Segment, error) {
	defer rows.Close()

	segments := make([]domain.Segment, 0)
	for rows.Next() {
		var seg domain.Segment
		if err := rows.Scan(&seg.ID, &seg.Slug, &seg.AutoAssignPercent, &seg.Status, &seg.CreatedAt, &seg.DeletedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan segment: %w", err)
		}
		segments = append(segments, seg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: scan segments rows: %w", err)
	}
	return segments, nil
}

func (r *SegmentRepo) SoftDelete(ctx context.Context, segmentID int64) error {
	const q = `UPDATE segments SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`
	if _, err := r.querier(ctx).Exec(ctx, q, segmentID); err != nil {
		return fmt.Errorf("postgres: soft delete segment: %w", err)
	}
	return nil
}

func (r *SegmentRepo) MarkApplied(ctx context.Context, segmentID int64) error {
	const q = `UPDATE segments SET status = 'applied' WHERE id = $1`
	if _, err := r.querier(ctx).Exec(ctx, q, segmentID); err != nil {
		return fmt.Errorf("postgres: mark segment applied: %w", err)
	}
	return nil
}

func (r *SegmentRepo) ListMemberUserIDs(ctx context.Context, segmentID int64) ([]uuid.UUID, error) {
	const q = `SELECT user_id FROM user_segments WHERE segment_id = $1`

	rows, err := r.querier(ctx).Query(ctx, q, segmentID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list member user ids: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: scan member user id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list member user ids rows: %w", err)
	}
	return ids, nil
}

func (r *SegmentRepo) DeleteMembershipsBySegment(ctx context.Context, segmentID int64) error {
	const q = `DELETE FROM user_segments WHERE segment_id = $1`
	if _, err := r.querier(ctx).Exec(ctx, q, segmentID); err != nil {
		return fmt.Errorf("postgres: delete memberships by segment: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
