package e2e_tests

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/technopolitica/open-mobility/server"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var dbConfig = struct {
	username string
	password string
	port     nat.Port
	dbname   string
}{
	username: "postgres",
	password: "postgres",
	port:     nat.Port("5432/tcp"),
	dbname:   "test",
}

func getConnectionString(host string, port nat.Port) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbConfig.username, dbConfig.password, host, port.Int(), dbConfig.dbname)
}

func startDbContainer(ctx context.Context) (container *postgres.PostgresContainer, connectionString string, err error) {
	container, err = postgres.RunContainer(ctx,
		postgres.WithDatabase(dbConfig.dbname),
		postgres.WithUsername(dbConfig.username),
		postgres.WithPassword(dbConfig.password),
		testcontainers.WithImage("docker.io/postgres:14-alpine"),
		testcontainers.WithWaitStrategy(wait.ForSQL(dbConfig.port, "pgx", func(host string, port nat.Port) string {
			return getConnectionString(host, port)
		})),
	)
	if err != nil {
		err = fmt.Errorf("failed to initialize database server: %s", err)
		return
	}
	host, err := container.ContainerIP(ctx)
	if err != nil {
		err = fmt.Errorf("failed to fetch db container host: %s", err)
		return
	}
	connectionString = getConnectionString(host, dbConfig.port)
	return
}

func initDatabase(ctx context.Context) (db *pgxpool.Pool, dbContainer *postgres.PostgresContainer, err error) {
	dbContainer, dbConnectionString, err := startDbContainer(ctx)
	if err != nil {
		return
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("failed to detect working directory: %s", err)
		return
	}
	_, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "arigaio/atlas:0.12.0-alpine",
			Cmd: []string{
				"migrate", "apply",
				"--url", dbConnectionString,
				"--dir", "file:///migrations",
			},
			Mounts: testcontainers.ContainerMounts{
				testcontainers.ContainerMount{
					Source: testcontainers.GenericBindMountSource{
						HostPath: fmt.Sprintf("%s/../migrations", workingDirectory),
					},
					Target: "/migrations",
				},
			},
			WaitingFor: wait.ForExit(),
		},
		Started: true,
	})
	if err != nil {
		err = fmt.Errorf("failed to apply schema migration: %s", err)
		return
	}

	hostConnectionString, err := dbContainer.ConnectionString(ctx)
	if err != nil {
		err = fmt.Errorf("failed to fetch host connection string %s", err)
		return
	}
	db, err = pgxpool.New(ctx, hostConnectionString)
	if err != nil {
		err = fmt.Errorf("failed to open DB connection: %s", err)
		return
	}
	_, err = db.Exec(ctx, fmt.Sprintf("CREATE DATABASE original_%s WITH TEMPLATE %s", dbConfig.dbname, dbConfig.dbname))
	if err != nil {
		err = fmt.Errorf("failed to create test database copy")
		return
	}

	return
}

type TestServer struct {
	dbContainer *postgres.PostgresContainer
	httpServer  *httptest.Server
}

func (ts TestServer) Close(ctx context.Context) {
	ts.dbContainer.Terminate(ctx)
	ts.httpServer.Close()
}

func (ts TestServer) BaseURL() *url.URL {
	url, err := url.Parse(ts.httpServer.URL)
	if err != nil {
		panic(fmt.Sprintf("invalid url %s", err))
	}
	return url
}

func (ts TestServer) DBConnectionString() (string, error) {
	return ts.dbContainer.ConnectionString(context.Background())
}

func NewTestServer(ctx context.Context) (testServer TestServer, err error) {
	db, dbContainer, err := initDatabase(ctx)
	if err != nil {
		return
	}
	testServer = TestServer{
		dbContainer: dbContainer,
		httpServer:  httptest.NewServer(server.New(db)),
	}
	return
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e")
}

var dbConn *pgx.Conn
var apiClient *TestClient

var _ = SynchronizedBeforeSuite(func() []byte {
	ctx := context.Background()
	testServer, err := NewTestServer(ctx)
	Expect(err).NotTo(HaveOccurred())
	baseURL := testServer.BaseURL()
	apiClient = NewTestClient(*baseURL)

	DeferCleanup(func() {
		testServer.Close(ctx)
	})

	cs, err := testServer.DBConnectionString()
	Expect(err).NotTo(HaveOccurred())
	return []byte(cs)
}, func(data []byte) {
	connectionString := string(data)
	conn, err := pgx.Connect(context.Background(), connectionString)
	Expect(err).NotTo(HaveOccurred())
	dbConn = conn
})

func ClearData(dbConn *pgx.Conn) (err error) {
	ctx := context.Background()
	tx, err := dbConn.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", dbConfig.dbname))
	if err != nil {
		return
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE original_%s", dbConfig.dbname, dbConfig.dbname))
	return
}

var _ = AfterEach(func() {
	ClearData(dbConn)
})
