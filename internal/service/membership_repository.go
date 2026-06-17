package service

import (
	"context"
	"time"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type MembershipRepository interface {
	BatchInsert(ctx context.Context, userID int64, segmentIDs []int64, expiresAt *time.Time) ([]int64, error)
	BatchDelete(ctx context.Context, userID int64, segmentIDs []int64) ([]int64, error)
	ListActive(ctx context.Context, userID int64) ([]domain.ActiveSegment, error)
}
