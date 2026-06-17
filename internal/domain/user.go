package domain

import (
	"fmt"
	"time"
)

type User struct {
	ID        int64
	CreatedAt time.Time
}

var ErrUserNotFound = fmt.Errorf("user not found: %w", ErrNotFound)
