package domain

import (
	"time"

	"github.com/google/uuid"
)

type Operation string

const (
	OpAdd    Operation = "add"
	OpRemove Operation = "remove"
)

type HistoryRecord struct {
	ID        int64
	UserID    uuid.UUID
	Slug      string
	Operation Operation
	CreatedAt time.Time
}
