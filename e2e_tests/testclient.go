package e2e_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type TestClient struct {
	baseURL url.URL
}

func NewTestClient(baseURL url.URL) *TestClient {
	return &TestClient{baseURL}
}

func (client *TestClient) endpoint(path ...string) *url.URL {
	return client.baseURL.JoinPath(path...)
}

func (client *TestClient) RegisterVehicles(vehicles any) (response *http.Response, err error) {
	vehiclesBytes, err := json.Marshal(vehicles)
	if err != nil {
		return
	}

	response, err = http.Post(client.endpoint("/vehicles").String(), "application/vnd.mds+json", bytes.NewBuffer(vehiclesBytes))
	if err != nil {
		err = fmt.Errorf("failed to send request: %s", err)
		return
	}

	return
}

func (client *TestClient) GetVehicle(vehicleId string) (response *http.Response, err error) {
	request, err := http.NewRequest("GET", client.endpoint("/vehicles", vehicleId).String(), nil)
	if err != nil {
		err = fmt.Errorf("failed to create request: %s", err)
		return
	}
	request.Header.Set("ContentType", "application/vnd.mds+json")

	response, err = http.DefaultClient.Do(request)
	if err != nil {
		err = fmt.Errorf("failed to send request: %s", err)
		return
	}
	return
}
