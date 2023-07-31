package testutils

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/technopolitica/open-mobility/types"
)

func MakeValidVehicle(provider uuid.UUID) *types.Vehicle {
	return &types.Vehicle{
		DeviceID:        uuid.New(),
		ProviderID:      provider,
		VehicleType:     types.VehicleTypeMoped,
		PropulsionTypes: types.NewSet(types.PropulsionTypeCombustion, types.PropulsionTypeElectric),
	}
}

func GenerateRandomUUID() uuid.UUID {
	id, err := uuid.NewRandom()
	Expect(err).NotTo(HaveOccurred())
	return id
}

var maxUUIDTries = 5

func MakeUUIDExcluding(excludedUuids ...uuid.UUID) (id uuid.UUID) {
	var excludedSet map[uuid.UUID]bool
	tryN := 0
	for tryN < maxUUIDTries {
		var err error
		id, err = uuid.NewRandom()
		if err != nil {
			continue
		}
		if !excludedSet[id] {
			return id
		}
		tryN += 1
	}
	err := fmt.Errorf("failed to generate unique UUID")
	Expect(err).NotTo(HaveOccurred())
	return
}
