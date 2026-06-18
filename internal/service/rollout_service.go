package service

import (
	"context"
	"math"

	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type RolloutEnqueuer interface {
	Enqueue(segment domain.Segment)
}

type RolloutService struct {
	users       UserRepository
	memberships MembershipRepository
	history     HistoryRepository
	segments    SegmentRepository
	tx          Transactor
	batchSize   int
}

func NewRolloutService(
	users UserRepository,
	memberships MembershipRepository,
	history HistoryRepository,
	segments SegmentRepository,
	tx Transactor,
	batchSize int,
) *RolloutService {
	if batchSize <= 0 {
		batchSize = 1000
	}
	return &RolloutService{
		users:       users,
		memberships: memberships,
		history:     history,
		segments:    segments,
		tx:          tx,
		batchSize:   batchSize,
	}
}

func (s *RolloutService) Apply(ctx context.Context, segment domain.Segment) error {
	if segment.AutoAssignPercent != nil {
		percent := *segment.AutoAssignPercent

		total, err := s.users.Count(ctx)
		if err != nil {
			return err
		}

		target := int(math.Round(float64(total) * float64(percent) / 100))
		if target > 0 {
			userIDs, err := s.users.ListNonMembers(ctx, segment.ID, target)
			if err != nil {
				return err
			}
			for _, batch := range chunkUUIDs(userIDs, s.batchSize) {
				if err := s.applyBatch(ctx, segment, batch); err != nil {
					return err
				}
			}
		}
	}

	return s.segments.MarkApplied(ctx, segment.ID)
}

func (s *RolloutService) applyBatch(ctx context.Context, segment domain.Segment, userIDs []uuid.UUID) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		added, err := s.memberships.BatchAddUsers(ctx, segment.ID, userIDs)
		if err != nil {
			return err
		}

		records := make([]domain.HistoryRecord, 0, len(added))
		for _, id := range added {
			records = append(records, domain.HistoryRecord{
				UserID:    id,
				Slug:      segment.Slug,
				Operation: domain.OpAdd,
			})
		}
		return s.history.BatchInsert(ctx, records)
	})
}

func chunkUUIDs(ids []uuid.UUID, size int) [][]uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	chunks := make([][]uuid.UUID, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := min(start+size, len(ids))
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}
