package matchers

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/gomega"
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

func normalizeToJSONString(actual any) ([]byte, error) {
	switch a := actual.(type) {
	case []byte:
		return a, nil
	case string:
		return []byte(a), nil
	default:
		return json.Marshal(a)
	}
}

func MatchJSONObject(matchWith any) OmegaMatcher {
	switch matchWith := matchWith.(type) {
	case OmegaMatcher:
		return WithTransform(parseJSONObject, matchWith)
	default:
		jsonString, err := json.Marshal(matchWith)
		if err != nil {
			// KLUDGE: probably should deal with this error instead of panic...
			panic(err)
		}
		return WithTransform(normalizeToJSONString, MatchJSON(jsonString))
	}
}

func JSONValue(value any) (output any) {
	serializedValue, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(serializedValue, &output)
	if err != nil {
		panic(err)
	}
	return output
}
