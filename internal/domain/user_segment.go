package domain

import "time"

type UserSegment struct {
	UserID    int64
	SegmentID int64
	ExpiresAt *time.Time
	CreatedAt time.Time
}
