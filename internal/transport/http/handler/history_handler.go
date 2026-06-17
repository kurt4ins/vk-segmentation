package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/pkg/errmap"
	"github.com/kurt4ins/vk-segmentation/internal/transport/http/dto"
)

const ReportsURLPrefix = "/reports/"

type HistoryService interface {
	Report(ctx context.Context, userID uuid.UUID, from, to time.Time) (string, error)
}

type HistoryHandler struct {
	svc HistoryService
}

func NewHistoryHandler(svc HistoryService) *HistoryHandler {
	return &HistoryHandler{svc: svc}
}

func (h *HistoryHandler) Register(r chi.Router) {
	r.Get("/users/{id}/history", h.report)
}

func (h *HistoryHandler) report(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserID(w, r)
	if !ok {
		return
	}

	from, err := parseRange(r.URL.Query().Get("from"), time.Time{})
	if err != nil {
		errmap.Write(w, err)
		return
	}
	to, err := parseRange(r.URL.Query().Get("to"), time.Now().UTC())
	if err != nil {
		errmap.Write(w, err)
		return
	}

	filename, err := h.svc.Report(r.Context(), userID, from, to)
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ReportResponse{Link: ReportsURLPrefix + filename})
}

func parseRange(raw string, def time.Time) (time.Time, error) {
	if raw == "" {
		return def, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q: use RFC3339 or YYYY-MM-DD: %w", raw, domain.ErrValidation)
}
