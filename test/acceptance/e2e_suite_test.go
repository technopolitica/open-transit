package acceptance

import (
	"context"
	"flag"
	"fmt"
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
var dbClient *testutils.DBClient
var testDBName string

func validateDBURL(url string) error {
	dbConfig, err := pgx.ParseConfig(url)
	if err != nil {
		return err
	}
	if dbConfig.Database != "" {
		return fmt.Errorf("database should not be specified")
	}
	return nil
}

var _ = SynchronizedBeforeSuite(func(ctx context.Context) {
	format.UseStringerRepresentation = true

	if dbURL == "" {
		dbServer, err := testutils.StartDBServer(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to start database server")
		DeferCleanup(dbServer.Terminate)
		dbURL = dbServer.ConnectionString
	}
	err := validateDBURL(dbURL)
	Expect(err).NotTo(HaveOccurred(), "invalid db-url")

	if migrateBinaryPath == "" {
		var err error
		migrateBinaryPath, err = gexec.Build("github.com/technopolitica/open-transit/cmd/open-transit-migrate")
		Expect(err).NotTo(HaveOccurred(), "failed to build migrate binary")
		DeferCleanup(gexec.CleanupBuildArtifacts)
	}

	if serverBinaryPath == "" {
		var err error
		serverBinaryPath, err = gexec.Build("github.com/technopolitica/open-transit/cmd/open-transit-server")
		Expect(err).NotTo(HaveOccurred(), "failed to build server binary")
		DeferCleanup(gexec.CleanupBuildArtifacts)
	}

	dbClient, err := testutils.NewDBClient(ctx, dbURL)
	Expect(err).NotTo(HaveOccurred(), "failed to connect to database")
	DeferCleanup(dbClient.Close)

	err = dbClient.InitializeSourceDB(ctx, migrateBinaryPath)
	Expect(err).NotTo(HaveOccurred(), "failed to initialize source database")
	DeferCleanup(dbClient.CleanupSourceDB)
}, func(ctx context.Context) {
	var err error
	dbClient, err = testutils.NewDBClient(ctx, dbURL)
	Expect(err).NotTo(HaveOccurred(), "failed to create database client")
	DeferCleanup(dbClient.Close)

	testDB, err := dbClient.CreateTestDB(ctx)
	testDBName = testDB.Name
	DeferCleanup(func(ctx context.Context) error {
		return dbClient.CleanupTestDB(ctx, testDBName)
	})

	apiServer, err := testutils.StartAPIServer(ctx, serverBinaryPath, testDB.ConnectionString)
	Expect(err).NotTo(HaveOccurred(), "failed to start server")
	DeferCleanup(apiServer.Terminate)

	apiClient = testutils.NewTestClient(*apiServer.BaseURL, *apiServer.PrivateKey)
})

var _ = AfterEach(OncePerOrdered, func(ctx context.Context) {
	dbClient.ResetTestDB(ctx, testDBName)
	apiClient.Unauthenticate()
})
