package db

import (
	"github.com/google/uuid"
	"github.com/technopolitica/open-mobility/internal/domain"
)

type VehicleDTO struct {
	ID                      uuid.UUID     `db:"id"`
	ExternalID              string        `db:"external_id"`
	Provider                uuid.UUID     `db:"provider"`
	DataProvider            uuid.UUID     `db:"data_provider"`
	VehicleType             string        `db:"vehicle_type"`
	PropulsionTypes         []string      `db:"propulsion_types"`
	Attributes              domain.Record `db:"attributes"`
	AccessibilityAttributes domain.Record `db:"accessibility_attributes"`
	BatteryCapacity         int32         `db:"battery_capacity"`
	FuelCapacity            int32         `db:"fuel_capacity"`
	MaximumSpeed            int32         `db:"maximum_speed"`
}
