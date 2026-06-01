-- Implements DESIGN-005 FoodItemEntity clear-tags query.
DELETE FROM food_item_tags WHERE food_item_id = $1;
