-- Implements DESIGN-005 MealEntity update query.
UPDATE meals
SET type = $2,
    name = $3,
    physical_state = $4,
    prep_time_minutes = $5,
    average_unit_weight_grams = $6,
    protein_per_100 = $7,
    carbohydrates_per_100 = $8,
    fat_per_100 = $9,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
