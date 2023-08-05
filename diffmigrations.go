//go:build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/technopolitica/open-mobility/devutils"
)

func main() {
	ctx := context.Background()
	dbContainer, err := devutils.StartDBContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start database: %s", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	err = dbContainer.DiffMigrations(ctx, cwd)
	if err != nil {
		log.Fatalf("failed to run migration diff: %s", err)
	}
}
