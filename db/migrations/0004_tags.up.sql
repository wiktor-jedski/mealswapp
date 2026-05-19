CREATE TABLE tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(trim(name))) STORED,
    kind text NOT NULL CHECK (kind IN ('diet', 'allergen', 'functionality', 'curation')),
    parent_id uuid REFERENCES tags(id) ON DELETE SET NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tags_unique_kind_name UNIQUE (kind, normalized_name)
);

CREATE TABLE food_item_tags (
    food_item_id uuid NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (food_item_id, tag_id)
);

CREATE INDEX tags_kind_idx ON tags (kind);
CREATE INDEX tags_parent_id_idx ON tags (parent_id);
CREATE INDEX food_item_tags_tag_id_idx ON food_item_tags (tag_id);
