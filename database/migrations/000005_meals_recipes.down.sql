-- Implements DESIGN-005 MealEntity.
-- Implements DESIGN-005 RecipeEntity.
ALTER TABLE meal_classifications
    DROP CONSTRAINT IF EXISTS meal_classifications_meal_id_fkey;

ALTER TABLE IF EXISTS meal_tags
    DROP CONSTRAINT IF EXISTS meal_tags_meal_id_fkey;

DROP TRIGGER IF EXISTS recipe_ingredients_recipe_required ON recipe_ingredients;
DROP TRIGGER IF EXISTS meals_recipe_ingredient_required ON meals;
DROP FUNCTION IF EXISTS ensure_recipe_meal_has_ingredients();
DROP INDEX IF EXISTS recipe_ingredients_food_item_idx;
DROP TABLE IF EXISTS recipe_ingredients;
DROP INDEX IF EXISTS meals_type_idx;
DROP TABLE IF EXISTS meals;

DELETE FROM schema_migrations WHERE version = 5;
