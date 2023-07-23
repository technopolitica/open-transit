package types

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type SomeStruct struct {
	Foo string `json:"foo"`
}

var _ = Describe("FailureDetails", func() {
	It("can be serialized as JSON", func() {
		Expect(json.Marshal(FailureDetails[SomeStruct]{
			Item: SomeStruct{
				Foo: "bar",
			},
			ApiError: ApiError{
				Type:    ApiErrorTypeBadParam,
				Details: []string{"details"},
			},
		})).To(MatchJSON(`
		     {
			 	"item": { "foo": "bar" },
				"error": "bad_param",
				"error_description": "A validation error occurred",
				"error_details": ["details"]
		     }
		 `))
	})
})
