-- Implements DESIGN-005 UnitConverter canonical database boundary rollback.
DROP TRIGGER IF EXISTS food_recipe_unit_basis_trigger ON food_items;
DROP FUNCTION IF EXISTS validate_food_recipe_unit_basis();
DROP TRIGGER IF EXISTS recipe_ingredient_unit_basis_trigger ON recipe_ingredients;
DROP FUNCTION IF EXISTS validate_recipe_ingredient_unit_basis();
DROP TRIGGER IF EXISTS saved_diet_meal_unit_basis_trigger ON meals;
DROP FUNCTION IF EXISTS validate_saved_diet_meal_unit_basis();
DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
DROP FUNCTION IF EXISTS validate_saved_diet_entry_unit_basis();

ALTER TABLE recipe_ingredients
    DROP CONSTRAINT IF EXISTS recipe_ingredients_unit_canonical;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'recipe_ingredients_unit_not_blank'
          AND conrelid = 'recipe_ingredients'::regclass
    ) THEN
        ALTER TABLE recipe_ingredients
            ADD CONSTRAINT recipe_ingredients_unit_not_blank CHECK (btrim(unit) <> '');
    END IF;
END;
$$;

DELETE FROM schema_migrations WHERE version = 22;
