package domain

import "github.com/google/uuid"

type AuthInfo struct {
	ProviderID uuid.UUID `json:"provider_id"`
}
