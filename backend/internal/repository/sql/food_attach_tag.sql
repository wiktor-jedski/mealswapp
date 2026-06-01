-- Implements DESIGN-005 FoodItemEntity attach-tag query.
INSERT INTO food_item_tags (food_item_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
