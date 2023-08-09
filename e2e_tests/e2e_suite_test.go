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
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/technopolitica/open-mobility/e2e_tests/testutils"
	"github.com/technopolitica/open-mobility/server"
)

type TestServer struct {
	dbContainer DBContainer
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

func StartTestServer(ctx context.Context, publicKey rsa.PublicKey) (testServer TestServer, err error) {
	dbContainer, err := StartDBContainer(ctx)
	if err != nil {
		return
	}
	err = dbContainer.MigrateToLatest(ctx)
	if err != nil {
		err = fmt.Errorf("failed to migrate DB: %w", err)
		return
	}
	dbConnectionURL, err := dbContainer.ExternalConnectionURL(ctx)
	if err != nil {
		err = fmt.Errorf("failed to obtain DB connection info: %w", err)
		return
	}
	db, err := pgxpool.New(ctx, dbConnectionURL.String())
	if err != nil {
		err = fmt.Errorf("failed to connect to DB: %w", err)
		return
	}
	testServer = TestServer{
		dbContainer: dbContainer,
		httpServer:  httptest.NewServer(server.New(db, publicKey)),
	}
	err = testServer.initialize(ctx)
	if err != nil {
		err = fmt.Errorf("failed to initialize server: %w", err)
	}
	return
}

func (server TestServer) initialize(ctx context.Context) (err error) {
	dbConnectionURL, err := server.DBConnectionURL(ctx)
	if err != nil {
		return
	}
	db, err := pgx.Connect(ctx, dbConnectionURL.String())
	if err != nil {
		err = fmt.Errorf("failed to open DB connection: %s", err)
		return
	}
	_, err = db.Exec(ctx, fmt.Sprintf("CREATE DATABASE original_%[1]s WITH TEMPLATE %[1]s", dbConnectionURL.DBName))
	if err != nil {
		err = fmt.Errorf("failed to create test database copy")
		return
	}
	return
}

func (server TestServer) DBConnectionURL(ctx context.Context) (connectionURL ConnectionURL, err error) {
	return server.dbContainer.ExternalConnectionURL(ctx)
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
	DBConnectionURL ConnectionURL
	BaseURL         url.URL
	PrivateKey      x509EncodedKey
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e")
}

var dbname string
var dbConn *pgx.Conn
var apiClient *testutils.TestClient

const RSA256BitSize = 128 * 8

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	format.UseStringerRepresentation = true

	privateKey, err := rsa.GenerateKey(rand.Reader, RSA256BitSize)
	Expect(err).NotTo(HaveOccurred())

	testServer, err := StartTestServer(ctx, privateKey.PublicKey)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func(ctx context.Context) {
		dbConn.Close(ctx)
		testServer.Close(ctx)
	})

	dbConnectionURL, err := testServer.DBConnectionURL(ctx)
	Expect(err).NotTo(HaveOccurred())
	data, err := json.Marshal(setupData{
		DBConnectionURL: dbConnectionURL,
		BaseURL:         *testServer.BaseURL(),
		PrivateKey:      x509EncodedKey{*privateKey},
	})
	Expect(err).NotTo(HaveOccurred())
	return data
}, func(ctx context.Context, data []byte) {
	var sd setupData
	err := json.Unmarshal(data, &sd)
	Expect(err).NotTo(HaveOccurred())

	conn, err := pgx.Connect(ctx, sd.DBConnectionURL.String())
	Expect(err).NotTo(HaveOccurred())
	dbConn = conn
	dbname = sd.DBConnectionURL.DBName

	apiClient = testutils.NewTestClient(sd.BaseURL, sd.PrivateKey.PrivateKey)
})

func ClearData(ctx context.Context, dbConn *pgx.Conn) (err error) {
	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", dbname))
	if err != nil {
		return
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("CREATE DATABASE %[1]s WITH TEMPLATE original_%[1]s", dbname))
	return
}

var _ = AfterEach(OncePerOrdered, func(ctx context.Context) {
	ClearData(ctx, dbConn)
	apiClient.Reset()
})
