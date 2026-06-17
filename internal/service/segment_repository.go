package service

import (
	"context"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type SegmentRepository interface {
	Create(ctx context.Context, slug string, autoPercent *int) (domain.Segment, error)
	GetBySlug(ctx context.Context, slug string) (domain.Segment, error)
	List(ctx context.Context) ([]domain.Segment, error)
	SoftDelete(ctx context.Context, segmentID int64) error
	ListMemberUserIDs(ctx context.Context, segmentID int64) ([]int64, error)
	DeleteMembershipsBySegment(ctx context.Context, segmentID int64) error
}
