package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type SegmentRepository interface {
	Create(ctx context.Context, slug string, autoPercent *int) (domain.Segment, error)
	GetBySlug(ctx context.Context, slug string) (domain.Segment, error)
	List(ctx context.Context) ([]domain.Segment, error)
	SoftDelete(ctx context.Context, segmentID int64) error
	ListMemberUserIDs(ctx context.Context, segmentID int64) ([]uuid.UUID, error)
	DeleteMembershipsBySegment(ctx context.Context, segmentID int64) error
	ListBySlugs(ctx context.Context, slugs []string) ([]domain.Segment, error)
	ListPercentSegments(ctx context.Context) ([]domain.Segment, error)
}
