package testutils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

type APIServer struct {
	BaseURL    *url.URL
	PrivateKey *rsa.PrivateKey
	session    *gexec.Session
}

const rsa256BitSize = 128 * 8

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

func StartAPIServer(ctx context.Context, serverBinaryPath string, dbConnectionString string) (server APIServer, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsa256BitSize)
	if err != nil {
		err = fmt.Errorf("failed to generate private/public key pair: %w", err)
		return
	}
	publicKeyFilePath, err := writePublicKeyFile(&privateKey.PublicKey)
	if err != nil {
		err = fmt.Errorf("failed to write public key file: %w", err)
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
		"-public-key", fmt.Sprintf("file://%s", publicKeyFilePath),
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
		PrivateKey: privateKey,
		BaseURL:    baseURL,
		session:    session,
	}
	err = server.waitToAcceptConnections(ctx)
	return
}

func (server APIServer) Terminate() {
	server.session.Terminate().Wait()
}
