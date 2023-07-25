package e2e_tests

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TestClient struct {
	baseURL    url.URL
	authToken  string
	signingKey rsa.PrivateKey
}

func NewTestClient(baseURL url.URL, signingKey rsa.PrivateKey) *TestClient {
	return &TestClient{baseURL: baseURL, signingKey: signingKey}
}

func (client *TestClient) endpoint(path ...string) string {
	return client.baseURL.JoinPath(path...).String()
}

func (client *TestClient) authenticateWithAuthToken(signingMethod jwt.SigningMethod, key any, claims jwt.Claims) (err error) {
	authToken := jwt.NewWithClaims(signingMethod, claims)
	client.authToken, err = authToken.SignedString(key)
	return
}

func (client *TestClient) AuthenticateWithUnsignedJWT() (err error) {
	providerId, err := uuid.NewRandom()
	if err != nil {
		return
	}
	return client.authenticateWithAuthToken(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, struct {
		jwt.RegisteredClaims
		Provider uuid.UUID `json:"provider_id"`
	}{
		Provider: providerId,
	})
}

func (client *TestClient) AuthenticateAsProvider(providerId uuid.UUID) (err error) {
	return client.authenticateWithAuthToken(jwt.SigningMethodRS256, &client.signingKey, struct {
		jwt.RegisteredClaims
		Provider uuid.UUID `json:"provider_id"`
	}{
		Provider: providerId,
	})
}

func (client *TestClient) Reset() {
	*client = TestClient{
		baseURL:    client.baseURL,
		signingKey: client.signingKey,
	}
}

func (client *TestClient) sendRequestWithDefaultHeaders(method string, endpoint string, body any) (res *http.Response, err error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		err = fmt.Errorf("failed to serialize JSON payload: %w", err)
		return
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		return
	}
	if client.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.authToken))
	}
	req.Header.Set("Content-Type", "application/vnd.mds+json")

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to send request: %s", err)
		return
	}
	return
}

func (client *TestClient) RegisterVehicles(vehicles any) (response *http.Response, err error) {
	return client.sendRequestWithDefaultHeaders("POST", client.endpoint("/vehicles"), vehicles)
}

func (client *TestClient) GetVehicle(vehicleId string) (response *http.Response, err error) {
	return client.sendRequestWithDefaultHeaders("GET", client.endpoint("/vehicles", vehicleId), nil)
}
