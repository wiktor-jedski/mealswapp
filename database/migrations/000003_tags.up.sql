-- Implements DESIGN-005 TagEntity.
CREATE TABLE IF NOT EXISTS tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    kind text NOT NULL CHECK (kind IN ('category', 'functionality')),
    parent_id uuid REFERENCES tags(id) ON DELETE RESTRICT,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tags_name_not_blank CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS tags_active_root_name_idx
    ON tags (kind, normalized_name)
    WHERE parent_id IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS tags_active_child_name_idx
    ON tags (kind, parent_id, normalized_name)
    WHERE parent_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS tags_kind_parent_idx
    ON tags (kind, parent_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS food_item_tags (
    food_item_id uuid NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (food_item_id, tag_id)
);

CREATE INDEX IF NOT EXISTS food_item_tags_tag_idx
    ON food_item_tags (tag_id);

CREATE TABLE IF NOT EXISTS meal_tags (
    meal_id uuid NOT NULL,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (meal_id, tag_id)
);

CREATE INDEX IF NOT EXISTS meal_tags_tag_idx
    ON meal_tags (tag_id);

INSERT INTO schema_migrations (version)
VALUES (3)
ON CONFLICT (version) DO NOTHING;
