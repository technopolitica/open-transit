package e2e_tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gexec"
	"github.com/technopolitica/open-mobility/test/testutils"
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
	DBConnectionString string
	APIBaseURL         *url.URL
	PrivateKey         *rsa.PrivateKey
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

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e")
}

var apiClient *testutils.TestClient
var dbConn *pgx.Conn

const RSA256BitSize = 128 * 8

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	format.UseStringerRepresentation = true

	DeferCleanup(gexec.CleanupBuildArtifacts)

	dbServer, err := testutils.StartDBServer(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to start database server")
	DeferCleanup(dbServer.Terminate)
	err = dbServer.MigrateToLatest(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to migrate database to latest schema version")

	privateKey, err := rsa.GenerateKey(rand.Reader, RSA256BitSize)
	Expect(err).NotTo(HaveOccurred(), "failed to generate private/public key")
	publicKeyFilePath, err := writePublicKeyFile(&privateKey.PublicKey)
	Expect(err).NotTo(HaveOccurred(), "failed to write public key file")

	apiServer, err := testutils.StartAPIServer(ctx, dbServer.ConnectionString, fmt.Sprintf("file://%s", publicKeyFilePath))
	Expect(err).NotTo(HaveOccurred(), "failed to start server")
	DeferCleanup(apiServer.Terminate)

	suiteData := suiteData{
		DBConnectionString: dbServer.ConnectionString,
		APIBaseURL:         apiServer.BaseURL,
		PrivateKey:         privateKey,
	}
	output, err := suiteData.Encode()
	Expect(err).NotTo(HaveOccurred(), "failed to encode suite data")
	return output
}, func(ctx context.Context, data []byte) {
	var suiteData suiteData
	err := suiteData.Decode(data)
	Expect(err).NotTo(HaveOccurred(), "failed to decode suite data")

	dbConn, err = pgx.Connect(ctx, suiteData.DBConnectionString)
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
