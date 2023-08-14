package db

import (
	"context"
	"errors"
	"fmt"

	_ "embed"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/technopolitica/open-mobility/internal/domain"
)

func dtoFromVehicle(domainVehicle domain.Vehicle) VehicleDTO {
	return VehicleDTO{
		ID:                      domainVehicle.DeviceID,
		ExternalID:              domainVehicle.VehicleID,
		Provider:                domainVehicle.ProviderID,
		DataProvider:            domainVehicle.DataProviderID,
		VehicleType:             domainVehicle.VehicleType.String(),
		Attributes:              domainVehicle.VehicleAttributes,
		AccessibilityAttributes: domainVehicle.AccessibilityAttributes,
		PropulsionTypes:         domain.Stringify(domainVehicle.PropulsionTypes),
		BatteryCapacity:         int32(domainVehicle.BatteryCapacity),
		FuelCapacity:            int32(domainVehicle.FuelCapacity),
		MaximumSpeed:            int32(domainVehicle.MaximumSpeed),
	}
}

func vehicleFromDTO(vehicle VehicleDTO) domain.Vehicle {
	// FIXME: how should we handle parsing errors?
	vehicleType, _ := domain.ParseVehicleType(vehicle.VehicleType)
	propulsionTypes := make([]domain.PropulsionType, 0, len(vehicle.PropulsionTypes))
	for _, pt := range vehicle.PropulsionTypes {
		ptParsed, _ := domain.ParsePropulsionType(pt)
		propulsionTypes = append(propulsionTypes, ptParsed)
	}
	return domain.Vehicle{
		DeviceID:                vehicle.ID,
		VehicleID:               vehicle.ExternalID,
		ProviderID:              vehicle.Provider,
		DataProviderID:          vehicle.DataProvider,
		VehicleType:             vehicleType,
		PropulsionTypes:         propulsionTypes,
		VehicleAttributes:       vehicle.Attributes,
		AccessibilityAttributes: vehicle.AccessibilityAttributes,
		BatteryCapacity:         int(vehicle.BatteryCapacity),
		FuelCapacity:            int(vehicle.FuelCapacity),
		MaximumSpeed:            int(vehicle.MaximumSpeed),
	}
}

//go:embed queries/fetch-vehicle.sql
var fetchVehicleQuery string

func (repo Repository) FetchVehicle(ctx context.Context, params domain.FetchVehicleParams) (vehicle domain.Vehicle, err error) {
	rows, err := repo.Query(ctx, fetchVehicleQuery, pgx.NamedArgs{"id": params.VehicleID, "provider": params.ProviderID})
	if err != nil {
		err = fmt.Errorf("failed to execute query: %w", err)
		return
	}

	vehicleDTOs, err := pgx.CollectRows(rows, pgx.RowToStructByName[VehicleDTO])
	if err != nil {
		err = fmt.Errorf("failed to map row to VehicleDTO: %w", err)
		return
	}
	if len(vehicleDTOs) == 0 {
		err = ErrNotFound
		return
	}

	vehicle = vehicleFromDTO(vehicleDTOs[0])
	return
}

//go:embed queries/list-vehicles.sql
var listVehiclesQuery string

//go:embed queries/count-vehicles.sql
var countVehiclesQuery string

func (repo Repository) ListVehicles(ctx context.Context, arg domain.ListVehiclesParams) (page domain.Page[domain.Vehicle], err error) {
	repo.WithinTransaction(ctx, func(tx pgx.Tx) (err error) {
		rows, err := tx.Query(ctx, listVehiclesQuery, pgx.NamedArgs{"provider": arg.ProviderID, "limit": arg.Limit, "offset": arg.Offset})
		if err != nil {
			return
		}
		defer rows.Close()
		vehicleDTOs, err := pgx.CollectRows(rows, pgx.RowToStructByName[VehicleDTO])
		page.Items = make([]domain.Vehicle, 0, len(vehicleDTOs))
		for _, dto := range vehicleDTOs {
			page.Items = append(page.Items, vehicleFromDTO(dto))
		}

		row := tx.QueryRow(ctx, countVehiclesQuery, pgx.NamedArgs{"provider": arg.ProviderID})
		row.Scan(&page.Total)
		return
	})
	return
}

//go:embed queries/insert-vehicle.sql
var insertVehicleQuery string

func (repo Repository) InsertVehicle(ctx context.Context, vehicle domain.Vehicle) error {
	vehicleDTO := dtoFromVehicle(vehicle)
	_, err := repo.Exec(ctx, insertVehicleQuery, pgx.NamedArgs{
		"id":                       vehicleDTO.ID,
		"provider":                 vehicleDTO.Provider,
		"external_id":              vehicleDTO.ExternalID,
		"data_provider":            vehicleDTO.DataProvider,
		"vehicle_type":             vehicleDTO.VehicleType,
		"propulsion_types":         vehicleDTO.PropulsionTypes,
		"attributes":               vehicleDTO.Attributes,
		"accessibility_attributes": vehicleDTO.AccessibilityAttributes,
		"battery_capacity":         vehicleDTO.BatteryCapacity,
		"fuel_capacity":            vehicleDTO.FuelCapacity,
		"maximum_speed":            vehicleDTO.MaximumSpeed,
	})

	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) && pgErr.ColumnName == "id" && pgErr.Code == pgerrcode.UniqueViolation {
		return ErrConflict
	}

	return err
}

//go:embed queries/update-vehicle.sql
var updateVehicleQuery string

func (repo Repository) UpdateVehicle(ctx context.Context, vehicle domain.Vehicle) error {
	vehicleDTO := dtoFromVehicle(vehicle)
	res, err := repo.Exec(ctx, updateVehicleQuery, pgx.NamedArgs{
		"id":                       vehicleDTO.ID,
		"provider":                 vehicleDTO.Provider,
		"external_id":              vehicleDTO.ExternalID,
		"data_provider":            vehicleDTO.DataProvider,
		"vehicle_type":             vehicleDTO.VehicleType,
		"propulsion_types":         vehicleDTO.PropulsionTypes,
		"attributes":               vehicleDTO.Attributes,
		"accessibility_attributes": vehicleDTO.AccessibilityAttributes,
		"battery_capacity":         vehicleDTO.BatteryCapacity,
		"fuel_capacity":            vehicleDTO.FuelCapacity,
		"maximum_speed":            vehicleDTO.MaximumSpeed,
	})

	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}

	return err
}
