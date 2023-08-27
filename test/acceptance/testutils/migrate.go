package testutils

import (
	"context"
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

func MigrateToLatest(ctx context.Context, migrateBinaryPath string, dbURL string) (err error) {
	migrateCmd := exec.Command(
		migrateBinaryPath,
		"-db-url", dbURL,
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
