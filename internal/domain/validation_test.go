package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kurt4ins/vk-segmentation/internal/domain"
)

func TestValidateSlug(t *testing.T) {
	valid := []string{"MAIL_GPT", "AB_TEST", "A", "X1_Y2"}
	for _, s := range valid {
		require.NoError(t, domain.ValidateSlug(s), s)
	}

	invalid := []string{"", "lower", "with space", "BAD!", "tab\t"}
	for _, s := range invalid {
		require.ErrorIs(t, domain.ValidateSlug(s), domain.ErrValidation, s)
	}
}

func TestValidatePercent(t *testing.T) {
	require.NoError(t, domain.ValidatePercent(nil))
	for _, p := range []int{0, 50, 100} {
		require.NoError(t, domain.ValidatePercent(&p), p)
	}
	for _, p := range []int{-1, 101, 1000} {
		require.ErrorIs(t, domain.ValidatePercent(&p), domain.ErrValidation, p)
	}
}
