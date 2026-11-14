package database

import (
	"errors"
	"fmt"

	"client-services/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func WrapError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("postgres %s: %w", operation, domain.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return fmt.Errorf("postgres %s: %w: invalid reference", operation, domain.ErrInvalidArgument)
		case "23502", "23514", "22001":
			return fmt.Errorf("postgres %s: %w: invalid stored value", operation, domain.ErrInvalidArgument)
		}
	}
	return fmt.Errorf("postgres %s: %w", operation, err)
}
