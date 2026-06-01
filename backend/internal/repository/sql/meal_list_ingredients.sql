-- Implements DESIGN-005 MealEntity ingredient query.
SELECT food_item_id, quantity, unit, position
FROM recipe_ingredients
WHERE meal_id = $1
ORDER BY position, food_item_id;
