package testutils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

type APIServer struct {
	BaseURL *url.URL
	session *gexec.Session
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

func (server APIServer) waitToAcceptConnections(ctx context.Context) (err error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	done := make(chan error)
	go func() {
		var connErr error
		for {
			select {
			case <-ticker.C:
				connErr = server.pingHealthEndpoint()
				if connErr == nil {
					done <- nil
					return
				}
			case <-ctx.Done():
				done <- fmt.Errorf("cancelled (%s), last error: %w", context.Cause(ctx), connErr)
				return
			}
		}
	}()
	err = <-done
	return
}

func (server APIServer) pingHealthEndpoint() (err error) {
	healthEndpoint := server.BaseURL.JoinPath("health")
	res, err := http.Get(healthEndpoint.String())
	if err == nil && res.StatusCode != http.StatusOK {
		err = fmt.Errorf("got unexpected http status code in response: %s", res.Status)
	}
	return
}

func StartAPIServer(ctx context.Context, dbConnectionString string, publicKey string) (server APIServer, err error) {
	serverBinaryPath, err := gexec.Build("github.com/technopolitica/open-transit/cmd/server")
	if err != nil {
		err = fmt.Errorf("failed to build server binary: %w", err)
		return
	}

	addr, err := findOpenPort()
	if err != nil {
		err = fmt.Errorf("failed to find open port: %w", err)
		return
	}
	serverCmd := exec.Command(
		serverBinaryPath,
		"-port", fmt.Sprint(addr.Port),
		"-db-url", dbConnectionString,
		"-public-key", publicKey,
	)
	session, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
	if err != nil {
		err = fmt.Errorf("failed to start server: %w", err)
		return
	}
	baseURL, err := url.Parse(fmt.Sprintf("http://%s", addr.String()))
	if err != nil {
		err = fmt.Errorf("failed to parse base URL: %w", err)
		return
	}
	server = APIServer{
		BaseURL: baseURL,
		session: session,
	}
	err = server.waitToAcceptConnections(ctx)
	return
}

func (server APIServer) Terminate() {
	server.session.Terminate().Wait()
}
