package e2e_tests

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/technopolitica/open-mobility/internal/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type DBContainer struct {
	container *postgres.PostgresContainer
}

var dbConfig = struct {
	Username string
	Password string
	DBName   string
}{
	Username: "postgres",
	Password: "postgres",
	DBName:   "test",
}

type ConnectionURL struct {
	Host     string
	Port     nat.Port
	Username string
	Password string
	DBName   string
}

func NewConnectionURL(host string, port nat.Port) ConnectionURL {
	return ConnectionURL{
		Host:     host,
		Port:     port,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
		DBName:   dbConfig.DBName,
	}
}

func (u ConnectionURL) String() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", u.Username, u.Password, u.Host, u.Port.Int(), u.DBName)
}

var defaultPGPort = nat.Port("5432/tcp")

func StartDBContainer(ctx context.Context) (dbContainer DBContainer, err error) {
	container, err := postgres.RunContainer(ctx,
		postgres.WithDatabase(dbConfig.DBName),
		postgres.WithUsername(dbConfig.Username),
		postgres.WithPassword(dbConfig.Password),
		testcontainers.WithImage("docker.io/postgres:14-alpine"),
		testcontainers.WithWaitStrategy(wait.ForSQL(defaultPGPort, "pgx", func(host string, port nat.Port) string {
			return NewConnectionURL(host, port).String()
		})),
	)
	if err != nil {
		err = fmt.Errorf("failed to initialize database server: %w", err)
		return
	}
	dbContainer = DBContainer{container}
	return
}

func (dbContainer DBContainer) ExternalConnectionURL(ctx context.Context) (connectionURL ConnectionURL, err error) {
	host, err := dbContainer.container.Host(ctx)
	if err != nil {
		return
	}
	port, err := dbContainer.container.MappedPort(ctx, defaultPGPort)
	if err != nil {
		return
	}
	connectionURL = NewConnectionURL(host, port)
	return
}

func (dbContainer DBContainer) MigrateToLatest(ctx context.Context) (err error) {
	connectionURL, err := dbContainer.ExternalConnectionURL(ctx)
	if err != nil {
		err = fmt.Errorf("failed to retrieve DB connection info: %w", err)
		return
	}
	err = db.MigrateToLatest(ctx, connectionURL.String())
	return
}

func (dbContainer DBContainer) Terminate(ctx context.Context) error {
	return dbContainer.container.Terminate(ctx)
}
