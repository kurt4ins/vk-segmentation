package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type MembershipService struct {
	users       UserRepository
	segments    SegmentRepository
	memberships MembershipRepository
	history     HistoryRepository
	tx          Transactor
}

func NewMembershipService(
	users UserRepository,
	segments SegmentRepository,
	memberships MembershipRepository,
	history HistoryRepository,
	tx Transactor,
) *MembershipService {
	return &MembershipService{
		users:       users,
		segments:    segments,
		memberships: memberships,
		history:     history,
		tx:          tx,
	}
}

func (s *MembershipService) ListActive(ctx context.Context, userID int64) ([]domain.ActiveSegment, error) {
	return s.memberships.ListActive(ctx, userID)
}

func (s *MembershipService) UpdateSegments(
	ctx context.Context,
	userID int64,
	add []string,
	remove []string,
	ttl *time.Duration,
) ([]domain.ActiveSegment, error) {
	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	add = uniqueSlugs(add)
	remove = uniqueSlugs(remove)

	removeSet := make(map[string]struct{}, len(remove))
	for _, slug := range remove {
		removeSet[slug] = struct{}{}
	}
	for _, slug := range add {
		if _, ok := removeSet[slug]; ok {
			return nil, fmt.Errorf("slug %q present in both add and remove: %w", slug, domain.ErrValidation)
		}
	}

	slugToID, idToSlug, err := s.resolveSlugs(ctx, append(append([]string{}, add...), remove...))
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if ttl != nil {
		t := time.Now().Add(*ttl)
		expiresAt = &t
	}

	addIDs := idsForSlugs(add, slugToID)
	removeIDs := idsForSlugs(remove, slugToID)

	var active []domain.ActiveSegment
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		inserted, err := s.memberships.BatchInsert(ctx, userID, addIDs, expiresAt)
		if err != nil {
			return err
		}
		deleted, err := s.memberships.BatchDelete(ctx, userID, removeIDs)
		if err != nil {
			return err
		}

		records := make([]domain.HistoryRecord, 0, len(inserted)+len(deleted))
		for _, id := range inserted {
			records = append(records, domain.HistoryRecord{UserID: userID, Slug: idToSlug[id], Operation: domain.OpAdd})
		}
		for _, id := range deleted {
			records = append(records, domain.HistoryRecord{UserID: userID, Slug: idToSlug[id], Operation: domain.OpRemove})
		}
		if err := s.history.BatchInsert(ctx, records); err != nil {
			return err
		}

		active, err = s.memberships.ListActive(ctx, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return active, nil
}

func (s *MembershipService) resolveSlugs(ctx context.Context, slugs []string) (map[string]int64, map[int64]string, error) {
	slugs = uniqueSlugs(slugs)
	if len(slugs) == 0 {
		return map[string]int64{}, map[int64]string{}, nil
	}

	segments, err := s.segments.ListBySlugs(ctx, slugs)
	if err != nil {
		return nil, nil, err
	}

	slugToID := make(map[string]int64, len(segments))
	idToSlug := make(map[int64]string, len(segments))
	for _, seg := range segments {
		slugToID[seg.Slug] = seg.ID
		idToSlug[seg.ID] = seg.Slug
	}
	for _, slug := range slugs {
		if _, ok := slugToID[slug]; !ok {
			return nil, nil, fmt.Errorf("unknown segment slug %q: %w", slug, domain.ErrValidation)
		}
	}
	return slugToID, idToSlug, nil
}

func uniqueSlugs(slugs []string) []string {
	seen := make(map[string]struct{}, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	return out
}

func idsForSlugs(slugs []string, slugToID map[string]int64) []int64 {
	ids := make([]int64, 0, len(slugs))
	for _, slug := range slugs {
		ids = append(ids, slugToID[slug])
	}
	return ids
}
