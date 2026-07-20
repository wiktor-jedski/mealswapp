-- Implements DESIGN-005 UnitConverter canonical and recipe/per-unit boundaries.
-- Implements DESIGN-008 SavedDataRepository quantity-unit persistence boundary.
ALTER TABLE recipe_ingredients
    DROP CONSTRAINT IF EXISTS recipe_ingredients_unit_not_blank;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'recipe_ingredients_unit_canonical'
          AND conrelid = 'recipe_ingredients'::regclass
    ) THEN
        ALTER TABLE recipe_ingredients
            ADD CONSTRAINT recipe_ingredients_unit_canonical
            CHECK (unit IN ('g', 'ml', 'oz', 'fl_oz', 'serving'));
    END IF;
END;
$$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM saved_diet_meal_entries entry
        JOIN meals meal ON meal.id = entry.meal_id
        WHERE meal.physical_state = 'solid' AND entry.unit NOT IN ('g', 'oz')
           OR meal.physical_state = 'liquid' AND entry.unit NOT IN ('ml', 'fl_oz')
    ) THEN
        RAISE EXCEPTION 'saved diet entries contain cross-basis quantity units'
            USING ERRCODE = '23514';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM recipe_ingredients ingredient
        JOIN food_items food ON food.id = ingredient.food_item_id
        WHERE food.physical_state = 'solid' AND ingredient.unit NOT IN ('g', 'oz', 'serving')
           OR food.physical_state = 'liquid' AND ingredient.unit NOT IN ('ml', 'fl_oz', 'serving')
    ) THEN
        RAISE EXCEPTION 'recipe ingredients contain cross-basis quantity units'
            USING ERRCODE = '23514';
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION validate_saved_diet_entry_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    meal_state text;
BEGIN
    SELECT physical_state INTO meal_state FROM meals WHERE id = NEW.meal_id;
    IF meal_state = 'solid' AND NEW.unit NOT IN ('g', 'oz')
       OR meal_state = 'liquid' AND NEW.unit NOT IN ('ml', 'fl_oz') THEN
        RAISE EXCEPTION 'saved diet unit % does not match meal physical state %', NEW.unit, meal_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
CREATE TRIGGER saved_diet_entry_unit_basis_trigger
BEFORE INSERT OR UPDATE OF meal_id, unit ON saved_diet_meal_entries
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_entry_unit_basis();

CREATE OR REPLACE FUNCTION validate_saved_diet_meal_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM saved_diet_meal_entries
        WHERE meal_id = NEW.id
          AND (NEW.physical_state = 'solid' AND unit NOT IN ('g', 'oz')
               OR NEW.physical_state = 'liquid' AND unit NOT IN ('ml', 'fl_oz'))
    ) THEN
        RAISE EXCEPTION 'meal physical state % conflicts with saved diet quantities', NEW.physical_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_meal_unit_basis_trigger ON meals;
CREATE TRIGGER saved_diet_meal_unit_basis_trigger
BEFORE UPDATE OF physical_state ON meals
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_meal_unit_basis();

CREATE OR REPLACE FUNCTION validate_recipe_ingredient_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    food_state text;
BEGIN
    SELECT physical_state INTO food_state FROM food_items WHERE id = NEW.food_item_id;
    IF food_state = 'solid' AND NEW.unit NOT IN ('g', 'oz', 'serving')
       OR food_state = 'liquid' AND NEW.unit NOT IN ('ml', 'fl_oz', 'serving') THEN
        RAISE EXCEPTION 'recipe unit % does not match ingredient physical state %', NEW.unit, food_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS recipe_ingredient_unit_basis_trigger ON recipe_ingredients;
CREATE TRIGGER recipe_ingredient_unit_basis_trigger
BEFORE INSERT OR UPDATE OF food_item_id, unit ON recipe_ingredients
FOR EACH ROW
EXECUTE FUNCTION validate_recipe_ingredient_unit_basis();

CREATE OR REPLACE FUNCTION validate_food_recipe_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM recipe_ingredients
        WHERE food_item_id = NEW.id
          AND (NEW.physical_state = 'solid' AND unit NOT IN ('g', 'oz', 'serving')
               OR NEW.physical_state = 'liquid' AND unit NOT IN ('ml', 'fl_oz', 'serving'))
    ) THEN
        RAISE EXCEPTION 'food physical state % conflicts with recipe quantities', NEW.physical_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS food_recipe_unit_basis_trigger ON food_items;
CREATE TRIGGER food_recipe_unit_basis_trigger
BEFORE UPDATE OF physical_state ON food_items
FOR EACH ROW
EXECUTE FUNCTION validate_food_recipe_unit_basis();

INSERT INTO schema_migrations (version)
VALUES (22)
ON CONFLICT (version) DO NOTHING;
