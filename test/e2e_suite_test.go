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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func findOpenPort() (addr *net.TCPAddr, err error) {
	addr, err = net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}
	defer listener.Close()
	addr = listener.Addr().(*net.TCPAddr)
	return
}

func pingHealthEndpoint(baseURL *url.URL) (err error) {
	healthEndpoint := baseURL.JoinPath("health")
	res, err := http.Get(healthEndpoint.String())
	if err == nil && res.StatusCode != http.StatusOK {
		err = fmt.Errorf("got unexpected http status code in response: %s", res.Status)
	}
	return
}

type suiteData struct {
	DBConnectionURL string
	APIBaseURL      *url.URL
	PrivateKey      *rsa.PrivateKey
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

	dbContainer, err := StartDBContainer(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to start DB")
	dbConnectionString := dbContainer.ConnectionString

	migrateBinaryPath, err := gexec.Build("github.com/technopolitica/open-mobility/cmd/migrate")
	Expect(err).NotTo(HaveOccurred(), "failed to build migrate binary")
	migrateCmd := exec.Command(
		migrateBinaryPath,
		"-db-url", dbConnectionString,
		"migrate",
		"-to", "latest")
	migrateSession, err := gexec.Start(migrateCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred(), "failed to start migration command")
	Eventually(migrateSession).Should(gexec.Exit(0))

	privateKey, err := rsa.GenerateKey(rand.Reader, RSA256BitSize)
	Expect(err).NotTo(HaveOccurred(), "failed to generate private/public key")
	publicKeyFilePath, err := writePublicKeyFile(&privateKey.PublicKey)
	Expect(err).NotTo(HaveOccurred(), "failed to write public key file")

	serverBinaryPath, err := gexec.Build("github.com/technopolitica/open-mobility/cmd/server")
	Expect(err).NotTo(HaveOccurred(), "failed to build server binary")

	addr, err := findOpenPort()
	Expect(err).NotTo(HaveOccurred(), "failed to find open port")
	serverCmd := exec.Command(
		serverBinaryPath,
		"-port", fmt.Sprint(addr.Port),
		"-db-url", dbConnectionString,
		"-public-key", fmt.Sprintf("file://%s", publicKeyFilePath),
	)
	serverSession, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred(), "failed to start server")
	DeferCleanup(func() {
		serverSession.Terminate().Wait()
	})
	baseURL, err := url.Parse(fmt.Sprintf("http://%s", addr.String()))
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error { return pingHealthEndpoint(baseURL) }).Should(Succeed())

	data := suiteData{
		DBConnectionURL: dbConnectionString,
		APIBaseURL:      baseURL,
		PrivateKey:      privateKey,
	}
	output := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(output)
	err = encoder.Encode(data)
	Expect(err).NotTo(HaveOccurred(), "failed to encode suite data")
	return output.Bytes()
}, func(ctx context.Context, data []byte) {
	decoder := gob.NewDecoder(bytes.NewBuffer(data))
	var suiteData suiteData
	err := decoder.Decode(&suiteData)
	Expect(err).NotTo(HaveOccurred(), "failed to decode suite data")

	dbConn, err = pgx.Connect(ctx, suiteData.DBConnectionURL)
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
