package migrations

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed *.sql
var files embed.FS

type Database interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func Apply(ctx context.Context, db Database) error {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT PG_ADVISORY_XACT_LOCK(7247314501)"); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY)"); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	entries, err := files.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := files.ReadFile(entry.Name())
		if err != nil {
			return err
		}
		version, statement, err := parse(entry.Name(), string(data))
		if err != nil {
			return err
		}
		var applied bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&applied); err != nil {
			return fmt.Errorf("read migration %d: %w", version, err)
		}
		if applied {
			continue
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func parse(name, data string) (int64, string, error) {
	prefix, _, ok := strings.Cut(name, "_")
	version, err := strconv.ParseInt(prefix, 10, 64)
	if !ok || err != nil || version <= 0 {
		return 0, "", fmt.Errorf("invalid migration filename: %s", name)
	}
	_, up, ok := strings.Cut(data, "-- +goose Up")
	if !ok {
		return 0, "", fmt.Errorf("missing Up section: %s", name)
	}
	up, _, _ = strings.Cut(up, "-- +goose Down")
	if strings.TrimSpace(up) == "" {
		return 0, "", fmt.Errorf("empty Up section: %s", name)
	}
	return version, up, nil
}
