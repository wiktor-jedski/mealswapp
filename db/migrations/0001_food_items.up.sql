CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE food_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(trim(name))) STORED,
    physical_state text NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    serving_unit text NOT NULL CHECK (serving_unit IN ('gram', 'milliliter', 'piece', 'serving')),
    serving_size numeric(12, 3) NOT NULL CHECK (serving_size > 0),
    calories_per_100 numeric(12, 3) NOT NULL CHECK (calories_per_100 >= 0),
    protein_grams_per_100 numeric(12, 3) NOT NULL CHECK (protein_grams_per_100 >= 0),
    carbs_grams_per_100 numeric(12, 3) NOT NULL CHECK (carbs_grams_per_100 >= 0),
    fat_grams_per_100 numeric(12, 3) NOT NULL CHECK (fat_grams_per_100 >= 0),
    micronutrients jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_provider text,
    source_external_id text,
    source_url text,
    source_imported_at timestamptz,
    curation_state text NOT NULL DEFAULT 'draft' CHECK (curation_state IN ('draft', 'approved', 'rejected', 'inactive')),
    image_url text,
    prep_time_minutes integer NOT NULL DEFAULT 0 CHECK (prep_time_minutes >= 0),
    average_unit_weight_grams numeric(12, 3) NOT NULL DEFAULT 0 CHECK (average_unit_weight_grams >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT food_items_micronutrients_object CHECK (jsonb_typeof(micronutrients) = 'object'),
    CONSTRAINT food_items_source_identity UNIQUE (source_provider, source_external_id)
);

CREATE UNIQUE INDEX food_items_normalized_name_unique_idx ON food_items (normalized_name);
CREATE INDEX food_items_name_idx ON food_items USING btree (name);
CREATE INDEX food_items_physical_state_idx ON food_items (physical_state);
CREATE INDEX food_items_curation_state_idx ON food_items (curation_state);
CREATE INDEX food_items_micronutrients_gin_idx ON food_items USING gin (micronutrients);
