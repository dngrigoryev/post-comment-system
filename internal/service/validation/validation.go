package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"client-services/internal/domain"
)

const (
	MaxPageSize = 100
)

func ID(name string, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: %s must be positive", domain.ErrInvalidArgument, name)
	}
	return nil
}

func User(userID int64) error {
	if userID <= 0 {
		return domain.ErrUnauthenticated
	}
	return nil
}

func Text(name, text string, maxLength int) error {
	if !utf8.ValidString(text) || strings.TrimSpace(text) == "" {
		return fmt.Errorf("%w: %s must be non-blank UTF-8 text", domain.ErrInvalidArgument, name)
	}
	if maxLength > 0 && utf8.RuneCountInString(text) > maxLength {
		return fmt.Errorf("%w: %s must contain at most %d characters", domain.ErrInvalidArgument, name, maxLength)
	}
	return nil
}

func Page(page domain.PageRequest) error {
	if page.First < 1 || page.First > MaxPageSize {
		return fmt.Errorf("%w: first must be between 1 and %d", domain.ErrInvalidArgument, MaxPageSize)
	}
	if page.After != nil && *page.After == "" {
		return fmt.Errorf("%w: after must not be empty", domain.ErrInvalidArgument)
	}
	return nil
}
