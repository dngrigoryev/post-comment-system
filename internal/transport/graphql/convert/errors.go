package convert

import (
	"errors"

	"client-services/internal/domain"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func Error(err error) error {
	if err == nil {
		return nil
	}
	code := "INTERNAL_SERVER_ERROR"
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		code, message = "BAD_USER_INPUT", err.Error()
	case errors.Is(err, domain.ErrUnauthenticated):
		code, message = "UNAUTHENTICATED", domain.ErrUnauthenticated.Error()
	case errors.Is(err, domain.ErrForbidden):
		code, message = "FORBIDDEN", domain.ErrForbidden.Error()
	case errors.Is(err, domain.ErrNotFound):
		code, message = "NOT_FOUND", domain.ErrNotFound.Error()
	case errors.Is(err, domain.ErrCommentsDisabled):
		code, message = "COMMENTS_DISABLED", domain.ErrCommentsDisabled.Error()
	}
	return &gqlerror.Error{
		Message:    message,
		Extensions: map[string]any{"code": code},
	}
}
