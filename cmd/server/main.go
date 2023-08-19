package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/technopolitica/open-transit/internal/server"
)

func loadPublicKey(publicKeyURL *url.URL) (publicKey *rsa.PublicKey, err error) {
	switch publicKeyURL.Scheme {
	case "file":
		filePath := publicKeyURL.Path
		var pemBytes []byte
		pemBytes, err = os.ReadFile(filePath)
		if err != nil {
			return
		}
		var pemBlock *pem.Block
		pemBlock, _ = pem.Decode(pemBytes)
		if pemBlock.Type != "RSA PUBLIC KEY" {
			err = fmt.Errorf("invalid public key of type %s", pemBlock.Type)
			return
		}
		publicKey, err = x509.ParsePKCS1PublicKey(pemBlock.Bytes)
		return
	default:
		err = fmt.Errorf("unsupported public key source: %s", publicKeyURL.Scheme)
		return
	}
}

var (
	dbURL     = flag.String("db-url", "", "URL-formatted connection string to the database server. Currently only postgres:// URLS are supported.")
	port      = flag.Int("port", 0, "port to listen on")
	publicKey = flag.String("public-key", "", "URL to the public key used to sign auth tokens. Currently only file:// proptocols are supported.")
)

func main() {
	ctx := context.Background()

	flag.Parse()
	if *dbURL == "" {
		log.Print("-db-url is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *publicKey == "" {
		log.Print("-public-key is required\n")
		flag.Usage()
		os.Exit(1)
	}
	publicKeyURL, err := url.Parse(*publicKey)
	if err != nil {
		log.Fatalf("failed to parse public key as URL: %s\n", err)
	}
	if publicKeyURL.Path == "" {
		log.Fatalf("public key url cannot have an empty path\n")
	}

	db, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %s\n", err)
	}
	defer db.Close()

	publicKey, err := loadPublicKey(publicKeyURL)
	if err != nil {
		log.Fatalf("failed to read public key: %s\n", err)
	}

	router := server.New(db, *publicKey)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen on specified address: %s\n", err)
	}

	done := make(chan error)
	go func() {
		done <- http.Serve(listener, router)
	}()
	log.Printf("listening on http://%s...\n", listener.Addr())
	err = <-done
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}
	fmt.Printf("shutting down...")
}
