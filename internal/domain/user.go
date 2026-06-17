package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

var ErrUserNotFound = fmt.Errorf("user not found: %w", ErrNotFound)
