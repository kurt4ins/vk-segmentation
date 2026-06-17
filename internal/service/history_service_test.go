package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/service"
	"github.com/kurt4ins/vk-segmentation/internal/service/mocks"
)

func TestHistoryService_Report_WritesCSV(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockHistoryReader(ctrl)
	dir := t.TempDir()

	uid := uuid.New()
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	reader.EXPECT().ListByUserAndRange(gomock.Any(), uid, gomock.Any(), gomock.Any()).Return(
		[]domain.HistoryRecord{
			{UserID: uid, Slug: "MAIL_GPT", Operation: domain.OpAdd, CreatedAt: ts},
			{UserID: uid, Slug: "MAIL_GPT", Operation: domain.OpRemove, CreatedAt: ts},
		}, nil)

	svc := service.NewHistoryService(reader, dir)
	filename, err := svc.Report(context.Background(), uid, time.Time{}, time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, filename)

	data, err := os.ReadFile(filepath.Join(dir, filename))
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "user_id;slug;operation;datetime")
	require.Contains(t, content, uid.String()+";MAIL_GPT;add;2026-06-01T12:00:00Z")
	require.Contains(t, content, uid.String()+";MAIL_GPT;remove;2026-06-01T12:00:00Z")
}

func TestHistoryService_Report_EmptyHistoryHeaderOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockHistoryReader(ctrl)
	dir := t.TempDir()

	uid := uuid.New()
	reader.EXPECT().ListByUserAndRange(gomock.Any(), uid, gomock.Any(), gomock.Any()).Return(nil, nil)

	svc := service.NewHistoryService(reader, dir)
	filename, err := svc.Report(context.Background(), uid, time.Time{}, time.Now())
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, filename))
	require.NoError(t, err)
	require.Equal(t, "user_id;slug;operation;datetime\n", string(data))
}
