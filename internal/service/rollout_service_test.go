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

func TestRolloutService_Apply_Target(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)

	seg := domain.Segment{ID: 5, Slug: "AB_TEST", AutoAssignPercent: intp(50)}
	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = uuid.New()
	}

	// N=10, P=50 -> target = round(10*50/100) = 5.
	users.EXPECT().Count(gomock.Any()).Return(int64(10), nil)
	users.EXPECT().ListNonMembers(gomock.Any(), int64(5), 5).Return(ids, nil)
	memberships.EXPECT().BatchAddUsers(gomock.Any(), int64(5), ids).Return(ids, nil)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 5)
			for _, r := range recs {
				require.Equal(t, "AB_TEST", r.Slug)
				require.Equal(t, domain.OpAdd, r.Operation)
			}
			return nil
		})
	segments.EXPECT().MarkApplied(gomock.Any(), int64(5)).Return(nil)

	svc := service.NewRolloutService(users, memberships, history, segments, fakeTx{}, 1000)
	require.NoError(t, svc.Apply(context.Background(), seg))
}

func TestRolloutService_Apply_ZeroTargetOnlyMarksApplied(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)

	seg := domain.Segment{ID: 9, Slug: "NONE", AutoAssignPercent: intp(0)}
	users.EXPECT().Count(gomock.Any()).Return(int64(10), nil)
	// target=0 -> no ListNonMembers / BatchAddUsers.
	segments.EXPECT().MarkApplied(gomock.Any(), int64(9)).Return(nil)

	svc := service.NewRolloutService(users, mocks.NewMockMembershipRepository(ctrl), mocks.NewMockHistoryRepository(ctrl), segments, fakeTx{}, 1000)
	require.NoError(t, svc.Apply(context.Background(), seg))
}

func TestRolloutService_Apply_Batches(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)

	seg := domain.Segment{ID: 3, Slug: "AB_TEST", AutoAssignPercent: intp(100)}
	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = uuid.New()
	}

	users.EXPECT().Count(gomock.Any()).Return(int64(5), nil)
	users.EXPECT().ListNonMembers(gomock.Any(), int64(3), 5).Return(ids, nil)
	// batchSize=2, 5 users -> 3 batches.
	memberships.EXPECT().BatchAddUsers(gomock.Any(), int64(3), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ int64, batch []uuid.UUID) ([]uuid.UUID, error) {
			require.LessOrEqual(t, len(batch), 2)
			return batch, nil
		}).Times(3)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).Times(3)
	segments.EXPECT().MarkApplied(gomock.Any(), int64(3)).Return(nil)

	svc := service.NewRolloutService(users, memberships, history, segments, fakeTx{}, 2)
	require.NoError(t, svc.Apply(context.Background(), seg))
}

func TestRolloutService_Apply_NilPercentOnlyMarksApplied(t *testing.T) {
	ctrl := gomock.NewController(t)
	segments := mocks.NewMockSegmentRepository(ctrl)

	seg := domain.Segment{ID: 11, Slug: "PLAIN"}
	segments.EXPECT().MarkApplied(gomock.Any(), int64(11)).Return(nil)

	svc := service.NewRolloutService(
		mocks.NewMockUserRepository(ctrl),
		mocks.NewMockMembershipRepository(ctrl),
		mocks.NewMockHistoryRepository(ctrl),
		segments, fakeTx{}, 1000)
	require.NoError(t, svc.Apply(context.Background(), seg))
}
