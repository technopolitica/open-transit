package e2e_tests

import (
	"context"
	"fmt"
	"net/url"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type DBContainer struct {
	container        *postgres.PostgresContainer
	ConnectionString string
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

func NewConnectionURL(host string, port nat.Port) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbConfig.Username, dbConfig.Password, host, port.Int(), dbConfig.DBName))
}

var defaultPGPort = nat.Port("5432/tcp")

func StartDBContainer(ctx context.Context) (dbContainer DBContainer, err error) {
	container, err := postgres.RunContainer(ctx,
		postgres.WithDatabase(dbConfig.DBName),
		postgres.WithUsername(dbConfig.Username),
		postgres.WithPassword(dbConfig.Password),
		testcontainers.WithImage("docker.io/postgres:14-alpine"),
		testcontainers.WithWaitStrategy(wait.ForSQL(defaultPGPort, "pgx", func(host string, port nat.Port) string {
			connectionURL, err := NewConnectionURL(host, port)
			if err != nil {
				panic(err)
			}
			return connectionURL.String()
		})),
	)
	if err != nil {
		err = fmt.Errorf("failed to initialize database server: %w", err)
		return
	}

	host, err := container.Host(ctx)
	if err != nil {
		err = fmt.Errorf("failed to determine host: %w", err)
		return
	}
	port, err := container.MappedPort(ctx, defaultPGPort)
	if err != nil {
		err = fmt.Errorf("failed to determine port: %w", err)
		return
	}
	connectionURL, err := NewConnectionURL(host, port)
	dbContainer = DBContainer{container: container, ConnectionString: connectionURL.String()}
	return
}

func (dbContainer DBContainer) Terminate(ctx context.Context) error {
	return dbContainer.container.Terminate(ctx)
}
