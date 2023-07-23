//go:generate atlas migrate diff --to file://schema.sql --dev-url docker://postgres/14/test
//go:generate go run github.com/kyleconroy/sqlc/cmd/sqlc@v1.19.1 generate

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/technopolitica/open-mobility/server"
)

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}
	defer db.Close()
	router := server.New(db)
	http.ListenAndServe(":3000", router)
}
