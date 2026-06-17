package domain

import "time"

type Operation string

const (
	OpAdd    Operation = "add"
	OpRemove Operation = "remove"
)

type HistoryRecord struct {
	ID        int64
	UserID    int64
	Slug      string
	Operation Operation
	CreatedAt time.Time
}
