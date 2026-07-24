-- Implements DESIGN-005 FoodItemEntity owner-scoped custom-item persistence.
CREATE TABLE IF NOT EXISTS custom_food_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT custom_food_items_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT custom_food_items_micronutrients_object CHECK (jsonb_typeof(micronutrients) = 'object'),
    CONSTRAINT custom_food_items_liquid_density_required CHECK (
        (physical_state = 'liquid'
            AND density_grams_per_milliliter IS NOT NULL
            AND density_source_kind IN ('imported', 'manual', 'estimated'))
        OR (physical_state = 'solid'
            AND average_serving_volume_milliliters IS NULL
            AND density_grams_per_milliliter IS NULL
            AND density_source_provider IS NULL
            AND density_source_food_id IS NULL
            AND density_source_kind IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS custom_food_items_owner_active_name_idx
    ON custom_food_items (owner_id, normalized_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS custom_food_items_owner_idx
    ON custom_food_items (owner_id, id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS custom_food_item_classifications (
    custom_food_item_id uuid NOT NULL REFERENCES custom_food_items(id) ON DELETE CASCADE,
    classification_id uuid NOT NULL REFERENCES classifications(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (custom_food_item_id, classification_id)
);

CREATE INDEX IF NOT EXISTS custom_food_item_classifications_classification_idx
    ON custom_food_item_classifications (classification_id);

INSERT INTO schema_migrations (version)
VALUES (25)
ON CONFLICT (version) DO NOTHING;
