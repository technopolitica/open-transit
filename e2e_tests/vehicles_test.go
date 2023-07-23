package e2e_tests

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/technopolitica/open-mobility/types"
)

func MakeValidVehicle() types.Vehicle {
	return types.Vehicle{
		DeviceId:        uuid.New(),
		ProviderId:      uuid.New(),
		VehicleType:     types.VehicleTypeMoped,
		PropulsionTypes: types.NewPropulsionTypeSet(types.PropulsionTypeCombustion, types.PropulsionTypeElectric),
	}
}

var _ = Describe("/vehicles", func() {
	When("user attempts to register a single vehicle with the null UUID", func() {
		It("returns a HTTP 400 Bad Request status", func() {
			invalidVehicle := MakeValidVehicle()
			invalidVehicle.DeviceId = uuid.UUID{}
			Expect(apiClient.RegisterVehicles([]any{invalidVehicle})).To(HaveHTTPStatus(http.StatusBadRequest))
		})

		It("returns bulk response w/ bad_param error", func() {
			invalidVehicle := MakeValidVehicle()
			invalidVehicle.DeviceId = uuid.UUID{}
			invalidVehicleJSON, err := json.Marshal(invalidVehicle)
			Expect(err).NotTo(HaveOccurred())
			Expect(apiClient.RegisterVehicles([]any{invalidVehicle})).To(HaveHTTPBody(MatchJSON(fmt.Sprintf(`
				{
					"success": 0,
					"total": 1,
					"failures": [
						{
							"item": %s,
							"error": "bad_param",
							"error_description": "A validation error occurred",
							"error_details": ["device_id: null UUID is not allowed"]
						}
					]
				}	
			`, invalidVehicleJSON))))
		})
	})

	When("user registers a valid vehicle", Ordered, func() {
		var validVehicle types.Vehicle

		BeforeAll(func() {
			validVehicle = MakeValidVehicle()
		})

		It("returns HTTP 201 Created status", func() {
			Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPStatus(http.StatusCreated))
		})

		It("fetching the newly registered vehicle returns HTTP 200 OK status", func() {
			Expect(apiClient.GetVehicle(validVehicle.DeviceId.String())).To(HaveHTTPStatus(http.StatusOK))
		})

		It("fetching the newly registered vehicle returns the same vehicle that was registered", func() {
			expected, err := json.Marshal(validVehicle)
			Expect(err).NotTo(HaveOccurred())
			Expect(apiClient.GetVehicle(validVehicle.DeviceId.String())).To(HaveHTTPBody(MatchJSON(
				expected,
			)))
		})
	})

	When("user attempts to fetch an unregistered vehicle", func() {
		It("returns HTTP 404 Not Found status", func() {
			vid, err := uuid.NewRandom()
			Expect(err).NotTo(HaveOccurred())
			Expect(apiClient.GetVehicle(vid.String())).To(HaveHTTPStatus(http.StatusNotFound))
		})
	})
})
