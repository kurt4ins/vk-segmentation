package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type HistoryReader interface {
	ListByUserAndRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.HistoryRecord, error)
}

type HistoryService struct {
	history    HistoryReader
	reportsDir string
}

func NewHistoryService(history HistoryReader, reportsDir string) *HistoryService {
	return &HistoryService{history: history, reportsDir: reportsDir}
}

func (s *HistoryService) Report(ctx context.Context, userID uuid.UUID, from, to time.Time) (string, error) {
	records, err := s.history.ListByUserAndRange(ctx, userID, from, to)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(s.reportsDir, 0o755); err != nil {
		return "", fmt.Errorf("history: create reports dir: %w", err)
	}

	filename := fmt.Sprintf("history_%s_%d.csv", userID, time.Now().UnixNano())
	path := filepath.Join(s.reportsDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("history: create report file: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	w.Comma = ';'

	if err := w.Write([]string{"user_id", "slug", "operation", "datetime"}); err != nil {
		return "", fmt.Errorf("history: write report header: %w", err)
	}
	for _, rec := range records {
		row := []string{
			rec.UserID.String(),
			rec.Slug,
			string(rec.Operation),
			rec.CreatedAt.Format(time.RFC3339),
		}
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("history: write report row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("history: flush report: %w", err)
	}
	return filename, nil
}
