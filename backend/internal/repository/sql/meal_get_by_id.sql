-- Implements DESIGN-005 MealEntity get-by-id query.
SELECT id, type, name, physical_state, prep_time_minutes, average_unit_weight_grams,
       protein_per_100, carbohydrates_per_100, fat_per_100, created_at, updated_at
FROM meals
WHERE id = $1 AND ($2::boolean OR deleted_at IS NULL);
