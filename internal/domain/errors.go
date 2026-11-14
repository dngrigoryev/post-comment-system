package domain

import "errors"

var (
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrUnauthenticated  = errors.New("authentication required")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrCommentsDisabled = errors.New("comments are disabled")
)
