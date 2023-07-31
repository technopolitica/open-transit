package e2e_tests

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/technopolitica/open-mobility/e2e_tests/matchers"
	"github.com/technopolitica/open-mobility/e2e_tests/testutils"
	"github.com/technopolitica/open-mobility/types"
)

func readJSONBody[T any](res *http.Response) (output T) {
	data, err := io.ReadAll(res.Body)
	Expect(err).NotTo(HaveOccurred())
	defer res.Body.Close()
	err = json.Unmarshal(data, &output)
	Expect(err).NotTo(HaveOccurred())
	return
}

func fetchFirstPage() types.PaginatedVehiclesResponse {
	return readJSONBody[types.PaginatedVehiclesResponse](apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2}))
}

func fetchLastPage() types.PaginatedVehiclesResponse {
	firstPage := fetchFirstPage()
	return readJSONBody[types.PaginatedVehiclesResponse](apiClient.Get(firstPage.Links.Last))
}

func AssertHasStandardUnauthorizedResponse(op func() *http.Response) {
	It("returns 401 Unauthorized status", func() {
		Expect(op()).To(HaveHTTPStatus(http.StatusUnauthorized))
	})

	It("has WWW-Authenticate: Bearer header in response", func() {
		Expect(op()).To(HaveHTTPHeaderWithValue("WWW-Authenticate", `Bearer, charset="UTF-8"`))
	})
}

var _ = Describe("/vehicles", func() {
	Context("unauthenticated", func() {
		When("user attempts to register a valid vehicle", func() {
			var validVehicle *types.Vehicle
			BeforeEach(func() {
				providerID := testutils.GenerateRandomUUID()
				validVehicle = testutils.MakeValidVehicle(providerID)
			})

			AssertHasStandardUnauthorizedResponse(func() *http.Response {
				return apiClient.RegisterVehicles(validVehicle)
			})
		})

		When("user attempts to list vehicles", func() {
			AssertHasStandardUnauthorizedResponse(func() *http.Response {
				return apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2})
			})
		})
	})

	Context("authenticated w/ an unsigned JWT", func() {
		BeforeEach(func() {
			apiClient.AuthenticateWithUnsignedJWT()
		})

		When("user attempts to register a valid vehicle", func() {
			var validVehicle *types.Vehicle
			BeforeEach(func() {
				providerID := testutils.GenerateRandomUUID()
				validVehicle = testutils.MakeValidVehicle(providerID)
			})

			AssertHasStandardUnauthorizedResponse(func() *http.Response {
				return apiClient.RegisterVehicles(validVehicle)
			})
		})

		When("user attempts to list vehicles", func() {
			AssertHasStandardUnauthorizedResponse(func() *http.Response {
				return apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2})
			})
		})
	})

	Context("authenticated as provider", func() {
		var providerID uuid.UUID
		BeforeEach(OncePerOrdered, func() {
			providerID = testutils.GenerateRandomUUID()
			apiClient.AuthenticateAsProvider(providerID)
		})

		When("user attempts to register a single vehicle with the null UUID", Ordered, func() {
			var invalidVehicle *types.Vehicle
			BeforeAll(func() {
				invalidVehicle = testutils.MakeValidVehicle(providerID)
				invalidVehicle.DeviceID = uuid.UUID{}
			})

			It("returns a HTTP 400 Bad Request status", func() {
				Expect(apiClient.RegisterVehicles([]any{invalidVehicle})).To(HaveHTTPStatus(http.StatusBadRequest))
			})

			It("returns bulk response w/ bad_param error", func() {
				Expect(apiClient.RegisterVehicles([]any{invalidVehicle})).To(HaveHTTPBody(MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
					"success": Equal(float64(0)),
					"total":   Equal(float64(1)),
					"failures": ConsistOf(MatchKeys(IgnoreExtras, Keys{
						"error":             Equal("bad_param"),
						"error_description": Equal("A validation error occurred"),
						"error_details":     ConsistOf("device_id: null UUID is not allowed"),
						"item":              MatchJSONObject(invalidVehicle),
					})),
				}))))
			})
		})

		When("provider registers a valid vehicle that they own", Ordered, func() {
			var validVehicle *types.Vehicle
			BeforeAll(func() {
				validVehicle = testutils.MakeValidVehicle(providerID)
			})

			It("returns HTTP 201 Created status", func() {
				Expect(apiClient.RegisterVehicles([]any{validVehicle})).To(HaveHTTPStatus(http.StatusCreated))
			})

			It("fetching the newly registered vehicle returns HTTP 200 OK status", func() {
				Expect(apiClient.GetVehicle(validVehicle.DeviceID.String())).To(HaveHTTPStatus(http.StatusOK))
			})

			It("fetching the newly registered vehicle returns the same vehicle that was registered", func() {
				Expect(apiClient.GetVehicle(validVehicle.DeviceID.String())).To(HaveHTTPBody(MatchJSONObject(validVehicle)))
			})
		})

		When("provider attempts to fetch an unregistered vehicle", func() {
			It("returns HTTP 404 Not Found status", func() {
				vid := testutils.GenerateRandomUUID()
				Expect(apiClient.GetVehicle(vid.String())).To(HaveHTTPStatus(http.StatusNotFound))
			})
		})

		When("provider attempts to register a single vehicle that they don't own", func() {
			var notProvidersVehicle *types.Vehicle
			BeforeEach(func() {
				notProvidersID := testutils.MakeUUIDExcluding(providerID)
				notProvidersVehicle = testutils.MakeValidVehicle(notProvidersID)
			})

			It("returns HTTP 400 Bad Request status", func() {
				Expect(apiClient.RegisterVehicles([]any{notProvidersVehicle})).To(HaveHTTPStatus(http.StatusBadRequest))
			})

			It("returns bulk error response w/ bad_param", func() {
				Expect(apiClient.RegisterVehicles([]any{notProvidersVehicle})).To(HaveHTTPBody(MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
					"success": Equal(float64(0)),
					"total":   Equal(float64(1)),
					"failures": ConsistOf(MatchKeys(IgnoreExtras, Keys{
						"error":             Equal("bad_param"),
						"error_description": Equal("A validation error occurred"),
						"error_details":     ConsistOf("provider_id: not allowed to register vehicle for another provider"),
						"item":              MatchJSONObject(notProvidersVehicle),
					})),
				}))))
			})
		})

		When("provider attempts to fetch a registered vehicle that they don't own", Ordered, func() {
			var notProvidersVehicle *types.Vehicle
			BeforeAll(func() {
				By("another provider registering a vehicle")
				notProvidersID := testutils.MakeUUIDExcluding(providerID)
				apiClient.AuthenticateAsProvider(notProvidersID)
				notProvidersVehicle = testutils.MakeValidVehicle(notProvidersID)
				apiClient.RegisterVehicles([]any{notProvidersVehicle})

				apiClient.AuthenticateAsProvider(providerID)
			})

			It("returns HTTP 404 Not Found status", func() {
				Expect(apiClient.GetVehicle(notProvidersVehicle.DeviceID.String())).To(HaveHTTPStatus(http.StatusNotFound))
			})
		})

		When("provider requests a list of registered vehicles", func() {
			When("there are no registered vehicles owned by the requesting provider", func() {
				It("returns 200 OK status", func() {
					Expect(apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 10})).To(HaveHTTPStatus(
						http.StatusOK,
					))
				})

				It("returns empty paginated response", func() {
					Expect(
						apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 10}),
					).To(HaveHTTPBody(MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
						"vehicles": BeEmpty(),
					}))))
				})
			})

			When("there are registered vehicles owned by the requesting provider", Ordered, func() {
				var registeredVehicles []types.Vehicle
				BeforeAll(func() {
					for i := 0; i < 5; i++ {
						vehicle := testutils.MakeValidVehicle(providerID)
						registeredVehicles = append(registeredVehicles, *vehicle)
					}
					apiClient.RegisterVehicles(registeredVehicles)
				})

				When("provider requests a list of vehicles w/ a limit smaller than the number of registered vehicles", func() {
					It("returns HTTP 200 OK status", func() {
						Expect(apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2})).To(HaveHTTPStatus(http.StatusOK))
					})

					It("returns an array of vehicles of legth = requested limit", func() {
						Expect(apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2})).To(HaveHTTPBody(
							MatchJSONObject(MatchKeys(IgnoreExtras, Keys{
								"vehicles": HaveLen(2),
							})),
						))
					})

					It("returns response w/o prev link on first page", func() {
						Expect(apiClient.ListVehicles(testutils.ListVehiclesOptions{Limit: 2})).To(HaveHTTPBody(
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
