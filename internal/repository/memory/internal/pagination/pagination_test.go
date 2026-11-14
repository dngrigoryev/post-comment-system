package pagination

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"client-services/internal/domain"
)

func TestWindowAfterCursor(t *testing.T) {
	ids := []int64{2, 5, 8, 11, 14}
	for _, tc := range []struct {
		name  string
		after int64
		first int
		want  []int64
	}{
		{"first page", 0, 2, []int64{2, 5, 8}},
		{"exact cursor", 5, 2, []int64{8, 11, 14}},
		{"cursor between IDs", 6, 1, []int64{8, 11}},
		{"partial last page", 11, 2, []int64{14}},
		{"end of list", 14, 2, []int64{}},
		{"cursor past list", 100, 2, []int64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IDs(ids, tc.after, tc.first); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	if got := IDs(nil, 0, 2); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestPageSizeBounds(t *testing.T) {
	for _, n := range []int{-1, 0, math.MaxInt32} {
		if err := ValidateSize(n); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(n, err)
		}
	}
	for _, n := range []int{1, 100, math.MaxInt32 - 1} {
		if err := ValidateSize(n); err != nil {
			t.Fatal(n, err)
		}
	}
}
