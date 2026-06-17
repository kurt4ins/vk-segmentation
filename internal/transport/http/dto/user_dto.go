package dto

import (
	"fmt"
	"time"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type UserResponse struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u domain.User) UserResponse {
	return UserResponse{ID: u.ID, CreatedAt: u.CreatedAt}
}

type UpdateSegmentsRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
	TTL    *string  `json:"ttl,omitempty"`
}

func (r UpdateSegmentsRequest) TTLDuration() (*time.Duration, error) {
	if r.TTL == nil || *r.TTL == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(*r.TTL)
	if err != nil {
		return nil, fmt.Errorf("invalid ttl %q: must be a duration like \"24h\": %w", *r.TTL, domain.ErrValidation)
	}
	if d <= 0 {
		return nil, fmt.Errorf("invalid ttl %q: must be positive: %w", *r.TTL, domain.ErrValidation)
	}
	return &d, nil
}

type ActiveSegmentResponse struct {
	Slug      string     `json:"slug"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func NewActiveSegmentResponses(items []domain.ActiveSegment) []ActiveSegmentResponse {
	out := make([]ActiveSegmentResponse, len(items))
	for i, a := range items {
		out[i] = ActiveSegmentResponse{Slug: a.Slug, ExpiresAt: a.ExpiresAt}
	}
	return out
}
