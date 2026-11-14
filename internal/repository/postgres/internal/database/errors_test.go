package database

import (
	"context"
	"errors"
	"math"
	"testing"

	"client-services/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDomainErrors(t *testing.T) {
	for _, tc := range []struct{ in, want error }{{pgx.ErrNoRows, domain.ErrNotFound}, {&pgconn.PgError{Code: "23503"}, domain.ErrInvalidArgument}, {&pgconn.PgError{Code: "23502"}, domain.ErrInvalidArgument}, {&pgconn.PgError{Code: "23514"}, domain.ErrInvalidArgument}, {&pgconn.PgError{Code: "22001"}, domain.ErrInvalidArgument}, {context.Canceled, context.Canceled}} {
		if err := WrapError("operation", tc.in); !errors.Is(err, tc.want) {
			t.Fatalf("%v -> %v", tc.in, err)
		}
	}
	unknown := &pgconn.PgError{Code: "XX000"}
	var got *pgconn.PgError
	if !errors.As(WrapError("read", unknown), &got) || got != unknown {
		t.Fatal("unknown database error lost")
	}
}
func TestLimitAndNullableID(t *testing.T) {
	for _, first := range []int{0, -1, math.MaxInt32} {
		if _, err := PageLimit(first); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	limit, err := PageLimit(20)
	if err != nil || limit != 21 {
		t.Fatal(limit, err)
	}
	if NullableID(nil).Valid {
		t.Fatal("NULL ID became zero")
	}
	id := int64(1 << 40)
	if got := NullableID(&id); !got.Valid || got.Int64 != id {
		t.Fatal(got)
	}
}
