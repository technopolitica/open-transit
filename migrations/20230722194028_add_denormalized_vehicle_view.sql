CREATE VIEW vehicle_denormalized AS
SELECT
    v.id,
    v.external_id,
    v.provider,
    v.data_provider,
    vt.name AS vehicle_type,
    array_agg(pt.name)::TEXT[] AS propulsion_types,
    v.attributes,
    v.accessibility_attributes,
    v.battery_capacity,
    v.fuel_capacity,
    v.maximum_speed
FROM vehicle AS v
  JOIN vehicle_type AS vt ON vt.id = v.vehicle_type
  JOIN vehicle_propulsion_type AS vpt ON vpt.vehicle = v.id
  JOIN propulsion_type AS pt ON pt.id = vpt.propulsion_type
GROUP BY v.id, vt.name;

CREATE OR REPLACE FUNCTION insert_denormalized_vehicle()
  RETURNS trigger
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
      vt.id,
      NEW.attributes,
      NEW.accessibility_attributes,
      NEW.battery_capacity,
      NEW.fuel_capacity,
      NEW.maximum_speed
  FROM vehicle_type AS vt
  WHERE vt.name = NEW.vehicle_type;
  
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

CREATE TRIGGER on_vehicle_denormalized_insert
INSTEAD OF INSERT ON vehicle_denormalized
FOR EACH ROW
EXECUTE PROCEDURE insert_denormalized_vehicle();

CREATE OR REPLACE FUNCTION update_denormalized_vehicle()
  RETURNS trigger
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
        vt.id,
        NEW.attributes,
        NEW.accessibility_attributes,
        NEW.battery_capacity,
        NEW.fuel_capacity,
        NEW.maximum_speed
    FROM vehicle_type AS vt
    WHERE vt.name = NEW.vehicle_type
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

CREATE TRIGGER on_vehicle_denormalized_update
INSTEAD OF UPDATE ON vehicle_denormalized
FOR EACH ROW
EXECUTE PROCEDURE update_denormalized_vehicle();