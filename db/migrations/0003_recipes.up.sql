CREATE TABLE recipes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(trim(name))) STORED,
    calories_total numeric(12, 3) NOT NULL DEFAULT 0 CHECK (calories_total >= 0),
    protein_grams_total numeric(12, 3) NOT NULL DEFAULT 0 CHECK (protein_grams_total >= 0),
    carbs_grams_total numeric(12, 3) NOT NULL DEFAULT 0 CHECK (carbs_grams_total >= 0),
    fat_grams_total numeric(12, 3) NOT NULL DEFAULT 0 CHECK (fat_grams_total >= 0),
    source_provider text,
    source_external_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT recipes_source_identity UNIQUE (source_provider, source_external_id)
);

CREATE TABLE recipe_ingredients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id uuid NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    food_item_id uuid NOT NULL REFERENCES food_items(id),
    quantity numeric(12, 3) NOT NULL CHECK (quantity > 0),
    unit text NOT NULL CHECK (unit IN ('gram', 'milliliter', 'piece', 'serving')),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT recipe_ingredients_unique_position UNIQUE (recipe_id, position)
);

CREATE INDEX recipes_user_id_idx ON recipes (user_id);
CREATE INDEX recipes_normalized_name_idx ON recipes (normalized_name);
CREATE INDEX recipe_ingredients_recipe_id_idx ON recipe_ingredients (recipe_id);
CREATE INDEX recipe_ingredients_food_item_id_idx ON recipe_ingredients (food_item_id);
