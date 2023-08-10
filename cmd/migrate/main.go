package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/technopolitica/open-mobility/internal/db"
)

func main() {
	var connectionURL = flag.String("url", "", "URL-formatted connection string to the DB to operate upon")

	flag.Parse()

	if *connectionURL == "" {
		fmt.Print("missing required -url param\n")
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

	conn, err := goose.OpenDBWithDriver("pgx", *connectionURL)
	if err != nil {
		log.Fatalf("failed to connect with database: %v\n", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Fatalf("failed to close database connection: %v\n", err)
		}
	}()

	goose.SetBaseFS(db.Migrations)
	err = goose.SetDialect("postgres")
	if err != nil {
		log.Fatalf("failed to set dialect: %s", err)
	}

	arguments := []string{}
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	if err := goose.Run(command, conn, "migrations", arguments...); err != nil {
		log.Fatalf("migrate %v: %v", command, err)
	}
}
