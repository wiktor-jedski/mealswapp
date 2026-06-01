-- Implements DESIGN-005 MealEntity attach-tag query.
INSERT INTO meal_tags (meal_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
