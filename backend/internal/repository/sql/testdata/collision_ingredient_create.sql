-- Implements DESIGN-005 RecipeEntity integration fixture.
INSERT INTO recipe_ingredients (meal_id, food_item_id, quantity, unit, position)
VALUES ($1, $1, 1, 'g', 0);
