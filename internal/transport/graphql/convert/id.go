package convert

import (
	"fmt"
	"strconv"

	"client-services/internal/domain"
)

func ID(name, raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive integer", domain.ErrInvalidArgument, name)
	}
	return id, nil
}
