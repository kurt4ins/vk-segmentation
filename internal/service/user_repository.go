package service

import (
	"context"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context) (domain.User, error)
	Exists(ctx context.Context, userID int64) (bool, error)
}
