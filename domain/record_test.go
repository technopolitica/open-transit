package domain

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Record", func() {
	It("can be unmarshalled from an empty JSON object", func() {
		var rec Record
		Expect(json.Unmarshal([]byte("{}"), &rec)).Error().NotTo(HaveOccurred())
		Expect(rec).To(Equal(Record{}))
	})

	It("can be unmarshalled from a JSON object w/ entries", func() {
		var rec Record
		Expect(json.Unmarshal([]byte(`{ "foo": "bar" }`), &rec)).Error().NotTo(HaveOccurred())
		Expect(rec).To(Equal(Record{map[string]any{"foo": "bar"}}))
	})

	When("it has entries", func() {
		It("marshalls to JSON object", func() {
			rec := Record{map[string]any{"foo": "bar"}}
			Expect(json.Marshal(rec)).To(MatchJSON(`{ "foo": "bar" }`))
		})
	})

	When("nil", func() {
		It("marshalls to empty object JSON value", func() {
			rec := Record{}
			Expect(json.Marshal(rec)).To(MatchJSON(`{}`))
		})

		It("converted to non-nil value for databases", func(ctx context.Context) {
			rec := Record{}
			Expect(rec.Value()).To(Equal(map[string]any{}))
		})
	})
})
