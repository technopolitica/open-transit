package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

type DBConnection interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

type queries struct {
	DBConnection
}

func (q queries) WithinTransaction(ctx context.Context, op func(pgx.Tx) error) (err error) {
	tx, err := q.DBConnection.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	err = op(tx)
	if err != nil {
		return
	}
	err = tx.Commit(ctx)
	if err != nil {
		return
	}
	return
}
