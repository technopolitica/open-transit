package types

import (
	"encoding/json"
	"fmt"
	"reflect"

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

var _ = Describe("PropulsionTypeSet", func() {
	It("marshals to JSON array", func() {
		Expect(json.Marshal(NewPropulsionTypeSet(PropulsionTypeCombustion, PropulsionTypeElectric))).
			To(MatchJSON(`["electric", "combustion"]`))
	})

	It("unmarshals from JSON array", func() {
		var output PropulsionTypeSet
		err := json.Unmarshal([]byte(`["electric", "combustion"]`), &output)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal(NewPropulsionTypeSet(PropulsionTypeCombustion, PropulsionTypeElectric)))
	})
})
