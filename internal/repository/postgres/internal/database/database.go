package database

import (
	"context"

	"client-services/internal/repository/postgres/dbgen"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	dbgen.DBTX
	BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error)
}
