//go:generate go run github.com/abice/go-enum@v0.5.6 --marshal --sql

package domain

import (
	"context"

	"github.com/google/uuid"
)

// ENUM(other, bicycle, bus, cargo_bicycle, car, delivery_robot, moped, scooter_standing, scooter_seated, truck)
type VehicleType int

// ENUM(unknown, human, electric_assist, electric, combustion, combustion_diesel, hybrid, hydrogen_fuel_cell, plug_in_hybrid)
type PropulsionType int

type Vehicle struct {
	DeviceID                uuid.UUID           `json:"device_id"`
	ProviderID              uuid.UUID           `json:"provider_id"`
	DataProviderID          uuid.UUID           `json:"data_provider_id,omitempty"`
	VehicleID               string              `json:"vehicle_id"`
	VehicleType             VehicleType         `json:"vehicle_type"`
	VehicleAttributes       Record              `json:"vehicle_attributes"`
	PropulsionTypes         Set[PropulsionType] `json:"propulsion_types"`
	AccessibilityAttributes Record              `json:"accessibility_attributes,omitempty"`
	BatteryCapacity         int                 `json:"battery_capacity,omitempty"`
	FuelCapacity            int                 `json:"fuel_capacity,omitempty"`
	MaximumSpeed            int                 `json:"maximum_speed"`
}

func ValidateVehicle(value any) []string {
	var errs []string
	switch v := value.(type) {
	case Vehicle:
		if v.DeviceID == (uuid.UUID{}) {
			errs = append(errs, "device_id: null UUID is not allowed")
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

type FetchVehicleParams struct {
	VehicleID  uuid.UUID
	ProviderID uuid.UUID
}

type ListVehiclesParams struct {
	ProviderID uuid.UUID
	Offset     int32
	Limit      int32
}

type VehicleRepository interface {
	FetchVehicle(ctx context.Context, params FetchVehicleParams) (Vehicle, error)
	ListVehicles(ctx context.Context, params ListVehiclesParams) (Page[Vehicle], error)
	InsertVehicle(ctx context.Context, vehicle Vehicle) error
	UpdateVehicle(ctx context.Context, vehicle Vehicle) error
}
