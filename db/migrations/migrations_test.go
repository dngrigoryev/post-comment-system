package migrations

import (
	"context"
	"errors"
	"strings"
	"testing"

	"client-services/internal/testutil/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestParseOnlyUp(t *testing.T) {
	version, sql, err := parse("00001_init.sql", "-- +goose Up\nCREATE TABLE example(id INT);\n-- +goose Down\nDROP TABLE example;")
	if err != nil || version != 1 || !strings.Contains(sql, "CREATE TABLE") || strings.Contains(sql, "DROP TABLE") {
		t.Fatalf("%d %s %v", version, sql, err)
	}
	for _, tc := range []struct{ name, sql string }{{"bad.sql", "-- +goose Up\nSELECT 1"}, {"0_init.sql", "-- +goose Up\nSELECT 1"}, {"1_init.sql", "SELECT 1"}, {"1_init.sql", "-- +goose Up\n-- +goose Down"}} {
		if _, _, err := parse(tc.name, tc.sql); err == nil {
			t.Fatal(tc)
		}
	}
}
func TestApplyTransactionAndIdempotence(t *testing.T) {
	broken := errors.New("migration error")
	for _, tc := range []struct {
		name                string
		applied             bool
		writeErr, commitErr error
	}{{name: "new database"}, {name: "already applied", applied: true}, {name: "SQL failure", writeErr: broken}, {name: "commit failure", commitErr: broken}} {
		t.Run(tc.name, func(t *testing.T) {
			creates, records := 0, 0
			committed, rolledBack := false, false
			tx := &pgtest.Tx{DB: &pgtest.DB{}, CommitFunc: func(context.Context) error { committed = true; return tc.commitErr }, RollbackFunc: func(context.Context) error { rolledBack = true; return nil }}
			tx.RowFunc = func(_ context.Context, s string, a ...any) pgx.Row {
				if len(a) != 1 || a[0] != int64(1) {
					t.Fatalf("arguments: %v", a)
				}
				return pgtest.Row{Values: []any{tc.applied}}
			}
			tx.ExecFunc = func(_ context.Context, s string, a ...any) (pgconn.CommandTag, error) {
				if strings.Contains(s, "CREATE TABLE users") {
					creates++
					if strings.Contains(s, "DROP TABLE") {
						t.Fatal("Down section executed")
					}
					return pgconn.CommandTag{}, tc.writeErr
				}
				if strings.HasPrefix(s, "INSERT INTO schema_migrations") {
					records++
				}
				return pgconn.CommandTag{}, nil
			}
			db := &pgtest.DB{BeginFunc: func(context.Context, pgx.TxOptions) (pgx.Tx, error) { return tx, nil }}
			err := Apply(t.Context(), db)
			want := tc.writeErr
			if want == nil {
				want = tc.commitErr
			}
			if !errors.Is(err, want) {
				t.Fatal(err)
			}
			if !rolledBack || committed != (tc.writeErr == nil) {
				t.Fatal("transaction lifecycle")
			}
			if tc.applied {
				if creates != 0 || records != 0 {
					t.Fatal("migration repeated")
				}
			} else {
				if creates != 1 || records != boolInt(tc.writeErr == nil) {
					t.Fatalf("creates=%d records=%d", creates, records)
				}
			}
		})
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
