package pgtest

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type DB struct {
	QueryFunc func(context.Context, string, ...any) (pgx.Rows, error)
	RowFunc   func(context.Context, string, ...any) pgx.Row
	ExecFunc  func(context.Context, string, ...any) (pgconn.CommandTag, error)
	BeginFunc func(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func (d *DB) Query(ctx context.Context, s string, a ...any) (pgx.Rows, error) {
	return d.QueryFunc(ctx, s, a...)
}
func (d *DB) QueryRow(ctx context.Context, s string, a ...any) pgx.Row {
	return d.RowFunc(ctx, s, a...)
}
func (d *DB) Exec(ctx context.Context, s string, a ...any) (pgconn.CommandTag, error) {
	return d.ExecFunc(ctx, s, a...)
}
func (d *DB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	return d.BeginFunc(ctx, o)
}

type Row struct {
	Values []any
	Err    error
}

func (r Row) Scan(dest ...any) error {
	if r.Err != nil {
		return r.Err
	}
	return scan(dest, r.Values)
}

type Rows struct {
	Data     [][]any
	Error    error
	position int
	closed   bool
}

func NewRows(data ...[]any) *Rows { return &Rows{Data: data, position: -1} }
func (r *Rows) Next() bool {
	if r.closed {
		return false
	}
	r.position++
	return r.position < len(r.Data)
}
func (r *Rows) Scan(dest ...any) error                       { return scan(dest, r.Data[r.position]) }
func (r *Rows) Close()                                       { r.closed = true }
func (r *Rows) Err() error                                   { return r.Error }
func (r *Rows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *Rows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *Rows) Values() ([]any, error)                       { return r.Data[r.position], nil }
func (r *Rows) RawValues() [][]byte                          { return nil }
func (r *Rows) Conn() *pgx.Conn                              { return nil }

func scan(dest, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("scan: %d destinations, %d values", len(dest), len(values))
	}
	for i, d := range dest {
		v := reflect.ValueOf(d).Elem()
		src := reflect.ValueOf(values[i])
		if !src.IsValid() {
			v.SetZero()
		} else if src.Type().AssignableTo(v.Type()) {
			v.Set(src)
		} else {
			return fmt.Errorf("scan %d: cannot assign %T to %T", i, values[i], d)
		}
	}
	return nil
}

func (r *Rows) TypeMap() *pgtype.Map { return pgtype.NewMap() }

type Tx struct {
	*DB
	CommitFunc   func(context.Context) error
	RollbackFunc func(context.Context) error
}

func (t *Tx) Begin(ctx context.Context) (pgx.Tx, error) { return t.DB.BeginTx(ctx, pgx.TxOptions{}) }
func (t *Tx) Commit(ctx context.Context) error          { return t.CommitFunc(ctx) }
func (t *Tx) Rollback(ctx context.Context) error        { return t.RollbackFunc(ctx) }
func (*Tx) Conn() *pgx.Conn                             { return nil }
func (*Tx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	panic("unexpected CopyFrom")
}
func (*Tx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { panic("unexpected SendBatch") }
func (*Tx) LargeObjects() pgx.LargeObjects                         { panic("unexpected LargeObjects") }
func (*Tx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	panic("unexpected Prepare")
}
