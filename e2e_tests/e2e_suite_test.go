package e2e_tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
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
	"github.com/technopolitica/open-mobility/e2e_tests/testutils"
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

func NewTestServer(ctx context.Context, publicKey rsa.PublicKey) (testServer TestServer, err error) {
	db, dbContainer, err := initDatabase(ctx)
	if err != nil {
		return
	}
	testServer = TestServer{
		dbContainer: dbContainer,
		httpServer:  httptest.NewServer(server.New(db, publicKey)),
	}
	return
}

type x509EncodedKey struct {
	rsa.PrivateKey
}

func (sd x509EncodedKey) MarshalJSON() (data []byte, err error) {
	encodedKey := x509.MarshalPKCS1PrivateKey(&sd.PrivateKey)
	if err != nil {
		return
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(encodedKey))
}

func (sd *x509EncodedKey) UnmarshalJSON(data []byte) (err error) {
	var base64EncodedKey string
	err = json.Unmarshal(data, &base64EncodedKey)
	if err != nil {
		return
	}
	encodedKey, err := base64.StdEncoding.DecodeString(base64EncodedKey)
	if err != nil {
		return
	}
	parsedKey, err := x509.ParsePKCS1PrivateKey(encodedKey)
	if err != nil {
		return
	}
	*sd = x509EncodedKey(x509EncodedKey{*parsedKey})
	return
}

type setupData struct {
	DBConnectionString string
	BaseURL            url.URL
	PrivateKey         x509EncodedKey
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e")
}

var dbConn *pgx.Conn
var apiClient *testutils.TestClient

const RSA256BitSize = 128 * 8

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	privateKey, err := rsa.GenerateKey(rand.Reader, RSA256BitSize)
	Expect(err).NotTo(HaveOccurred())

	testServer, err := NewTestServer(ctx, privateKey.PublicKey)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func(ctx context.Context) {
		dbConn.Close(ctx)
		testServer.Close(ctx)
	})

	cs, err := testServer.DBConnectionString()
	Expect(err).NotTo(HaveOccurred())
	data, err := json.Marshal(setupData{
		DBConnectionString: cs,
		BaseURL:            *testServer.BaseURL(),
		PrivateKey:         x509EncodedKey{*privateKey},
	})
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(ctx context.Context, data []byte) {
	var sd setupData
	err := json.Unmarshal(data, &sd)
	Expect(err).NotTo(HaveOccurred())

	conn, err := pgx.Connect(ctx, sd.DBConnectionString)
	Expect(err).NotTo(HaveOccurred())
	dbConn = conn

	apiClient = testutils.NewTestClient(sd.BaseURL, sd.PrivateKey.PrivateKey)
})

func ClearData(ctx context.Context, dbConn *pgx.Conn) (err error) {
	tx, err := dbConn.BeginTx(ctx, pgx.TxOptions{})
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

var _ = AfterEach(OncePerOrdered, func(ctx context.Context) {
	ClearData(ctx, dbConn)
	apiClient.Reset()
})
