-- Implements DESIGN-008 SavedDataRepository.
INSERT INTO saved_diet_meal_entries (saved_diet_id, meal_id, food_item_id, quantity, unit, position)
VALUES (
    $1,
    CASE WHEN $3::text = 'meal' THEN $2::uuid ELSE NULL END,
    CASE WHEN $3::text = 'food_item' THEN $2::uuid ELSE NULL END,
    $4, $5, $6
);
