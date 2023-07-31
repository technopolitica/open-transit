package testutils

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
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

func (client *TestClient) authenticateWithAuthToken(signingMethod jwt.SigningMethod, key any, claims jwt.Claims) {
	authToken := jwt.NewWithClaims(signingMethod, claims)
	var err error
	client.authToken, err = authToken.SignedString(key)
	Expect(err).NotTo(HaveOccurred())
}

func (client *TestClient) AuthenticateWithUnsignedJWT() {
	providerID := GenerateRandomUUID()
	client.authenticateWithAuthToken(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, struct {
		jwt.RegisteredClaims
		Provider uuid.UUID `json:"provider_id"`
	}{
		Provider: providerID,
	})
}

func (client *TestClient) AuthenticateAsProvider(providerID uuid.UUID) {
	client.authenticateWithAuthToken(jwt.SigningMethodRS256, &client.signingKey, struct {
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

func (client *TestClient) sendRequestWithDefaultHeaders(method string, endpoint *url.URL, body any) (res *http.Response) {
	jsonBody, err := json.Marshal(body)
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest(method, endpoint.String(), bytes.NewBuffer(jsonBody))
	Expect(err).NotTo(HaveOccurred())
	if client.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.authToken))
	}
	req.Header.Set("Content-Type", "application/vnd.mds+json")

	res, err = http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return
}

func (client *TestClient) RegisterVehicles(vehicles any) (response *http.Response) {
	return client.sendRequestWithDefaultHeaders("POST", client.endpoint("/vehicles"), vehicles)
}

func (client *TestClient) GetVehicle(vehicleID string) (response *http.Response) {
	return client.sendRequestWithDefaultHeaders("GET", client.endpoint("/vehicles", vehicleID), nil)
}

func (client *TestClient) Get(path string) (response *http.Response) {
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

func (client *TestClient) ListVehicles(options ListVehiclesOptions) (response *http.Response) {
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
