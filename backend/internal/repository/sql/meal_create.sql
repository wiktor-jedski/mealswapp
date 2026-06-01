-- Implements DESIGN-005 MealEntity create query.
INSERT INTO meals (
    type, name, physical_state, prep_time_minutes, average_unit_weight_grams,
    protein_per_100, carbohydrates_per_100, fat_per_100
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;
