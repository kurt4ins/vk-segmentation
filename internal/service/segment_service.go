package service

import (
	"context"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type SegmentService struct {
	segments SegmentRepository
	history  HistoryRepository
	tx       Transactor
}

func NewSegmentService(segments SegmentRepository, history HistoryRepository, tx Transactor) *SegmentService {
	return &SegmentService{segments: segments, history: history, tx: tx}
}

func (s *SegmentService) Create(ctx context.Context, slug string, autoPercent *int) (domain.Segment, error) {
	if err := domain.ValidateSlug(slug); err != nil {
		return domain.Segment{}, err
	}
	if err := domain.ValidatePercent(autoPercent); err != nil {
		return domain.Segment{}, err
	}
	return s.segments.Create(ctx, slug, autoPercent)
}

func (s *SegmentService) List(ctx context.Context) ([]domain.Segment, error) {
	return s.segments.List(ctx)
}

func (s *SegmentService) Delete(ctx context.Context, slug string) error {
	seg, err := s.segments.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		userIDs, err := s.segments.ListMemberUserIDs(ctx, seg.ID)
		if err != nil {
			return err
		}
		if err := s.segments.SoftDelete(ctx, seg.ID); err != nil {
			return err
		}
		if err := s.segments.DeleteMembershipsBySegment(ctx, seg.ID); err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		records := make([]domain.HistoryRecord, len(userIDs))
		for i, uid := range userIDs {
			records[i] = domain.HistoryRecord{
				UserID:    uid,
				Slug:      seg.Slug,
				Operation: domain.OpRemove,
			}
		}
		return s.history.BatchInsert(ctx, records)
	})
}
