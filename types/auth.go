package types

import "github.com/google/uuid"

type AuthInfo struct {
	ProviderId uuid.UUID `json:"provider_id"`
}
