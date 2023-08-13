package db

import (
	"context"
	"embed"
	"fmt"
	"strconv"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var Migrations embed.FS

func MigrateTo(ctx context.Context, connectionURL string, version string) (err error) {
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

	goose.SetBaseFS(Migrations)
	err = goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	if version == "latest" {
		err = goose.UpContext(ctx, db, "migrations")
		return
	}
	versionInt, err := strconv.ParseInt(version, 10, 0)
	if err != nil {
		err = fmt.Errorf("failed to parse version: %w", err)
		return
	}
	err = goose.UpToContext(ctx, db, "migrations", versionInt)
	return
}
