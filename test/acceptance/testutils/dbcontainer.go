package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type DBServer struct {
	container        *postgres.PostgresContainer
	ConnectionString string
}

var dbConfig = struct {
	Username string
	Password string
}{
	Username: "postgres",
	Password: "postgres",
}

func connectionString(config pgconn.Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d?sslmode=disable", config.User, config.Password, config.Host, config.Port)
}

var defaultPGPort = nat.Port("5432/tcp")

const defaultTimeout = 15 * time.Minute

func StartDBServer(ctx context.Context) (server DBServer, err error) {
	timeout := defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = deadline.Sub(time.Now())
	}
	container, err := postgres.RunContainer(ctx,
		postgres.WithUsername(dbConfig.Username),
		postgres.WithPassword(dbConfig.Password),
		testcontainers.WithImage("docker.io/postgres:14-alpine"),
		testcontainers.WithWaitStrategyAndDeadline(timeout, wait.ForSQL(defaultPGPort, "pgx", func(host string, port nat.Port) string {
			return connectionString(pgconn.Config{Host: host, Port: uint16(port.Int()), User: dbConfig.Username, Password: dbConfig.Password})
		})),
	)
	if err != nil {
		err = fmt.Errorf("failed to start database server: %w", err)
		return
	}

	host, err := container.Host(ctx)
	if err != nil {
		err = fmt.Errorf("failed to determine host name: %w", err)
		return
	}
	port, err := container.MappedPort(ctx, defaultPGPort)
	if err != nil {
		err = fmt.Errorf("failed to determined port: %w", err)
		return
	}

	connectionURL := connectionString(pgconn.Config{Host: host, Port: uint16(port.Int()), User: dbConfig.Username, Password: dbConfig.Password})
	server = DBServer{container: container, ConnectionString: connectionURL}
	return
}

func (dbContainer DBServer) Terminate(ctx context.Context) error {
	return dbContainer.container.Terminate(ctx)
}
