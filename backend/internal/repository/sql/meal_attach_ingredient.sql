-- Implements DESIGN-005 MealEntity attach-ingredient query.
INSERT INTO recipe_ingredients (meal_id, food_item_id, quantity, unit, position)
VALUES ($1, $2, $3, $4, $5);
