//go:generate go run github.com/abice/go-enum@v0.5.6 --marshal --sql

package types

import (
	"encoding/json"
	"sort"

	"github.com/google/uuid"
)

// ENUM(other, bicycle, bus, cargo_bicycle, car, delivery_robot, moped, scooter_standing, scooter_seated, truck)
type VehicleType int

// ENUM(unknown, human, electric_assist, electric, combustion, combustion_diesel, hybrid, hydrogen_fuel_cell, plug_in_hybrid)
type PropulsionType int

type VehicleAttributes map[string]any

type AccessibilityAttributes map[string]any

type PropulsionTypeSet []PropulsionType

func NewPropulsionTypeSet(propulsionTypes ...PropulsionType) PropulsionTypeSet {
	sort.SliceStable(propulsionTypes, func(i int, j int) bool {
		return propulsionTypes[i] < propulsionTypes[j]
	})
	return propulsionTypes
}

func (pts PropulsionTypeSet) Marshal() ([]byte, error) {
	sort.SliceStable(pts, func(i int, j int) bool {
		return pts[i] < pts[j]
	})
	return json.Marshal(pts)
}

func (pts *PropulsionTypeSet) Unmarshal(data []byte) (err error) {
	var elements []PropulsionType
	err = json.Unmarshal(data, &elements)
	if err != nil {
		return
	}
	*pts = elements
	return
}

type Vehicle struct {
	DeviceId                uuid.UUID               `json:"device_id"`
	ProviderId              uuid.UUID               `json:"provider_id"`
	DataProviderId          uuid.NullUUID           `json:"data_provider_id,omitempty"`
	VehicleId               string                  `json:"vehicle_id"`
	VehicleType             VehicleType             `json:"vehicle_type"`
	VehicleAttributes       VehicleAttributes       `json:"vehicle_attributes"`
	PropulsionTypes         PropulsionTypeSet       `json:"propulsion_types"`
	AccessibilityAttributes AccessibilityAttributes `json:"accessibility_attributes,omitempty"`
	BatteryCapacity         int                     `json:"battery_capacity,omitempty"`
	FuelCapacity            int                     `json:"fuel_capacity,omitempty"`
	MaximumSpeed            int                     `json:"maximum_speed"`
}

func ValidateVehicle(value any, auth AuthInfo) []string {
	var errs []string
	switch v := value.(type) {
	case Vehicle:
		if v.DeviceId == (uuid.UUID{}) {
			errs = append(errs, "device_id: null UUID is not allowed")
		}
		if v.ProviderId != auth.ProviderId {
			errs = append(errs, "provider_id: not allowed to register vehicle for another provider")
		}
	default:
		panic("cannot validate unknown type")
	}
	return errs
}
