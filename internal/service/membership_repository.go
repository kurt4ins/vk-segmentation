package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type MembershipRepository interface {
	BatchInsert(ctx context.Context, userID uuid.UUID, segmentIDs []int64, expiresAt *time.Time) ([]int64, error)
	BatchDelete(ctx context.Context, userID uuid.UUID, segmentIDs []int64) ([]int64, error)
	ListActive(ctx context.Context, userID uuid.UUID) ([]domain.ActiveSegment, error)
	BatchAddUsers(ctx context.Context, segmentID int64, userIDs []uuid.UUID) ([]uuid.UUID, error)
	DeleteExpired(ctx context.Context) ([]domain.ExpiredMembership, error)
}
