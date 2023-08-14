package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

type DBConnection interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
	LoadType(ctx context.Context, typeName string) (*pgtype.Type, error)
	TypeMap() *pgtype.Map
}

type Repository struct {
	DBConnection
}

func registerTypes(ctx context.Context, conn DBConnection, typeNames []string) error {
	for _, typeName := range typeNames {
		dataType, err := conn.LoadType(ctx, typeName)
		if err != nil {
			return err
		}
		conn.TypeMap().RegisterType(dataType)
	}
	return nil
}

func NewRepository(ctx context.Context, conn DBConnection) (Repository, error) {
	return Repository{conn}, nil
}

func (repo Repository) WithinTransaction(ctx context.Context, op func(pgx.Tx) error) (err error) {
	tx, err := repo.DBConnection.Begin(ctx)
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
