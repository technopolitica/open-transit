package domain

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Set", func() {
	It("marshals to an ordered JSON array", func() {
		Expect(json.Marshal(NewSet("electric", "combustion"))).
			To(MatchJSON(`["combustion", "electric"]`))
	})

	It("unmarshals from JSON array", func() {
		var output Set[string]
		err := json.Unmarshal([]byte(`["electric", "combustion"]`), &output)
		Expect(err).NotTo(HaveOccurred())

		Expect(output).To(Equal(NewSet("electric", "combustion")))
	})

	It("removes duplicates", func() {
		Expect(NewSet("combustion", "electric", "combustion")).To(Equal(
			NewSet("combustion", "electric"),
		))
	})

	It("compares equal with other PropulsionTypeSets w/ same elements regardless of ordering", func() {
		Expect(NewSet("combustion", "electric")).To(Equal(
			NewSet("electric", "combustion"),
		))
	})
})
