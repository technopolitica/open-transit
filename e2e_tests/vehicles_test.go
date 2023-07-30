package e2e_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/technopolitica/open-mobility/types"
)

func readJSONBody[T any](res *http.Response, err error) (output T) {
	Expect(err).NotTo(HaveOccurred())
	data, err := io.ReadAll(res.Body)
	Expect(err).NotTo(HaveOccurred())
	defer res.Body.Close()
	err = json.Unmarshal(data, &output)
	Expect(err).NotTo(HaveOccurred())
	return
}

func fetchFirstPage() types.PaginatedVehiclesResponse {
	return readJSONBody[types.PaginatedVehiclesResponse](apiClient.ListVehicles(ListVehiclesOptions{Limit: 2}))
}

func fetchLastPage() types.PaginatedVehiclesResponse {
	firstPage := fetchFirstPage()
	return readJSONBody[types.PaginatedVehiclesResponse](apiClient.Get(firstPage.Links.Last))
}

func MakeValidVehicle(provider uuid.UUID) types.Vehicle {
	return types.Vehicle{
		DeviceID:        uuid.New(),
		ProviderID:      provider,
		VehicleType:     types.VehicleTypeMoped,
		PropulsionTypes: types.NewSet(types.PropulsionTypeCombustion, types.PropulsionTypeElectric),
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
				providerID, err := uuid.NewRandom()
				Expect(err).NotTo(HaveOccurred())
				validVehicle = MakeValidVehicle(providerID)
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
				providerID, err := uuid.NewRandom()
				Expect(err).NotTo(HaveOccurred())
				validVehicle = MakeValidVehicle(providerID)
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
		var providerID uuid.UUID

		BeforeEach(OncePerOrdered, func() {
			pid, err := uuid.NewRandom()
			providerID = pid
			Expect(err).NotTo(HaveOccurred())
			err = apiClient.AuthenticateAsProvider(providerID)
			Expect(err).NotTo(HaveOccurred())
		})

		When("user attempts to register a single vehicle with the null UUID", Ordered, func() {
			var invalidVehicle types.Vehicle

			BeforeAll(func() {
				invalidVehicle = MakeValidVehicle(providerID)
				invalidVehicle.DeviceID = uuid.UUID{}
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
				validVehicle = MakeValidVehicle(providerID)
			})

			It("returns HTTP 201 Created status", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPStatus(http.StatusCreated))
			})

			It("fetching the newly registered vehicle returns HTTP 200 OK status", func() {
				Expect(apiClient.GetVehicle(validVehicle.DeviceID.String())).To(HaveHTTPStatus(http.StatusOK))
			})

			It("fetching the newly registered vehicle returns the same vehicle that was registered", func() {
				expected, err := json.Marshal(validVehicle)
				Expect(err).NotTo(HaveOccurred())
				Expect(apiClient.GetVehicle(validVehicle.DeviceID.String())).To(HaveHTTPBody(MatchJSON(
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
				notProvidersID, err := MakeUUIDExcluding(providerID)
				Expect(err).NotTo(HaveOccurred())
				notProvidersVehicle = MakeValidVehicle(notProvidersID)
			})

			It("returns HTTP 400 Bad Request status", func() {
				Expect(apiClient.RegisterVehicles([]any{notProvidersVehicle})).To(HaveHTTPStatus(http.StatusBadRequest))
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
				notProvidersID, err := MakeUUIDExcluding(providerID)
				Expect(err).NotTo(HaveOccurred())
				err = apiClient.AuthenticateAsProvider(notProvidersID)
				Expect(err).NotTo(HaveOccurred())
				notProvidersVehicle = MakeValidVehicle(notProvidersID)
				_, err = apiClient.RegisterVehicles([]any{notProvidersVehicle})
				Expect(err).NotTo(HaveOccurred())

				err = apiClient.AuthenticateAsProvider(providerID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns HTTP 404 Not Found status", func() {
				Expect(apiClient.GetVehicle(notProvidersVehicle.DeviceID.String())).To(HaveHTTPStatus(http.StatusNotFound))
			})
		})

		When("provider requests a list of registered vehicles", func() {
			When("there are no registered vehicles owned by the requesting provider", func() {
				It("returns 200 OK status", func() {
					Expect(apiClient.ListVehicles(ListVehiclesOptions{Limit: 10})).To(HaveHTTPStatus(
						http.StatusOK,
					))
				})

				It("returns empty paginated response", func() {
					Expect(
						apiClient.ListVehicles(ListVehiclesOptions{Limit: 10}),
					).To(HaveHTTPBody(MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
						"vehicles": BeEmpty(),
					}))))
				})
			})

			When("there are registered vehicles owned by the requesting provider", Ordered, func() {
				var registeredVehicles []types.Vehicle
				BeforeAll(func() {
					for i := 0; i < 5; i++ {
						vehicle := MakeValidVehicle(providerID)
						registeredVehicles = append(registeredVehicles, vehicle)
					}
					_, err := apiClient.RegisterVehicles(registeredVehicles)
					Expect(err).NotTo(HaveOccurred())
				})

				When("provider requests a list of vehicles w/ a limit smaller than the number of registered vehicles", func() {
					It("returns HTTP 200 OK status", func() {
						Expect(apiClient.ListVehicles(ListVehiclesOptions{Limit: 2})).To(HaveHTTPStatus(http.StatusOK))
					})

					It("returns an array of vehicles of legth = requested limit", func() {
						Expect(apiClient.ListVehicles(ListVehiclesOptions{Limit: 2})).To(HaveHTTPBody(
							MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
								"vehicles": HaveLen(2),
							})),
						))
					})

					It("returns response w/o prev link on first page", func() {
						Expect(apiClient.ListVehicles(ListVehiclesOptions{Limit: 2})).To(HaveHTTPBody(
							MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
								"links": Not(HaveKey("prev")),
							})),
						))
					})

					It("returns response w/o next link on last page", func() {
						lastPage := fetchLastPage()

						Expect(apiClient.Get(lastPage.Links.Last)).To(HaveHTTPBody(
							MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
								"links": Not(HaveKey("next")),
							})),
						))
					})

					It("allows user to page through full set of vehicles from first page by following next links", func() {
						foundVehicles := make([]types.Vehicle, 0, len(registeredVehicles))

						firstPage := fetchFirstPage()
						foundVehicles = append(foundVehicles, firstPage.Vehicles...)

						next := firstPage.Links.Next
						for next != "" {
							page := readJSONBody[types.PaginatedVehiclesResponse](apiClient.Get(next))

							foundVehicles = append(foundVehicles, page.Vehicles...)
							next = page.Links.Next
						}

						Expect(foundVehicles).To(ConsistOf(registeredVehicles))
					})

					It("allows user to page through full set of vehicles from last page by following prev links", func() {
						foundVehicles := make([]types.Vehicle, 0, len(registeredVehicles))

						lastPage := fetchLastPage()
						foundVehicles = append(foundVehicles, lastPage.Vehicles...)

						prev := lastPage.Links.Prev
						for prev != "" {
							page := readJSONBody[types.PaginatedVehiclesResponse](apiClient.Get(prev))
							foundVehicles = append(foundVehicles, page.Vehicles...)
							prev = page.Links.Prev
						}

						Expect(foundVehicles).To(ConsistOf(registeredVehicles))
					})
				})
			})
		})
	})
})
