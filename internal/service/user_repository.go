package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, id uuid.UUID) (domain.User, error)
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)
}
