package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserSegment struct {
	UserID    uuid.UUID
	SegmentID int64
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type ActiveSegment struct {
	Slug      string
	ExpiresAt *time.Time
}

type ExpiredMembership struct {
	UserID uuid.UUID
	Slug   string
}
