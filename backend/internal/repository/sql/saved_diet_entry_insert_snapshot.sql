-- Implements DESIGN-008 SavedDataRepository immutable daily-diet entry snapshot.
INSERT INTO saved_diet_meal_entries (id, saved_diet_id, meal_id, food_item_id, quantity, unit, position, created_at)
VALUES (
    $1, $2,
    CASE WHEN $4::text = 'meal' THEN $3::uuid ELSE NULL END,
    CASE WHEN $4::text = 'food_item' THEN $3::uuid ELSE NULL END,
    $5, $6, $7, $8
);
