package db

import (
	"context"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func MigrateToLatest(ctx context.Context, connectionURL string) (err error) {
	db, err := goose.OpenDBWithDriver("pgx", connectionURL)
	if err != nil {
		return fmt.Errorf("failed to connect with database: %w", err)
	}

	defer func() {
		dbErr := db.Close()
		if dbErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close database connection: %w", err)
			} else {
				err = fmt.Errorf("multiple errors occurred: %w, %s", err, dbErr)
			}
		}
	}()

	goose.SetBaseFS(embedMigrations)
	err = goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	err = goose.UpContext(ctx, db, "migrations")
	return
}
