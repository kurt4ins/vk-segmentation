package service

import (
	"context"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type HistoryRepository interface {
	BatchInsert(ctx context.Context, records []domain.HistoryRecord) error
}
