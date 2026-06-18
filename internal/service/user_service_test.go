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

func TestUserService_Register_NoPercentSegments(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)

	uid := uuid.New()
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.User{ID: uid}, nil)
	segments.EXPECT().ListPercentSegments(gomock.Any()).Return(nil, nil)
	// no BatchInsert / history expected.

	svc := service.NewUserService(users, segments, memberships, history, fakeTx{})
	u, err := svc.Register(context.Background())
	require.NoError(t, err)
	require.Equal(t, uid, u.ID)
}

func TestUserService_Register_DiceAllIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)

	uid := uuid.New()
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.User{ID: uid}, nil)
	segments.EXPECT().ListPercentSegments(gomock.Any()).Return([]domain.Segment{
		{ID: 1, Slug: "A", AutoAssignPercent: intp(50)},
		{ID: 2, Slug: "B", AutoAssignPercent: intp(100)},
	}, nil)
	// rng==0 -> 0 < P/100 for any P>0 -> both segments selected.
	memberships.EXPECT().BatchInsert(gomock.Any(), uid, []int64{1, 2}, gomock.Nil()).Return([]int64{1, 2}, nil)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 2)
			for _, r := range recs {
				require.Equal(t, domain.OpAdd, r.Operation)
			}
			return nil
		})

	svc := service.NewUserService(users, segments, memberships, history, fakeTx{},
		service.WithRNG(func() float64 { return 0 }))
	_, err := svc.Register(context.Background())
	require.NoError(t, err)
}

func TestUserService_Register_DiceSelective(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	segments := mocks.NewMockSegmentRepository(ctrl)
	memberships := mocks.NewMockMembershipRepository(ctrl)
	history := mocks.NewMockHistoryRepository(ctrl)

	uid := uuid.New()
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.User{ID: uid}, nil)
	segments.EXPECT().ListPercentSegments(gomock.Any()).Return([]domain.Segment{
		{ID: 1, Slug: "A", AutoAssignPercent: intp(50)},
		{ID: 2, Slug: "B", AutoAssignPercent: intp(100)},
	}, nil)
	// rng==0.6 -> 0.6 < 0.50 false (skip A); 0.6 < 1.00 true (select B).
	memberships.EXPECT().BatchInsert(gomock.Any(), uid, []int64{2}, gomock.Nil()).Return([]int64{2}, nil)
	history.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, recs []domain.HistoryRecord) error {
			require.Len(t, recs, 1)
			require.Equal(t, "B", recs[0].Slug)
			require.Equal(t, domain.OpAdd, recs[0].Operation)
			return nil
		})

	svc := service.NewUserService(users, segments, memberships, history, fakeTx{},
		service.WithRNG(func() float64 { return 0.6 }))
	_, err := svc.Register(context.Background())
	require.NoError(t, err)
}
