//go:generate go run github.com/kyleconroy/sqlc/cmd/sqlc@v1.19.1 generate

package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/technopolitica/open-mobility/server"
)

func loadPublicKey() (publicKey *rsa.PublicKey, err error) {
	encodedPublicKey := os.Getenv("PUBLIC_KEY")
	return x509.ParsePKCS1PublicKey([]byte(encodedPublicKey))
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}
	defer db.Close()
	publicKey, err := loadPublicKey()
	if err != nil {
		log.Fatalf("failed to read public key: %s", err)
	}
	router := server.New(db, *publicKey)
	http.ListenAndServe(":3000", router)
}
