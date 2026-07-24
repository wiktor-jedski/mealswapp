-- Implements DESIGN-005 FoodItemEntity custom-item classification assignment query.
INSERT INTO custom_food_item_classifications (custom_food_item_id, classification_id)
VALUES ($1, $2);
