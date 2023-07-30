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
WHERE id = $1 AND provider = $2
`

type FetchVehicleParams struct {
	VehicleID  uuid.UUID
	ProviderID uuid.UUID
}

func (q *Queries) FetchVehicle(ctx context.Context, arg FetchVehicleParams) (VehicleDenormalized, error) {
	row := q.db.QueryRow(ctx, fetchVehicle, arg.VehicleID, arg.ProviderID)
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

const listVehicles = `-- name: ListVehicles :many
SELECT id, external_id, provider, data_provider, vehicle_type, propulsion_types, attributes, accessibility_attributes, battery_capacity, fuel_capacity, maximum_speed FROM vehicle_denormalized
WHERE provider = $1
LIMIT $3::int
OFFSET $2::int
`

type ListVehiclesParams struct {
	ProviderID uuid.UUID
	Offset     int32
	Limit      int32
}

func (q *Queries) ListVehicles(ctx context.Context, arg ListVehiclesParams) ([]VehicleDenormalized, error) {
	rows, err := q.db.Query(ctx, listVehicles, arg.ProviderID, arg.Offset, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VehicleDenormalized
	for rows.Next() {
		var i VehicleDenormalized
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listVehiclesCount = `-- name: ListVehiclesCount :one
SELECT COUNT(*) FROM vehicle_denormalized
WHERE provider = $1
`

func (q *Queries) ListVehiclesCount(ctx context.Context, providerID uuid.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, listVehiclesCount, providerID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

type RegisterNewVehiclesParams struct {
	ID                      uuid.UUID
	ExternalID              pgtype.Text
	Provider                uuid.UUID
	DataProvider            uuid.NullUUID
	VehicleType             types.VehicleType
	PropulsionTypes         types.Set[types.PropulsionType]
	Attributes              types.VehicleAttributes
	AccessibilityAttributes types.AccessibilityAttributes
	BatteryCapacity         pgtype.Int4
	FuelCapacity            pgtype.Int4
	MaximumSpeed            pgtype.Int4
}
