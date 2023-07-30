//go:generate go run github.com/abice/go-enum@v0.5.6 --marshal --sql

package types

import (
	"github.com/google/uuid"
)

// ENUM(other, bicycle, bus, cargo_bicycle, car, delivery_robot, moped, scooter_standing, scooter_seated, truck)
type VehicleType int

// ENUM(unknown, human, electric_assist, electric, combustion, combustion_diesel, hybrid, hydrogen_fuel_cell, plug_in_hybrid)
type PropulsionType int

type VehicleAttributes map[string]any

type AccessibilityAttributes map[string]any

type Vehicle struct {
	DeviceID                uuid.UUID               `json:"device_id"`
	ProviderID              uuid.UUID               `json:"provider_id"`
	DataProviderID          uuid.UUID               `json:"data_provider_id,omitempty"`
	VehicleID               string                  `json:"vehicle_id"`
	VehicleType             VehicleType             `json:"vehicle_type"`
	VehicleAttributes       VehicleAttributes       `json:"vehicle_attributes"`
	PropulsionTypes         Set[PropulsionType]     `json:"propulsion_types"`
	AccessibilityAttributes AccessibilityAttributes `json:"accessibility_attributes,omitempty"`
	BatteryCapacity         int                     `json:"battery_capacity,omitempty"`
	FuelCapacity            int                     `json:"fuel_capacity,omitempty"`
	MaximumSpeed            int                     `json:"maximum_speed"`
}

func ValidateVehicle(value any, auth AuthInfo) []string {
	var errs []string
	switch v := value.(type) {
	case Vehicle:
		if v.DeviceID == (uuid.UUID{}) {
			errs = append(errs, "device_id: null UUID is not allowed")
		}
		if v.ProviderID != auth.ProviderID {
			errs = append(errs, "provider_id: not allowed to register vehicle for another provider")
		}
	default:
		panic("cannot validate unknown type")
	}
	return errs
}

type PaginatedVehiclesResponse struct {
	PaginatedResponse
	Vehicles []Vehicle `json:"vehicles"`
}
