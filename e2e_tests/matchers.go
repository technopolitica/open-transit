package e2e_tests

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func parseJSONObject(actual any) (object map[string]any, err error) {
	data, ok := actual.([]byte)
	if !ok {
		err = fmt.Errorf("MatchJSONObject matcher actual value must be of type []byte. Got:\n%s", format.Object(actual, 1))
		return
	}
	err = json.Unmarshal(data, &object)
	if err != nil {
		err = fmt.Errorf("MatchJSONObject failed to parse JSON object from actual value: %w", err)
	}
	return
}

func MatchJSONObject(matcher gomega.OmegaMatcher) gomega.OmegaMatcher {
	return gomega.WithTransform(parseJSONObject, matcher)
}
