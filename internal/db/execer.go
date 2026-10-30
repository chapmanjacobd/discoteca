package db

import (
	"context"
	"database/sql"
)

// dbExecer is the subset of [sql.DB] and [sql.Conn] used by migrations.
//
// Migrations run on a single [sql.Conn] so connection-scoped PRAGMAs such as
// foreign_keys apply to every statement, including statements inside a
// transaction.
type dbExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// dbQueryer is the read-only subset implemented by [sql.DB], [sql.Conn], and
// [sql.Tx].
type dbQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
