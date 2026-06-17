package domain

import (
	"fmt"
	"time"
)

type SegmentStatus string

const (
	StatusPending SegmentStatus = "pending"
	StatusApplied SegmentStatus = "applied"
)

type Segment struct {
	ID                int64
	Slug              string
	AutoAssignPercent *int
	Status            SegmentStatus
	CreatedAt         time.Time
	DeletedAt         *time.Time
}

var (
	ErrSegmentNotFound      = fmt.Errorf("segment not found: %w", ErrNotFound)
	ErrSegmentAlreadyExists = fmt.Errorf("segment already exists: %w", ErrConflict)
)
