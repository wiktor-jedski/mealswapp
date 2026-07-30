-- Implements DESIGN-008 AccountDeleter bounded saved-diet deletion conflict projection.
SELECT d.id, d.name FROM saved_diet_meal_entries e JOIN saved_diets d ON d.id=e.saved_diet_id WHERE e.custom_food_item_id=$1 AND d.user_id=$2 ORDER BY d.id LIMIT 25;
