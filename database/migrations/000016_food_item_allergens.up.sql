-- Implements DESIGN-002 FilterProcessor named allergen repository filters.
CREATE TABLE IF NOT EXISTS food_item_allergens (
    food_item_id uuid NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    allergen_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (food_item_id, allergen_key),
    CONSTRAINT food_item_allergens_key_not_blank CHECK (btrim(allergen_key) <> ''),
    CONSTRAINT food_item_allergens_key_normalized CHECK (allergen_key = lower(btrim(allergen_key)))
);

CREATE INDEX IF NOT EXISTS food_item_allergens_key_idx
    ON food_item_allergens (allergen_key);

INSERT INTO schema_migrations (version)
VALUES (16)
ON CONFLICT (version) DO NOTHING;
