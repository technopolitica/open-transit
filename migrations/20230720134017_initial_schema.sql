-- Create "vehicle_type" table
CREATE TABLE "public"."vehicle_type" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "name" text NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "vehicle_type_name_check" CHECK (name <> '' :: text)
);

-- Create "vehicle" table
CREATE TABLE "public"."vehicle" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "external_id" text NULL,
    "provider" uuid NOT NULL,
    "data_provider" uuid NULL,
    "vehicle_type" uuid NOT NULL,
    "attributes" jsonb NULL DEFAULT '{}',
    "accessibility_attributes" jsonb NULL DEFAULT '{}',
    "battery_capacity" integer NULL,
    "fuel_capacity" integer NULL,
    "maximum_speed" integer NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "vehicle_vehicle_type_fkey" FOREIGN KEY ("vehicle_type") REFERENCES "public"."vehicle_type" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "vehicle_external_id_check" CHECK (external_id <> '' :: text)
);

-- Create "propulsion_type" table
CREATE TABLE "public"."propulsion_type" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "name" text NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "propulsion_type_name_check" CHECK (name <> '' :: text)
);

-- Create "vehicle_propulsion_type" table
CREATE TABLE "public"."vehicle_propulsion_type" (
    "vehicle" uuid NOT NULL,
    "propulsion_type" uuid NOT NULL,
    PRIMARY KEY ("vehicle", "propulsion_type"),
    CONSTRAINT "vehicle_propulsion_type_propulsion_type_fkey" FOREIGN KEY ("propulsion_type") REFERENCES "public"."propulsion_type" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "vehicle_propulsion_type_vehicle_fkey" FOREIGN KEY ("vehicle") REFERENCES "public"."vehicle" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);