package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) BeginImmediate(ctx context.Context) (*sql.Tx, error) {
	if db, ok := q.db.(*sql.DB); ok {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, "BEGIN IMMEDIATE")
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		return tx, nil
	}
	return nil, fmt.Errorf("underlying DBTX is not a *sql.DB")
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}
