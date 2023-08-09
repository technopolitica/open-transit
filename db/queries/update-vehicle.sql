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
