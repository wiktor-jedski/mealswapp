-- Implements DESIGN-005 MealEntity.
-- Implements DESIGN-005 RecipeEntity.
CREATE TABLE IF NOT EXISTS meals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type text NOT NULL CHECK (type IN ('single', 'composite')),
    name text NOT NULL CHECK (btrim(name) <> ''),
    physical_state text NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time_minutes integer NOT NULL DEFAULT 0 CHECK (prep_time_minutes >= 0),
    average_unit_weight_grams numeric(12, 4) CHECK (average_unit_weight_grams IS NULL OR average_unit_weight_grams > 0),
    protein_per_100 numeric(12, 4) CHECK (protein_per_100 IS NULL OR protein_per_100 >= 0),
    carbohydrates_per_100 numeric(12, 4) CHECK (carbohydrates_per_100 IS NULL OR carbohydrates_per_100 >= 0),
    fat_per_100 numeric(12, 4) CHECK (fat_per_100 IS NULL OR fat_per_100 >= 0),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT meals_single_macros_required CHECK (
        (type = 'single' AND protein_per_100 IS NOT NULL AND carbohydrates_per_100 IS NOT NULL AND fat_per_100 IS NOT NULL)
        OR (type = 'composite' AND protein_per_100 IS NULL AND carbohydrates_per_100 IS NULL AND fat_per_100 IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS meals_type_idx
    ON meals (type)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS recipe_ingredients (
    meal_id uuid NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    food_item_id uuid NOT NULL REFERENCES food_items(id) ON DELETE RESTRICT,
    quantity numeric(12, 4) NOT NULL CHECK (quantity > 0),
    unit text NOT NULL,
    position integer NOT NULL CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (meal_id, position),
    CONSTRAINT recipe_ingredients_unit_not_blank CHECK (btrim(unit) <> '')
);

CREATE INDEX IF NOT EXISTS recipe_ingredients_food_item_idx
    ON recipe_ingredients (food_item_id);

CREATE OR REPLACE FUNCTION ensure_recipe_meal_has_ingredients()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    checked_meal_id uuid;
BEGIN
    IF TG_TABLE_NAME = 'meals' THEN
        checked_meal_id := COALESCE(NEW.id, OLD.id);
    ELSE
        checked_meal_id := COALESCE(NEW.meal_id, OLD.meal_id);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM meals
        WHERE id = checked_meal_id
          AND type = 'composite'
          AND deleted_at IS NULL
    )
    AND NOT EXISTS (
        SELECT 1
        FROM recipe_ingredients
        WHERE meal_id = checked_meal_id
    ) THEN
        RAISE EXCEPTION 'composite meal % requires at least one ingredient', checked_meal_id
            USING ERRCODE = '23514';
    END IF;

    RETURN COALESCE(NEW, OLD);
END $$;

DROP TRIGGER IF EXISTS meals_recipe_ingredient_required ON meals;
CREATE CONSTRAINT TRIGGER meals_recipe_ingredient_required
AFTER INSERT OR UPDATE ON meals
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION ensure_recipe_meal_has_ingredients();

DROP TRIGGER IF EXISTS recipe_ingredients_recipe_required ON recipe_ingredients;
CREATE CONSTRAINT TRIGGER recipe_ingredients_recipe_required
AFTER DELETE OR UPDATE ON recipe_ingredients
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION ensure_recipe_meal_has_ingredients();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'meal_tags_meal_id_fkey'
          AND conrelid = 'meal_tags'::regclass
    ) THEN
        ALTER TABLE meal_tags
            ADD CONSTRAINT meal_tags_meal_id_fkey
            FOREIGN KEY (meal_id) REFERENCES meals(id) ON DELETE CASCADE;
    END IF;
END $$;

INSERT INTO schema_migrations (version)
VALUES (5)
ON CONFLICT (version) DO NOTHING;
