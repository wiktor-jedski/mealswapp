-- Implements DESIGN-005 FoodItemEntity.
-- Implements DESIGN-005 MacroNormalizer.
-- Implements DESIGN-005 MicronutrientVocabulary.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS food_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    physical_state text NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time_minutes integer NOT NULL DEFAULT 0 CHECK (prep_time_minutes >= 0),
    average_unit_weight_grams numeric(12, 4) CHECK (average_unit_weight_grams IS NULL OR average_unit_weight_grams > 0),
    average_serving_volume_milliliters numeric(12, 4) CHECK (average_serving_volume_milliliters IS NULL OR average_serving_volume_milliliters > 0),
    density_grams_per_milliliter numeric(12, 6) CHECK (density_grams_per_milliliter IS NULL OR density_grams_per_milliliter > 0),
    density_source_provider text,
    density_source_food_id text,
    density_source_kind text CHECK (density_source_kind IS NULL OR density_source_kind IN ('imported', 'manual', 'estimated')),
    protein_per_100 numeric(12, 4) NOT NULL CHECK (protein_per_100 >= 0),
    carbohydrates_per_100 numeric(12, 4) NOT NULL CHECK (carbohydrates_per_100 >= 0),
    fat_per_100 numeric(12, 4) NOT NULL CHECK (fat_per_100 >= 0),
    micronutrients jsonb NOT NULL DEFAULT '{}'::jsonb,
    image_url text,
    image_alt text,
    source_provider text,
    external_id text,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT food_items_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT food_items_micronutrients_object CHECK (jsonb_typeof(micronutrients) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS food_items_active_normalized_name_idx
    ON food_items (normalized_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS food_items_name_search_idx
    ON food_items (normalized_name text_pattern_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS food_items_physical_state_idx
    ON food_items (physical_state)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS food_items_prep_time_idx
    ON food_items (prep_time_minutes)
    WHERE deleted_at IS NULL;

INSERT INTO schema_migrations (version)
VALUES (2)
ON CONFLICT (version) DO NOTHING;
