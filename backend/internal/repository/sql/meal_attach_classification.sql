-- Implements DESIGN-005 MealEntity attach-classification query.
INSERT INTO meal_classifications (meal_id, classification_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
