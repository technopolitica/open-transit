SELECT
    id,
    external_id,
    provider,
    data_provider,
    vehicle_type,
    attributes,
    accessibility_attributes,
    battery_capacity,
    fuel_capacity,
    maximum_speed,
    propulsion_types
FROM vehicle_denormalized
WHERE id = @id AND provider = @provider;
