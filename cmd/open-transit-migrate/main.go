package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/technopolitica/open-transit/internal/db"
)

var connectionURL = flag.String("db-url", "", "URL-formatted connection string to the DB to operate upon")

func main() {
	ctx := context.Background()

	flag.Parse()

	if *connectionURL == "" {
		fmt.Print("missing required -db-url param\n")
		flag.Usage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Print("expected a subcommand\n")
		flag.Usage()
		os.Exit(1)
	}
	command := args[0]

	if command != "migrate" {
		fmt.Printf("unknown subcommand \"%s\"\n", command)
		flag.Usage()
		os.Exit(1)
	}
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	version := migrateCmd.String("to", "", "version to which the database should be migrated. May specify \"latest\" to migrate to the latest version.")

	migrateCmd.Parse(args[1:])

	if *version == "" {
		fmt.Print("missing required parameter -to\n")
		migrateCmd.Usage()
		os.Exit(1)
	}
	err := db.MigrateTo(ctx, *connectionURL, *version)
	if err != nil {
		log.Fatalf("failed to run migration: %s\n", err)
	}
}
