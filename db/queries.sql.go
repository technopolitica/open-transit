// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1
// source: queries.sql

package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/technopolitica/open-mobility/types"
)

const fetchVehicle = `-- name: FetchVehicle :one
SELECT id, external_id, provider, data_provider, vehicle_type, propulsion_types, attributes, accessibility_attributes, battery_capacity, fuel_capacity, maximum_speed FROM vehicle_denormalized
WHERE id = $1
`

func (q *Queries) FetchVehicle(ctx context.Context, id uuid.UUID) (VehicleDenormalized, error) {
	row := q.db.QueryRow(ctx, fetchVehicle, id)
	var i VehicleDenormalized
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.Provider,
		&i.DataProvider,
		&i.VehicleType,
		&i.PropulsionTypes,
		&i.Attributes,
		&i.AccessibilityAttributes,
		&i.BatteryCapacity,
		&i.FuelCapacity,
		&i.MaximumSpeed,
	)
	return i, err
}

type RegisterNewVehiclesParams struct {
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
