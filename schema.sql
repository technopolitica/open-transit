CREATE TABLE vehicle_type (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid() CHECK (id <> '00000000-0000-0000-0000-000000000000'),
  name TEXT NOT NULL CHECK (name <> '')
);

CREATE TABLE propulsion_type (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid() CHECK (id <> '00000000-0000-0000-0000-000000000000'),
  name TEXT NOT NULL CHECK (name <> '')
);

CREATE TABLE vehicle (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid() CHECK (id <> '00000000-0000-0000-0000-000000000000'),
  external_id TEXT CHECK (external_id <> ''),
  provider UUID NOT NULL CHECK (provider <> '00000000-0000-0000-0000-000000000000'),
  data_provider UUID NOT NULL,
  vehicle_type UUID NOT NULL REFERENCES vehicle_type(id),
  attributes JSONB DEFAULT '{}',
  accessibility_attributes JSONB DEFAULT '{}',
  battery_capacity INTEGER,
  fuel_capacity INTEGER,
  maximum_speed INTEGER
);

CREATE TABLE vehicle_propulsion_type (
  vehicle UUID NOT NULL REFERENCES vehicle(id),
  propulsion_type UUID NOT NULL REFERENCES propulsion_type(id),
  PRIMARY KEY (vehicle, propulsion_type)
);

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
