package testutils

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jackc/pgx/v5"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

func migrateToLatest(ctx context.Context, migrateBinaryPath string, dbURL string) (err error) {
	migrateCmd := exec.Command(
		migrateBinaryPath,
		"-db-url", dbURL,
		"migrate",
		"-to", "latest")
	session, err := gexec.Start(migrateCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		err = fmt.Errorf("failed to run command: %w", err)
		return
	}
	select {
	case <-session.Exited:
		if session.ExitCode() != 0 {
			err = fmt.Errorf("exited with non-zero code %d", session.ExitCode())
			return
		}
	case <-ctx.Done():
		err = fmt.Errorf("context cancelled: %w", context.Cause(ctx))
		return
	}
	return
}

type DBClient struct {
	conn *pgx.Conn
}

type TestDB struct {
	Name             string
	ConnectionString string
}

func NewDBClient(ctx context.Context, connString string) (client *DBClient, err error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		err = fmt.Errorf("failed to connect to database: %w", err)
		return
	}
	client = &DBClient{
		conn: conn,
	}
	return
}

const sourceDBName = "_original"

func (client DBClient) InitializeSourceDB(ctx context.Context, migrateBinaryPath string) (err error) {
	_, err = client.conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, sourceDBName))
	if err != nil {
		err = fmt.Errorf("failed to create database: %w", err)
		return
	}
	dbConfig := client.conn.Config().Copy()
	dbConfig.Database = sourceDBName
	err = migrateToLatest(ctx, migrateBinaryPath, connectionString(dbConfig.Config))
	if err != nil {
		err = fmt.Errorf("failed to migrate database to latest schema version: %w", err)
	}
	return
}

func (client DBClient) CleanupSourceDB(ctx context.Context) (err error) {
	_, err = client.conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, sourceDBName))
	if err != nil {
		err = fmt.Errorf("failed to drop source database: %w", err)
	}
	return
}

func (client DBClient) CreateTestDB(ctx context.Context) (testDB TestDB, err error) {
	testDB.Name = GenerateRandomUUID().String()
	config := client.conn.Config().Copy()
	config.Database = testDB.Name
	testDB.ConnectionString = connectionString(config.Config)
	err = client.ResetTestDB(ctx, testDB.Name)
	return
}

func (client DBClient) CleanupTestDB(ctx context.Context, testDBName string) (err error) {
	_, err = client.conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, testDBName))
	if err != nil {
		err = fmt.Errorf("failed to drop database: %w", err)
	}
	return
}

func (client DBClient) ResetTestDB(ctx context.Context, testDBName string) error {
	err := client.CleanupTestDB(ctx, testDBName)
	if err != nil {
		return err
	}
	_, err = client.conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s" WITH TEMPLATE "%s"`, testDBName, sourceDBName))
	if err != nil {
		return fmt.Errorf("failed to copy database from source: %w", err)
	}
	return nil
}

func (client DBClient) Close(ctx context.Context) error {
	return client.conn.Close(ctx)
}
