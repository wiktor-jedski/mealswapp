-- Implements DESIGN-005 MealEntity clear-classifications query.
DELETE FROM meal_classifications WHERE meal_id = $1;
