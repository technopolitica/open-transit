-- +goose Up
CREATE TABLE IF NOT EXISTS vehicle_type (
    name TEXT PRIMARY KEY UNIQUE CHECK (name != '')
);

INSERT INTO
vehicle_type (name)
VALUES
('bicycle'),
('bus'),
('cargo_bicycle'),
('car'),
('delivery_robot'),
('moped'),
('motorcycle'),
('scooter_standing'),
('scooter_seated'),
('truck'),
('other')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS propulsion_type (
    name TEXT PRIMARY KEY CHECK (name != '')
);

INSERT INTO
propulsion_type (name)
VALUES
('human'),
('electric_assist'),
('electric'),
('combustion'),
('combustion_diesel'),
('hybrid'),
('hydrogen_fuel_cell'),
('plug_in_hybrid')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS vehicle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid() CHECK (
        id != '00000000-0000-0000-0000-000000000000'
    ),
    external_id TEXT NOT NULL DEFAULT '',
    provider UUID NOT NULL CHECK (
        provider != '00000000-0000-0000-0000-000000000000'
    ),
    data_provider UUID NOT NULL CHECK (
        id != '00000000-0000-0000-0000-000000000000'
    ),
    vehicle_type TEXT NOT NULL REFERENCES vehicle_type (name) ON UPDATE CASCADE,
    attributes JSONB NOT NULL DEFAULT '{}' CHECK (
        jsonb_typeof(attributes) = 'object'
    ),
    accessibility_attributes JSONB NOT NULL DEFAULT '{}' CHECK (
        jsonb_typeof(attributes) = 'object'
    ),
    battery_capacity INTEGER NOT NULL DEFAULT 0,
    fuel_capacity INTEGER NOT NULL DEFAULT 0,
    maximum_speed INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS vehicle_propulsion_type (
    vehicle UUID NOT NULL REFERENCES vehicle (id),
    propulsion_type TEXT NOT NULL REFERENCES propulsion_type (
        name
    ) ON UPDATE CASCADE,
    PRIMARY KEY (vehicle, propulsion_type)
);
