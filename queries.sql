-- name: FetchVehicle :one
SELECT * FROM vehicle_denormalized
WHERE id = $1;

-- name: RegisterNewVehicles :copyfrom
INSERT INTO vehicle_denormalized (
    id,
    external_id,
    provider,
    data_provider,
    vehicle_type,
    propulsion_types,
    attributes,
    accessibility_attributes,
    battery_capacity,
    fuel_capacity,
    maximum_speed
) VALUES (
    @id,
    @external_id,
    @provider,
    @data_provider,
    @vehicle_type,
    @propulsion_types,
    @attributes,
    @accessibility_attributes,
    @battery_capacity,
    @fuel_capacity,
    @maximum_speed
);