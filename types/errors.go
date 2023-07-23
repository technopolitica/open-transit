//go:generate go run github.com/abice/go-enum@v0.5.6 --marshal --sql

package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ENUM(unknown, bad_param, missing_param, already_registered)
type ApiErrorType int

type ApiError struct {
	Type    ApiErrorType
	Details []string
}

func (res ApiError) Description() string {
	switch res.Type {
	case ApiErrorTypeBadParam:
		return "A validation error occurred"
	case ApiErrorTypeMissingParam:
		return "A required parameter is missing"
	case ApiErrorTypeAlreadyRegistered:
		return "A vehicle with device_id is already registered"
	default:
		return "An unknown error occurred"
	}
}

func (res ApiError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        ApiErrorType `json:"error"`
		Description string       `json:"error_description"`
		Details     []string     `json:"error_details"`
	}{
		Type:        res.Type,
		Description: res.Description(),
		Details:     res.Details,
	})
}

func (res ApiError) Error() string {
	return fmt.Sprintf("%s: %s\n%s", res.Type, res.Description(), strings.Join(res.Details, "\n"))
}

type FailureDetails[T any] struct {
	ApiError
	Item T
}

// KLUDGE:
//
//	 Because we are embedding ApiError in FailureDetails, and ApiError has a overridden
//		Marshaler implementation which gets promoted to FailureDetails by embedding, if
//		we do not manually serializel FailureDetails here the Item field will never be output
//		because the promoted Mashaler implementation knows nothing about FailureDetails fields.
func (failure FailureDetails[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Item        T            `json:"item"`
		Type        ApiErrorType `json:"error"`
		Description string       `json:"error_description"`
		Details     []string     `json:"error_details"`
	}{
		Item:        failure.Item,
		Type:        failure.Type,
		Description: failure.Description(),
		Details:     failure.Details,
	})
}

type BulkApiResponse[T any] struct {
	Success  int                 `json:"success"`
	Total    int                 `json:"total"`
	Failures []FailureDetails[T] `json:"failures"`
}
