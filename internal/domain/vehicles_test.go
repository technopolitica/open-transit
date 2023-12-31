package domain

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PropulsionType", func() {
	describe := func(pt PropulsionType, expected string) string {
		t := reflect.TypeOf(pt)
		return fmt.Sprintf("%s -> %s", t.Name(), expected)
	}

	DescribeTable("marshals/unmarshalls to/from a JSON string",
		func(pt PropulsionType, expected string) {
			encodedExpected := fmt.Sprintf(`"%s"`, expected)
			Expect(json.Marshal(pt)).To(MatchJSON(encodedExpected))

			var output PropulsionType
			err := json.Unmarshal([]byte(encodedExpected), &output)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal(pt))
		},
		Entry(describe, PropulsionTypeCombustion, "combustion"),
		Entry(describe, PropulsionTypeCombustionDiesel, "combustion_diesel"),
		Entry(describe, PropulsionTypeElectric, "electric"),
		Entry(describe, PropulsionTypeElectricAssist, "electric_assist"),
		Entry(describe, PropulsionTypeHuman, "human"),
		Entry(describe, PropulsionTypeHybrid, "hybrid"),
		Entry(describe, PropulsionTypeHydrogenFuelCell, "hydrogen_fuel_cell"),
		Entry(describe, PropulsionTypePlugInHybrid, "plug_in_hybrid"),
	)
})

var _ = Describe("Vehicle", func() {
	It("compares equal to other vehicle w/ same fields", func() {
		Expect(Vehicle{
			DeviceID:        uuid.MustParse("1443963e-7d93-469c-b8e1-a262715c3b49"),
			PropulsionTypes: NewSet(PropulsionTypeCombustion, PropulsionTypeElectric),
		}).To(Equal(Vehicle{
			DeviceID:        uuid.MustParse("1443963e-7d93-469c-b8e1-a262715c3b49"),
			PropulsionTypes: NewSet(PropulsionTypeElectric, PropulsionTypeCombustion, PropulsionTypeElectric),
		}))
	})
})

var _ = Describe("PaginatedVehiclesResponse", func() {
	It("marshalls to JSON object", func() {
		Expect(json.Marshal(PaginatedVehiclesResponse{
			PaginatedResponse: PaginatedResponse{
				Version: "2.0.0",
				Links: PaginationLinks{
					First: "http://onlya.test/first",
					Last:  "http://onlya.test/last",
					Next:  "http://onlya.test/next",
					Prev:  "http://onlya.test/prev",
				},
			},
			Vehicles: []Vehicle{},
		})).To(MatchJSON(`
		{
			"version": "2.0.0",
			"links": {
				"first": "http://onlya.test/first",
				"last": "http://onlya.test/last",
				"next": "http://onlya.test/next",
				"prev": "http://onlya.test/prev"
			},
			"vehicles": []
		}
		`))
	})
})
