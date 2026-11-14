package users

import (
	"context"
	"errors"
	"testing"

	"client-services/internal/domain"
	"client-services/internal/testutil/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestGetAndBatchUsers(t *testing.T) {
	calls := 0
	db := &pgtest.DB{RowFunc: func(context.Context, string, ...any) pgx.Row { return pgtest.Row{Values: []any{int64(3), "Alice"}} }, QueryFunc: func(context.Context, string, ...any) (pgx.Rows, error) {
		calls++
		return pgtest.NewRows([]any{int64(3), "Alice"}, []any{int64(1), "Bob"}), nil
	}}
	r := New(db)
	u, err := r.GetUser(t.Context(), 3)
	if err != nil || u.ID != 3 || u.Name != "Alice" {
		t.Fatal(u, err)
	}
	users, err := r.GetUsersBatch(t.Context(), []int64{1, 3, 99, 1})
	if err != nil || len(users) != 2 || users[1].Name != "Bob" {
		t.Fatal(users, err)
	}
	users, err = r.GetUsersBatch(t.Context(), nil)
	if err != nil || len(users) != 0 || calls != 1 {
		t.Fatal(users, err)
	}
	db.RowFunc = func(context.Context, string, ...any) pgx.Row { return pgtest.Row{Err: pgx.ErrNoRows} }
	_, err = r.GetUser(t.Context(), 99)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}
func TestEnsureUser(t *testing.T) {
	db := &pgtest.DB{ExecFunc: func(_ context.Context, s string, a ...any) (pgconn.CommandTag, error) {
		if len(a) != 3 || a[0] != int64(7) || a[1] != "Alice" || a[2] != "alice@example.com" {
			t.Fatalf("arguments %v", a)
		}
		return pgconn.CommandTag{}, nil
	}}
	if err := New(db).EnsureUser(t.Context(), domain.User{ID: 7, Name: "Alice"}, "alice@example.com"); err != nil {
		t.Fatal(err)
	}
}
