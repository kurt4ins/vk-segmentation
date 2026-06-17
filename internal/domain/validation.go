package domain

import (
	"fmt"
	"regexp"
)

var slugPattern = regexp.MustCompile(`^[A-Z0-9_]{1,100}$`)

func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must match [A-Z0-9_]{1,100}: %w", slug, ErrValidation)
	}
	return nil
}

func ValidatePercent(p *int) error {
	if p == nil {
		return nil
	}
	if *p < 0 || *p > 100 {
		return fmt.Errorf("invalid auto_assign_percent %d: must be between 0 and 100: %w", *p, ErrValidation)
	}
	return nil
}
