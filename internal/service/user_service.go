package service

import (
	"context"
	"math/rand/v2"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserService struct {
	users       UserRepository
	segments    SegmentRepository
	memberships MembershipRepository
	history     HistoryRepository
	tx          Transactor
	rng         func() float64
}

type UserServiceOption func(*UserService)

func WithRNG(rng func() float64) UserServiceOption {
	return func(s *UserService) { s.rng = rng }
}

func NewUserService(
	users UserRepository,
	segments SegmentRepository,
	memberships MembershipRepository,
	history HistoryRepository,
	tx Transactor,
	opts ...UserServiceOption,
) *UserService {
	s := &UserService{
		users:       users,
		segments:    segments,
		memberships: memberships,
		history:     history,
		tx:          tx,
		rng:         rand.Float64,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *UserService) Register(ctx context.Context) (domain.User, error) {
	var user domain.User
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := s.users.Create(ctx)
		if err != nil {
			return err
		}
		user = u

		segments, err := s.segments.ListPercentSegments(ctx)
		if err != nil {
			return err
		}

		idToSlug := make(map[int64]string, len(segments))
		addIDs := make([]int64, 0, len(segments))
		for _, seg := range segments {
			if seg.AutoAssignPercent == nil {
				continue
			}
			idToSlug[seg.ID] = seg.Slug
			if s.rng() < float64(*seg.AutoAssignPercent)/100 {
				addIDs = append(addIDs, seg.ID)
			}
		}
		if len(addIDs) == 0 {
			return nil
		}

		inserted, err := s.memberships.BatchInsert(ctx, user.ID, addIDs, nil)
		if err != nil {
			return err
		}

		records := make([]domain.HistoryRecord, 0, len(inserted))
		for _, id := range inserted {
			records = append(records, domain.HistoryRecord{
				UserID:    user.ID,
				Slug:      idToSlug[id],
				Operation: domain.OpAdd,
			})
		}
		return s.history.BatchInsert(ctx, records)
	})
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}
