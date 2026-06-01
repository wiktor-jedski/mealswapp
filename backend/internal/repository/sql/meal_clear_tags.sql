-- Implements DESIGN-005 MealEntity clear-tags query.
DELETE FROM meal_tags WHERE meal_id = $1;
