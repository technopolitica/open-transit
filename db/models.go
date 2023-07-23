// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1

package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/technopolitica/open-mobility/types"
)

type PropulsionType struct {
	ID   uuid.UUID
	Name string
}

type Vehicle struct {
	ID                      uuid.UUID
	ExternalID              pgtype.Text
	Provider                uuid.UUID
	DataProvider            uuid.NullUUID
	VehicleType             uuid.UUID
	Attributes              []byte
	AccessibilityAttributes []byte
	BatteryCapacity         pgtype.Int4
	FuelCapacity            pgtype.Int4
	MaximumSpeed            pgtype.Int4
}

type VehicleDenormalized struct {
	ID                      uuid.UUID
	ExternalID              pgtype.Text
	Provider                uuid.UUID
	DataProvider            uuid.NullUUID
	VehicleType             types.VehicleType
	PropulsionTypes         types.PropulsionTypeSet
	Attributes              types.VehicleAttributes
	AccessibilityAttributes types.AccessibilityAttributes
	BatteryCapacity         pgtype.Int4
	FuelCapacity            pgtype.Int4
	MaximumSpeed            pgtype.Int4
}

type VehiclePropulsionType struct {
	Vehicle        uuid.UUID
	PropulsionType uuid.UUID
}

type VehicleType struct {
	ID   uuid.UUID
	Name string
}
