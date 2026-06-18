package errmap

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

type envelope struct {
	Error errBody `json:"error"`
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, "validation_error"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}

func Write(w http.ResponseWriter, err error) {
	status, code := classify(err)

	msg := err.Error()
	if status >= http.StatusInternalServerError {
		msg = "internal server error"
	}

	WriteCode(w, status, code, msg)
}

func WriteCode(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: errBody{Code: code, Message: message}})
}
