-- Implements DESIGN-005 FoodItemEntity create query.
INSERT INTO food_items (
    name, physical_state, prep_time_minutes, average_unit_weight_grams, average_serving_volume_milliliters,
    density_grams_per_milliliter, density_source_provider, density_source_food_id, density_source_kind,
    protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients, image_url
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id;
