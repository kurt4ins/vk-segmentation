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

func newMembershipService(ctrl *gomock.Controller) (
	*service.MembershipService,
	*mocks.MockUserRepository,
	*mocks.MockSegmentRepository,
	*mocks.MockMembershipRepository,
	*mocks.MockHistoryRepository,
) {
	users := mocks.NewMockUserRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)
	svc := service.NewMembershipService(users, segments, memberships, history, fakeTx{})
	return svc, users, segments, memberships, history
}

func TestMembershipService_UpdateSegments_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, users, _, _, _ := newMembershipService(ctrl)

	uid := uuid.New()
	users.EXPECT().Exists(gomock.Any(), uid).Return(false, nil)

	_, err := svc.UpdateSegments(context.Background(), uid, []string{"A"}, nil, nil)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestMembershipService_UpdateSegments_DisjointViolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, users, _, _, _ := newMembershipService(ctrl)

	uid := uuid.New()
	users.EXPECT().Exists(gomock.Any(), uid).Return(true, nil)

	_, err := svc.UpdateSegments(context.Background(), uid, []string{"A"}, []string{"A"}, nil)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestMembershipService_UpdateSegments_UnknownSlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, users, segments, _, _ := newMembershipService(ctrl)

	uid := uuid.New()
	users.EXPECT().Exists(gomock.Any(), uid).Return(true, nil)
	// only A exists, B is unknown.
	segments.EXPECT().ListBySlugs(gomock.Any(), gomock.Any()).Return([]domain.Segment{{ID: 1, Slug: "A"}}, nil)

	_, err := svc.UpdateSegments(context.Background(), uid, []string{"A", "B"}, nil, nil)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestMembershipService_UpdateSegments_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, users, segments, memberships, history := newMembershipService(ctrl)

	uid := uuid.New()
	users.EXPECT().Exists(gomock.Any(), uid).Return(true, nil)
	segments.EXPECT().ListBySlugs(gomock.Any(), gomock.Any()).Return([]domain.Segment{
		{ID: 1, Slug: "A"},
		{ID: 2, Slug: "B"},
	}, nil)
	memberships.EXPECT().BatchInsert(gomock.Any(), uid, []int64{1}, gomock.Nil()).Return([]int64{1}, nil)
	memberships.EXPECT().BatchDelete(gomock.Any(), uid, []int64{2}).Return([]int64{2}, nil)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 2)
			require.Equal(t, domain.OpAdd, recs[0].Operation)
			require.Equal(t, "A", recs[0].Slug)
			require.Equal(t, domain.OpRemove, recs[1].Operation)
			require.Equal(t, "B", recs[1].Slug)
			return nil
		})
	active := []domain.ActiveSegment{{Slug: "A"}}
	memberships.EXPECT().ListActive(gomock.Any(), uid).Return(active, nil)

	got, err := svc.UpdateSegments(context.Background(), uid, []string{"A"}, []string{"B"}, nil)
	require.NoError(t, err)
	require.Equal(t, active, got)
}

func TestMembershipService_CleanExpired(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _, memberships, history := newMembershipService(ctrl)

	uid := uuid.New()
	memberships.EXPECT().DeleteExpired(gomock.Any()).Return([]domain.ExpiredMembership{
		{UserID: uid, Slug: "A"},
		{UserID: uid, Slug: "B"},
	}, nil)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 2)
			for _, r := range recs {
				require.Equal(t, domain.OpRemove, r.Operation)
			}
			return nil
		})

	removed, err := svc.CleanExpired(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, removed)
}

func TestMembershipService_CleanExpired_NothingToDo(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, _, memberships, _ := newMembershipService(ctrl)

	memberships.EXPECT().DeleteExpired(gomock.Any()).Return(nil, nil)
	// history.BatchInsert must NOT be called.

	removed, err := svc.CleanExpired(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, removed)
}
