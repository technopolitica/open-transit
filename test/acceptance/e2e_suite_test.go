package acceptance

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gexec"
	"github.com/technopolitica/open-transit/test/acceptance/testutils"
)

func writePublicKeyFile(publicKey *rsa.PublicKey) (filePath string, err error) {
	file, err := os.CreateTemp("", "open-transit-public-key*.pem")
	if err != nil {
		return
	}
	defer file.Close()
	serializedKey := x509.MarshalPKCS1PublicKey(publicKey)
	err = pem.Encode(file, &pem.Block{Type: "RSA PUBLIC KEY", Bytes: serializedKey})
	filePath = file.Name()
	return
}

type suiteData struct {
	APIBaseURL *url.URL
	PrivateKey *rsa.PrivateKey
}

func (dat *suiteData) Encode() (output []byte, err error) {
	outputBuf := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(outputBuf)
	err = encoder.Encode(dat)
	if err != nil {
		return
	}
	output = outputBuf.Bytes()
	return
}

func (dat *suiteData) Decode(data []byte) (err error) {
	decoder := gob.NewDecoder(bytes.NewBuffer(data))
	err = decoder.Decode(&dat)
	return
}

var dbURL, migrateBinaryPath, serverBinaryPath string

func init() {
	flag.StringVar(
		&dbURL,
		"db-url",
		"",
		"URL-encoded connection string to a postgres database. If not provided a postgres docker container will be started",
	)
	flag.StringVar(
		&migrateBinaryPath,
		"migrate-bin",
		"",
		"Path to the built migrate binary. If not provided a binary will be built automatically.",
	)
	flag.StringVar(
		&serverBinaryPath,
		"server-bin",
		"",
		"Path to the built server binary. If not provided a binary will be built automatically.",
	)
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Tests", decorators.Label("integration"))
}

var apiClient *testutils.TestClient
var dbConn *pgx.Conn

const RSA256BitSize = 128 * 8

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	format.UseStringerRepresentation = true

	DeferCleanup(gexec.CleanupBuildArtifacts)

	if dbURL == "" {
		dbServer, err := testutils.StartDBServer(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to start database server")
		DeferCleanup(dbServer.Terminate)
		dbURL = dbServer.ConnectionString
	}

	if migrateBinaryPath == "" {
		var err error
		migrateBinaryPath, err = gexec.Build("github.com/technopolitica/open-transit/cmd/open-transit-migrate")
		Expect(err).NotTo(HaveOccurred(), "failed to build migrate binary")
	}

	err := testutils.MigrateToLatest(ctx, migrateBinaryPath, dbURL)
	Expect(err).NotTo(HaveOccurred(), "failed to migrate database to latest schema version")

	privateKey, err := rsa.GenerateKey(rand.Reader, RSA256BitSize)
	Expect(err).NotTo(HaveOccurred(), "failed to generate private/public key")
	publicKeyFilePath, err := writePublicKeyFile(&privateKey.PublicKey)
	Expect(err).NotTo(HaveOccurred(), "failed to write public key file")

	if serverBinaryPath == "" {
		var err error
		serverBinaryPath, err = gexec.Build("github.com/technopolitica/open-transit/cmd/open-transit-server")
		Expect(err).NotTo(HaveOccurred(), "failed to build server binary")
	}
	apiServer, err := testutils.StartAPIServer(ctx, serverBinaryPath, dbURL, fmt.Sprintf("file://%s", publicKeyFilePath))
	Expect(err).NotTo(HaveOccurred(), "failed to start server")
	DeferCleanup(apiServer.Terminate)

	suiteData := suiteData{
		APIBaseURL: apiServer.BaseURL,
		PrivateKey: privateKey,
	}
	output, err := suiteData.Encode()
	Expect(err).NotTo(HaveOccurred(), "failed to encode suite data")
	return output
}, func(ctx context.Context, data []byte) {
	var suiteData suiteData
	err := suiteData.Decode(data)
	Expect(err).NotTo(HaveOccurred(), "failed to decode suite data")

	dbConn, err = pgx.Connect(ctx, dbURL)
	Expect(err).NotTo(HaveOccurred(), "failed to connect to DB")
	DeferCleanup(dbConn.Close)

	apiClient = testutils.NewTestClient(*suiteData.APIBaseURL, *suiteData.PrivateKey)
})

var _ = BeforeEach(OncePerOrdered, func(ctx context.Context) {
	// Start a transaction for every single test and rollback at the end of each test
	// to ensure isolation.
	tx, err := dbConn.Begin(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to start DB transaction")
	DeferCleanup(tx.Rollback)
})

var _ = AfterEach(OncePerOrdered, func(ctx context.Context) {
	apiClient.Unauthenticate()
})
