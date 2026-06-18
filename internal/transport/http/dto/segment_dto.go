package dto

import (
	"time"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type CreateSegmentRequest struct {
	Slug              string `json:"slug"`
	AutoAssignPercent *int   `json:"auto_assign_percent,omitempty"`
}

func (r CreateSegmentRequest) Validate() error {
	if err := domain.ValidateSlug(r.Slug); err != nil {
		return err
	}
	return domain.ValidatePercent(r.AutoAssignPercent)
}

type SegmentResponse struct {
	Slug              string    `json:"slug"`
	AutoAssignPercent *int      `json:"auto_assign_percent,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewSegmentResponse(s domain.Segment) SegmentResponse {
	return SegmentResponse{
		Slug:              s.Slug,
		AutoAssignPercent: s.AutoAssignPercent,
		Status:            string(s.Status),
		CreatedAt:         s.CreatedAt,
	}
}

func NewSegmentResponses(segments []domain.Segment) []SegmentResponse {
	out := make([]SegmentResponse, len(segments))
	for i, s := range segments {
		out[i] = NewSegmentResponse(s)
	}
	return out
}
