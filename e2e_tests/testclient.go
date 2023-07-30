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

type PaginationLinks struct {
	First *url.URL `json:"first,omitempty"`
	Last  *url.URL `json:"last,omitempty"`
	Next  *url.URL `json:"next,omitempty"`
	Prev  *url.URL `json:"prev,omitempty"`
}

type TestClient struct {
	baseURL    url.URL
	authToken  string
	signingKey rsa.PrivateKey
}

func NewTestClient(baseURL url.URL, signingKey rsa.PrivateKey) *TestClient {
	return &TestClient{baseURL: baseURL, signingKey: signingKey}
}

func (client *TestClient) endpoint(path ...string) *url.URL {
	return client.baseURL.JoinPath(path...)
}

func (client *TestClient) authenticateWithAuthToken(signingMethod jwt.SigningMethod, key any, claims jwt.Claims) (err error) {
	authToken := jwt.NewWithClaims(signingMethod, claims)
	client.authToken, err = authToken.SignedString(key)
	return
}

func (client *TestClient) AuthenticateWithUnsignedJWT() (err error) {
	providerID, err := uuid.NewRandom()
	if err != nil {
		return
	}
	return client.authenticateWithAuthToken(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, struct {
		jwt.RegisteredClaims
		Provider uuid.UUID `json:"provider_id"`
	}{
		Provider: providerID,
	})
}

func (client *TestClient) AuthenticateAsProvider(providerID uuid.UUID) (err error) {
	return client.authenticateWithAuthToken(jwt.SigningMethodRS256, &client.signingKey, struct {
		jwt.RegisteredClaims
		Provider uuid.UUID `json:"provider_id"`
	}{
		Provider: providerID,
	})
}

func (client *TestClient) BaseURL() *url.URL {
	copy := client.baseURL
	return &copy
}

func (client *TestClient) Reset() {
	*client = TestClient{
		baseURL:    client.baseURL,
		signingKey: client.signingKey,
	}
}

func (client *TestClient) sendRequestWithDefaultHeaders(method string, endpoint *url.URL, body any) (res *http.Response, err error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		err = fmt.Errorf("failed to serialize JSON payload: %w", err)
		return
	}

	req, err := http.NewRequest(method, endpoint.String(), bytes.NewBuffer(jsonBody))
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

func (client *TestClient) GetVehicle(vehicleID string) (response *http.Response, err error) {
	return client.sendRequestWithDefaultHeaders("GET", client.endpoint("/vehicles", vehicleID), nil)
}

func (client *TestClient) Get(path string) (response *http.Response, err error) {
	uri, err := url.ParseRequestURI(path)
	if err != nil {
		return
	}
	endpoint := client.baseURL.JoinPath(uri.Path)
	endpoint.RawQuery = uri.Query().Encode()
	return client.sendRequestWithDefaultHeaders("GET", endpoint, nil)
}

type ListVehiclesOptions struct {
	Limit  int
	Offset int
}

func (client *TestClient) ListVehicles(options ListVehiclesOptions) (response *http.Response, err error) {
	url := client.endpoint("/vehicles")
	query := url.Query()
	// Default to a limit of 10 so that we can use the zero value of the options struct to make tests a little more readable.
	if options.Limit == 0 {
		options.Limit = 10
	}
	query.Add("page[limit]", fmt.Sprint(options.Limit))
	if options.Offset != 0 {
		query.Add("page[offset]", fmt.Sprint(options.Offset))
	}
	url.RawQuery = query.Encode()

	return client.sendRequestWithDefaultHeaders("GET", url, nil)
}
