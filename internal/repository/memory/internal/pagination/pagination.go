package pagination

import (
	"fmt"
	"math"
	"sort"

	"client-services/internal/domain"
)

func ValidateSize(first int) error {
	if first < 1 || first >= math.MaxInt32 {
		return fmt.Errorf("%w: page size is out of range", domain.ErrInvalidArgument)
	}
	return nil
}

func IDs(ids []int64, afterID int64, first int) []int64 {
	start := sort.Search(len(ids), func(i int) bool {
		return ids[i] > afterID
	})
	end := start + min(first+1, len(ids)-start)
	return ids[start:end]
}
