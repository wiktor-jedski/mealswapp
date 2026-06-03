-- Implements DESIGN-005 FoodItemEntity attach-classification query.
INSERT INTO food_item_classifications (food_item_id, classification_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
