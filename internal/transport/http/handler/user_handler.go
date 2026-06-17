package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/pkg/errmap"
	"github.com/kurt4ins/vk-segmentation/internal/transport/http/dto"
)

type UserService interface {
	Register(ctx context.Context) (domain.User, error)
}

type MembershipService interface {
	UpdateSegments(ctx context.Context, userID int64, add, remove []string, ttl *time.Duration) ([]domain.ActiveSegment, error)
	ListActive(ctx context.Context, userID int64) ([]domain.ActiveSegment, error)
}

type UserHandler struct {
	users       UserService
	memberships MembershipService
}

func NewUserHandler(users UserService, memberships MembershipService) *UserHandler {
	return &UserHandler{users: users, memberships: memberships}
}

func (h *UserHandler) Register(r chi.Router) {
	r.Post("/users", h.create)
	r.Post("/users/{id}/segments", h.updateSegments)
	r.Get("/users/{id}/segments", h.getSegments)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	user, err := h.users.Register(r.Context())
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.NewUserResponse(user))
}

func (h *UserHandler) updateSegments(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateSegmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errmap.WriteCode(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	ttl, err := req.TTLDuration()
	if err != nil {
		errmap.Write(w, err)
		return
	}

	active, err := h.memberships.UpdateSegments(r.Context(), userID, req.Add, req.Remove, ttl)
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.NewActiveSegmentResponses(active))
}

func (h *UserHandler) getSegments(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserID(w, r)
	if !ok {
		return
	}

	active, err := h.memberships.ListActive(r.Context(), userID)
	if err != nil {
		errmap.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.NewActiveSegmentResponses(active))
}

func parseUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		errmap.WriteCode(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return 0, false
	}
	return id, true
}
