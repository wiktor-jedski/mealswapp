-- Implements DESIGN-005 MealEntity clear-ingredients query.
DELETE FROM recipe_ingredients WHERE meal_id = $1;
