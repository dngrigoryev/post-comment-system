package database

import (
	"fmt"
	"math"

	"client-services/internal/domain"
)

func PageLimit(first int) (int32, error) {
	if first < 1 || first >= math.MaxInt32 {
		return 0, fmt.Errorf("%w: page size is out of range", domain.ErrInvalidArgument)
	}
	return int32(first) + 1, nil
}
