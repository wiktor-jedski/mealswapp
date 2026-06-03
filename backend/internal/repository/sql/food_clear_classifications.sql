-- Implements DESIGN-005 FoodItemEntity clear-classifications query.
DELETE FROM food_item_classifications WHERE food_item_id = $1;
