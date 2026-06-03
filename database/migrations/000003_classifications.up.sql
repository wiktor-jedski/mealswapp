-- Implements DESIGN-005 ClassificationEntity.
CREATE TABLE IF NOT EXISTS classifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    kind text NOT NULL CHECK (kind IN ('food_category', 'culinary_role')),
    parent_id uuid REFERENCES classifications(id) ON DELETE RESTRICT,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT classifications_name_not_blank CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS classifications_active_root_name_idx
    ON classifications (kind, normalized_name)
    WHERE parent_id IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS classifications_active_child_name_idx
    ON classifications (kind, parent_id, normalized_name)
    WHERE parent_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS classifications_kind_parent_idx
    ON classifications (kind, parent_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS food_item_classifications (
    food_item_id uuid NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    classification_id uuid NOT NULL REFERENCES classifications(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (food_item_id, classification_id)
);

CREATE INDEX IF NOT EXISTS food_item_classifications_classification_idx
    ON food_item_classifications (classification_id);

CREATE TABLE IF NOT EXISTS meal_classifications (
    meal_id uuid NOT NULL,
    classification_id uuid NOT NULL REFERENCES classifications(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (meal_id, classification_id)
);

CREATE INDEX IF NOT EXISTS meal_classifications_classification_idx
    ON meal_classifications (classification_id);

INSERT INTO schema_migrations (version)
VALUES (3)
ON CONFLICT (version) DO NOTHING;
