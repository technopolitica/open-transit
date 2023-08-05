package devutils

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/docker/go-connections/nat"
	_ "github.com/jackc/pgx/v5/stdlib"
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

func (dbContainer DBContainer) internalConnectionURL(ctx context.Context) (connectionURL ConnectionURL, err error) {
	host, err := dbContainer.container.ContainerIP(ctx)
	if err != nil {
		err = fmt.Errorf("failed to fetch db container host: %w", err)
		return
	}
	connectionURL = NewConnectionURL(host, defaultPGPort)
	return
}

func (dbContainer DBContainer) DiffMigrations(ctx context.Context, rootDir string) (err error) {
	internalConnectionURL, err := dbContainer.internalConnectionURL(ctx)
	if err != nil {
		err = fmt.Errorf("failed to diff migrations: %w", err)
		return
	}
	return dbContainer.runAtlasCommand(ctx, rootDir, []string{
		"migrate", "diff",
		"--to", "file:///schema.sql",
		"--dev-url", internalConnectionURL.String(),
	})
}

func (dbContainer DBContainer) MigrateToLatest(ctx context.Context, rootDir string) (err error) {
	internalConnectionURL, err := dbContainer.internalConnectionURL(ctx)
	if err != nil {
		err = fmt.Errorf("failed to diff migrations: %w", err)
		return
	}
	return dbContainer.runAtlasCommand(ctx, rootDir, []string{
		"migrate", "apply",
		"--url", internalConnectionURL.String(),
		"--dir", "file:///migrations",
	})
}

func (dbContainer DBContainer) runAtlasCommand(ctx context.Context, rootDir string, args []string) (err error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "arigaio/atlas:0.12.0-alpine",
			Cmd:   args,
			Mounts: testcontainers.ContainerMounts{
				testcontainers.ContainerMount{
					Source: testcontainers.GenericBindMountSource{
						HostPath: fmt.Sprintf("%s/migrations", rootDir),
					},
					Target: "/migrations",
				},
				testcontainers.ContainerMount{
					Source: testcontainers.GenericBindMountSource{
						HostPath: fmt.Sprintf("%s/schema.sql", rootDir),
					},
					Target: "/schema.sql",
				},
			},
			WaitingFor: wait.ForExit(),
		},
		Started: true,
	})
	if err != nil {
		return
	}

	logs, err := container.Logs(ctx)
	if err != nil {
		return
	}
	output, err := io.ReadAll(logs)
	if err != nil {
		return
	}
	log.Print(string(output))
	state, err := container.State(ctx)
	if err != nil {
		return
	}
	if state.ExitCode != 0 {
		return fmt.Errorf("got non-zero exit code: %d", state.ExitCode)
	}
	return
}

func (dbContainer DBContainer) Terminate(ctx context.Context) error {
	return dbContainer.container.Terminate(ctx)
}
