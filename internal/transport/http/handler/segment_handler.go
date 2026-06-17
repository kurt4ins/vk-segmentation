package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/pkg/errmap"
	"github.com/kurt4ins/vk-segmentation/internal/transport/http/dto"
)

type SegmentService interface {
	Create(ctx context.Context, slug string, autoPercent *int) (domain.Segment, error)
	List(ctx context.Context) ([]domain.Segment, error)
	Delete(ctx context.Context, slug string) error
}

type SegmentHandler struct {
	svc SegmentService
}

func NewSegmentHandler(svc SegmentService) *SegmentHandler {
	return &SegmentHandler{svc: svc}
}

func (h *SegmentHandler) Register(r chi.Router) {
	r.Post("/segments", h.create)
	r.Get("/segments", h.list)
	r.Delete("/segments/{slug}", h.delete)
}

func (h *SegmentHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errmap.WriteCode(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		errmap.Write(w, err)
		return
	}

	seg, err := h.svc.Create(r.Context(), req.Slug, req.AutoAssignPercent)
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.NewSegmentResponse(seg))
}

func (h *SegmentHandler) list(w http.ResponseWriter, r *http.Request) {
	segments, err := h.svc.List(r.Context())
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.NewSegmentResponses(segments))
}

func (h *SegmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := h.svc.Delete(r.Context(), slug); err != nil {
		errmap.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
