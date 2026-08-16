-- Implements DESIGN-009 TagManager persisted allergen filter-option vocabulary.
CREATE TABLE IF NOT EXISTS allergen_vocabulary (
    key text PRIMARY KEY,
    name text NOT NULL,
    label_key text NOT NULL UNIQUE,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT allergen_vocabulary_key_normalized CHECK (key = lower(btrim(key)) AND key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT allergen_vocabulary_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT allergen_vocabulary_label_key_not_blank CHECK (btrim(label_key) <> '')
);

INSERT INTO allergen_vocabulary (key, name, label_key)
VALUES
    ('animal_product', 'Animal products', 'filter.allergen.animal_product'),
    ('dairy', 'Dairy', 'filter.allergen.dairy'),
    ('egg', 'Egg', 'filter.allergen.egg'),
    ('gluten', 'Gluten', 'filter.allergen.gluten'),
    ('meat', 'Meat', 'filter.allergen.meat'),
    ('peanut', 'Peanut', 'filter.allergen.peanut'),
    ('tree_nut', 'Tree nuts', 'filter.allergen.tree_nut')
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name,
    label_key = EXCLUDED.label_key,
    updated_at = now();

INSERT INTO schema_migrations (version)
VALUES (27)
ON CONFLICT (version) DO NOTHING;
