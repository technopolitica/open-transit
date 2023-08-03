-- name: FetchVehicle :one
SELECT * FROM vehicle_denormalized
WHERE id = @vehicle_id AND provider = @provider_id;

-- name: ListVehicles :many
SELECT * FROM vehicle_denormalized
WHERE provider = @provider_id
LIMIT @limit_::int
OFFSET @offset_::int;

-- name: ListVehiclesCount :one
SELECT COUNT(*) FROM vehicle_denormalized
WHERE provider = @provider_id;

-- name: UpdateVehicle :one
UPDATE vehicle_denormalized SET
    external_id = @external_id,
    data_provider = @data_provider,
    vehicle_type = @vehicle_type,
    propulsion_types = @propulsion_types,
    attributes = @attributes,
    accessibility_attributes = @accessibility_attributes,
    battery_capacity = @battery_capacity,
    fuel_capacity = @fuel_capacity,
    maximum_speed = @maximum_speed
WHERE id = @id AND provider = @provider
RETURNING id;

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