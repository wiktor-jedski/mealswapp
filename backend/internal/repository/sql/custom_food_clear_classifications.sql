-- Implements DESIGN-005 FoodItemEntity custom-item classification replacement query.
DELETE FROM custom_food_item_classifications WHERE custom_food_item_id = $1;
