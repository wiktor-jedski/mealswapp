CREATE TABLE meals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (lower(trim(name))) STORED,
    meal_type text NOT NULL CHECK (meal_type IN ('single', 'recipe')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE meal_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_id uuid NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    food_item_id uuid NOT NULL REFERENCES food_items(id),
    quantity numeric(12, 3) NOT NULL CHECK (quantity > 0),
    unit text NOT NULL CHECK (unit IN ('gram', 'milliliter', 'piece', 'serving')),
    position integer NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT meal_items_unique_position UNIQUE (meal_id, position)
);

CREATE INDEX meals_user_id_idx ON meals (user_id);
CREATE INDEX meals_normalized_name_idx ON meals (normalized_name);
CREATE INDEX meal_items_meal_id_idx ON meal_items (meal_id);
CREATE INDEX meal_items_food_item_id_idx ON meal_items (food_item_id);
