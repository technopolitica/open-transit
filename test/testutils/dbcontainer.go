package testutils

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/docker/go-connections/nat"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
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
	DBName   string
}{
	Username: "postgres",
	Password: "postgres",
	DBName:   "test",
}

func newConnectionString(host string, port nat.Port) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbConfig.Username, dbConfig.Password, host, port.Int(), dbConfig.DBName)
}

var defaultPGPort = nat.Port("5432/tcp")

func StartDBServer(ctx context.Context) (server DBServer, err error) {
	container, err := postgres.RunContainer(ctx,
		postgres.WithDatabase(dbConfig.DBName),
		postgres.WithUsername(dbConfig.Username),
		postgres.WithPassword(dbConfig.Password),
		testcontainers.WithImage("docker.io/postgres:14-alpine"),
		testcontainers.WithWaitStrategy(wait.ForSQL(defaultPGPort, "pgx", newConnectionString)),
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

	connectionURL := newConnectionString(host, port)
	server = DBServer{container: container, ConnectionString: connectionURL}
	return
}

func (dbContainer DBServer) MigrateToLatest(ctx context.Context) (err error) {
	migrateBinaryPath, err := gexec.Build("github.com/technopolitica/open-mobility/cmd/migrate")
	if err != nil {
		err = fmt.Errorf("failed to build binary: %w", err)
		return
	}
	migrateCmd := exec.Command(
		migrateBinaryPath,
		"-db-url", dbContainer.ConnectionString,
		"migrate",
		"-to", "latest")
	session, err := gexec.Start(migrateCmd, GinkgoWriter, GinkgoWriter)
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

func (dbContainer DBServer) Terminate(ctx context.Context) error {
	return dbContainer.container.Terminate(ctx)
}
