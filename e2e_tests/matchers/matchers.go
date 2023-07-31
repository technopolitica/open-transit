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

func MatchJSONObject(arg any) OmegaMatcher {
	matcher, ok := arg.(OmegaMatcher)
	if ok {
		return WithTransform(parseJSONObject, matcher)
	}

	switch match := arg.(type) {
	case []byte:
	case string:
		matcher = MatchJSON(match)
	default:
		jsonString, err := json.Marshal(match)
		if err != nil {
			// KLUDGE: probably should deal with this error instead of panic...
			panic(err)
		}
		matcher = MatchJSON(jsonString)
	}
	transformOrPassthrough := func(actual any) ([]byte, error) {
		switch a := actual.(type) {
		case []byte:
			return a, nil
		case string:
			return []byte(a), nil
		default:
			return json.Marshal(a)
		}
	}
	return WithTransform(transformOrPassthrough, matcher)
}
