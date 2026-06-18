package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/service"
	"github.com/kurt4ins/vk-segmentation/internal/service/mocks"
)

func TestSegmentService_Create_InvalidSlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := service.NewSegmentService(
		mocks.NewMockSegmentRepository(ctrl),
		mocks.NewMockHistoryRepository(ctrl),
		fakeTx{},
		mocks.NewMockRolloutEnqueuer(ctrl),
	)

	_, err := svc.Create(context.Background(), "bad slug!", nil)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestSegmentService_Create_InvalidPercent(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := service.NewSegmentService(
		mocks.NewMockSegmentRepository(ctrl),
		mocks.NewMockHistoryRepository(ctrl),
		fakeTx{},
		mocks.NewMockRolloutEnqueuer(ctrl),
	)

	_, err := svc.Create(context.Background(), "AB_TEST", intp(150))
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestSegmentService_Create_NoPercent_NoEnqueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	enq := mocks.NewMockRolloutEnqueuer(ctrl)
	svc := service.NewSegmentService(segRepo, mocks.NewMockHistoryRepository(ctrl), fakeTx{}, enq)

	want := domain.Segment{ID: 1, Slug: "MAIL_GPT", Status: domain.StatusApplied}
	segRepo.EXPECT().Create(gomock.Any(), "MAIL_GPT", nil).Return(want, nil)

	got, err := svc.Create(context.Background(), "MAIL_GPT", nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSegmentService_Create_WithPercent_Enqueues(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	enq := mocks.NewMockRolloutEnqueuer(ctrl)
	svc := service.NewSegmentService(segRepo, mocks.NewMockHistoryRepository(ctrl), fakeTx{}, enq)

	percent := intp(50)
	created := domain.Segment{ID: 2, Slug: "AB_TEST", AutoAssignPercent: percent, Status: domain.StatusPending}
	segRepo.EXPECT().Create(gomock.Any(), "AB_TEST", percent).Return(created, nil)
	enq.EXPECT().Enqueue(created)

	got, err := svc.Create(context.Background(), "AB_TEST", percent)
	require.NoError(t, err)
	require.Equal(t, created, got)
}

func TestSegmentService_Create_DuplicatePropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	svc := service.NewSegmentService(segRepo, mocks.NewMockHistoryRepository(ctrl), fakeTx{}, mocks.NewMockRolloutEnqueuer(ctrl))

	segRepo.EXPECT().Create(gomock.Any(), "MAIL_GPT", nil).Return(domain.Segment{}, domain.ErrSegmentAlreadyExists)

	_, err := svc.Create(context.Background(), "MAIL_GPT", nil)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestSegmentService_Delete_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	svc := service.NewSegmentService(segRepo, mocks.NewMockHistoryRepository(ctrl), fakeTx{}, mocks.NewMockRolloutEnqueuer(ctrl))

	segRepo.EXPECT().GetBySlug(gomock.Any(), "NOPE").Return(domain.Segment{}, domain.ErrSegmentNotFound)

	err := svc.Delete(context.Background(), "NOPE")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSegmentService_Delete_CascadeWritesRemoveHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	histRepo := mocks.NewMockHistoryRepository(ctrl)
	svc := service.NewSegmentService(segRepo, histRepo, fakeTx{}, mocks.NewMockRolloutEnqueuer(ctrl))

	seg := domain.Segment{ID: 7, Slug: "MAIL_GPT"}
	u1, u2 := uuid.New(), uuid.New()

	segRepo.EXPECT().GetBySlug(gomock.Any(), "MAIL_GPT").Return(seg, nil)
	segRepo.EXPECT().ListMemberUserIDs(gomock.Any(), int64(7)).Return([]uuid.UUID{u1, u2}, nil)
	segRepo.EXPECT().SoftDelete(gomock.Any(), int64(7)).Return(nil)
	segRepo.EXPECT().DeleteMembershipsBySegment(gomock.Any(), int64(7)).Return(nil)
	histRepo.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 2)
			for _, r := range recs {
				require.Equal(t, "MAIL_GPT", r.Slug)
				require.Equal(t, domain.OpRemove, r.Operation)
			}
			return nil
		})

	require.NoError(t, svc.Delete(context.Background(), "MAIL_GPT"))
}

func TestSegmentService_Delete_NoMembersSkipsHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	segRepo := mocks.NewMockSegmentRepository(ctrl)
	histRepo := mocks.NewMockHistoryRepository(ctrl)
	svc := service.NewSegmentService(segRepo, histRepo, fakeTx{}, mocks.NewMockRolloutEnqueuer(ctrl))

	seg := domain.Segment{ID: 8, Slug: "EMPTY_SEG"}
	segRepo.EXPECT().GetBySlug(gomock.Any(), "EMPTY_SEG").Return(seg, nil)
	segRepo.EXPECT().ListMemberUserIDs(gomock.Any(), int64(8)).Return([]uuid.UUID{}, nil)
	segRepo.EXPECT().SoftDelete(gomock.Any(), int64(8)).Return(nil)
	segRepo.EXPECT().DeleteMembershipsBySegment(gomock.Any(), int64(8)).Return(nil)
	// histRepo.BatchInsert must NOT be called (no expectation set).

	require.NoError(t, svc.Delete(context.Background(), "EMPTY_SEG"))
}
