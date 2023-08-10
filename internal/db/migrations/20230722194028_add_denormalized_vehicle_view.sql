-- +goose Up
CREATE OR REPLACE VIEW vehicle_denormalized AS
SELECT
    vehicle.id,
    vehicle.external_id,
    vehicle.provider,
    vehicle.data_provider,
    vehicle_type.name AS vehicle_type,
    vehicle.attributes,
    vehicle.accessibility_attributes,
    vehicle.battery_capacity,
    vehicle.fuel_capacity,
    vehicle.maximum_speed,
    propulsion_type.names_arr AS propulsion_types
FROM vehicle AS vehicle
CROSS JOIN LATERAL (
    SELECT ARRAY_AGG(propulsion_type.name) AS names_arr
    FROM vehicle_propulsion_type AS vpt
    INNER JOIN propulsion_type ON propulsion_type.id = vpt.propulsion_type
    WHERE vpt.vehicle = vehicle.id
    GROUP BY vehicle.id
) AS propulsion_type
INNER JOIN vehicle_type ON vehicle_type.id = vehicle.vehicle_type;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION INSERT_DENORMALIZED_VEHICLE()
RETURNS TRIGGER
AS $$
BEGIN
  INSERT INTO vehicle (
      id,
      external_id,
      provider,
      data_provider,
      vehicle_type,
      attributes,
      accessibility_attributes,
      battery_capacity,
      fuel_capacity,
      maximum_speed
  )
  SELECT
      NEW.id,
      NEW.external_id,
      NEW.provider,
      NEW.data_provider,
      vehicle_type.id,
      NEW.attributes,
      NEW.accessibility_attributes,
      NEW.battery_capacity,
      NEW.fuel_capacity,
      NEW.maximum_speed
  FROM vehicle_type
  WHERE vehicle_type.name = NEW.vehicle_type;
  
  INSERT INTO vehicle_propulsion_type(
    vehicle,
    propulsion_type
  )
  SELECT
    NEW.id AS vehicle,
    propulsion_type.id AS propulsion_type
  FROM propulsion_type
  WHERE propulsion_type.name = ANY(NEW.propulsion_types);
  
  RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE OR REPLACE TRIGGER on_vehicle_denormalized_insert
INSTEAD OF INSERT ON vehicle_denormalized
FOR EACH ROW
EXECUTE PROCEDURE INSERT_DENORMALIZED_VEHICLE();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION UPDATE_DENORMALIZED_VEHICLE()
RETURNS TRIGGER
AS $$
BEGIN
  UPDATE vehicle SET (
      external_id,
      provider,
      data_provider,
      vehicle_type,
      attributes,
      accessibility_attributes,
      battery_capacity,
      fuel_capacity,
      maximum_speed
  ) = (
    SELECT
        NEW.external_id,
        NEW.provider,
        NEW.data_provider,
        vehicle_type.id,
        NEW.attributes,
        NEW.accessibility_attributes,
        NEW.battery_capacity,
        NEW.fuel_capacity,
        NEW.maximum_speed
    FROM vehicle_type
    WHERE vehicle_type.name = NEW.vehicle_type
  ) WHERE id = NEW.id;
  
  -- Remove all existing propulsion type associations for the vehicle and
  -- replace them with the new propulsion types.
  DELETE FROM vehicle_propulsion_type
  WHERE vehicle = NEW.id;

  INSERT INTO vehicle_propulsion_type(
    vehicle,
    propulsion_type
  )
  SELECT
    NEW.id AS vehicle,
    propulsion_type.id AS propulsion_type
  FROM propulsion_type
  WHERE propulsion_type.name = ANY(NEW.propulsion_types);
  
  RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE OR REPLACE TRIGGER on_vehicle_denormalized_update
INSTEAD OF UPDATE ON vehicle_denormalized
FOR EACH ROW
EXECUTE PROCEDURE UPDATE_DENORMALIZED_VEHICLE();
