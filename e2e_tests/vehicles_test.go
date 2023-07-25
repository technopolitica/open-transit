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

func MakeValidVehicle(provider uuid.UUID) types.Vehicle {
	return types.Vehicle{
		DeviceId:        uuid.New(),
		ProviderId:      provider,
		VehicleType:     types.VehicleTypeMoped,
		PropulsionTypes: types.NewPropulsionTypeSet(types.PropulsionTypeCombustion, types.PropulsionTypeElectric),
	}
}

var maxUUIDTries = 5

func MakeUUIDExcluding(excludedUuids ...uuid.UUID) (id uuid.UUID, err error) {
	var excludedSet map[uuid.UUID]bool
	tryN := 0
	for tryN < maxUUIDTries {
		id, err = uuid.NewRandom()
		if err != nil {
			return
		}
		if !excludedSet[id] {
			return
		}
		tryN += 1
	}
	err = fmt.Errorf("failed to generate unique UUID")
	return
}

var _ = Describe("/vehicles", func() {
	Context("unauthenticated", func() {
		When("user attempts to register a valid vehicle", func() {
			var validVehicle types.Vehicle

			BeforeEach(func() {
				providerId, err := uuid.NewRandom()
				Expect(err).NotTo(HaveOccurred())
				validVehicle = MakeValidVehicle(providerId)
			})

			It("returns 401 Unauthorized status", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPStatus(401))
			})

			It("has WWW-Authenticate: Bearer header in response", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPHeaderWithValue("WWW-Authenticate", `Bearer, charset="UTF-8"`))
			})
		})
	})

	Context("authenticated w/ an unsigned JWT", func() {
		BeforeEach(func() {
			err := apiClient.AuthenticateWithUnsignedJWT()
			Expect(err).NotTo(HaveOccurred())
		})

		When("user attempts to register a valid vehicle", func() {
			var validVehicle types.Vehicle

			BeforeEach(func() {
				providerId, err := uuid.NewRandom()
				Expect(err).NotTo(HaveOccurred())
				validVehicle = MakeValidVehicle(providerId)
			})

			It("returns 401 Unauthorized status", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPStatus(401))
			})

			It("has WWW-Authenticate: Bearer header in response", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPHeaderWithValue("WWW-Authenticate", `Bearer, charset="UTF-8"`))
			})
		})
	})

	Context("authenticated as provider", func() {
		var providerId uuid.UUID

		BeforeEach(OncePerOrdered, func() {
			pid, err := uuid.NewRandom()
			providerId = pid
			Expect(err).NotTo(HaveOccurred())
			err = apiClient.AuthenticateAsProvider(providerId)
			Expect(err).NotTo(HaveOccurred())
		})

		When("user attempts to register a single vehicle with the null UUID", Ordered, func() {
			var invalidVehicle types.Vehicle

			BeforeAll(func() {
				invalidVehicle = MakeValidVehicle(providerId)
				invalidVehicle.DeviceId = uuid.UUID{}
			})

			It("returns a HTTP 400 Bad Request status", func() {
				Expect(apiClient.RegisterVehicles([]any{invalidVehicle})).To(HaveHTTPStatus(http.StatusBadRequest))
			})

			It("returns bulk response w/ bad_param error", func() {
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

		When("provider registers a valid vehicle that they own", Ordered, func() {
			var validVehicle types.Vehicle

			BeforeAll(func() {
				validVehicle = MakeValidVehicle(providerId)
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

		When("provider attempts to fetch an unregistered vehicle", func() {
			It("returns HTTP 404 Not Found status", func() {
				vid, err := uuid.NewRandom()
				Expect(err).NotTo(HaveOccurred())
				Expect(apiClient.GetVehicle(vid.String())).To(HaveHTTPStatus(http.StatusNotFound))
			})
		})

		When("provider attempts to register a single vehicle that they don't own", func() {
			var notProvidersVehicle types.Vehicle

			BeforeEach(func() {
				notProvidersId, err := MakeUUIDExcluding(providerId)
				Expect(err).NotTo(HaveOccurred())
				notProvidersVehicle = MakeValidVehicle(notProvidersId)
			})

			It("returns HTTP 400 Bad Request status", func() {
				Expect(apiClient.RegisterVehicles([]any{notProvidersVehicle})).To(HaveHTTPStatus(400))
			})

			It("returns bulk error response w/ bad_param", func() {
				notProvidersVehicleJSON, err := json.Marshal(notProvidersVehicle)
				Expect(err).NotTo(HaveOccurred())

				Expect(apiClient.RegisterVehicles([]any{notProvidersVehicle})).To(HaveHTTPBody(MatchJSON(fmt.Sprintf(`
				{
					"success": 0,
					"total": 1,
					"failures": [
						{
							"item": %s,
							"error": "bad_param",
							"error_description": "A validation error occurred",
							"error_details": ["provider_id: not allowed to register vehicle for another provider"]
						}
					]
				}	
			`, notProvidersVehicleJSON))))
			})
		})

		When("provider attempts to fetch a registered vehicle that they don't own", Ordered, func() {
			var notProvidersVehicle types.Vehicle

			BeforeAll(func() {
				By("another provider registering a vehicle")
				notProvidersId, err := MakeUUIDExcluding(providerId)
				Expect(err).NotTo(HaveOccurred())
				err = apiClient.AuthenticateAsProvider(notProvidersId)
				Expect(err).NotTo(HaveOccurred())
				notProvidersVehicle = MakeValidVehicle(notProvidersId)
				_, err = apiClient.RegisterVehicles([]any{notProvidersVehicle})
				Expect(err).NotTo(HaveOccurred())

				err = apiClient.AuthenticateAsProvider(providerId)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns HTTP 404 Not Found status", func() {
				Expect(apiClient.GetVehicle(notProvidersVehicle.DeviceId.String())).To(HaveHTTPStatus(404))
			})
		})
	})
})
