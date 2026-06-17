package errmap_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
	"github.com/kurt4ins/vk-segmentation/internal/pkg/errmap"
)

func TestWrite_MapsDomainErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not found", domain.ErrSegmentNotFound, http.StatusNotFound, "not_found"},
		{"conflict", domain.ErrSegmentAlreadyExists, http.StatusConflict, "conflict"},
		{"validation", fmt.Errorf("bad: %w", domain.ErrValidation), http.StatusBadRequest, "validation_error"},
		{"internal", fmt.Errorf("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			errmap.Write(rec, tc.err)

			require.Equal(t, tc.wantCode, rec.Code)

			var body struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.wantBody, body.Error.Code)
		})
	}
}

func TestWrite_HidesInternalMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	errmap.Write(rec, fmt.Errorf("sensitive db dsn leaked"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "sensitive")
	require.Contains(t, rec.Body.String(), "internal server error")
}
