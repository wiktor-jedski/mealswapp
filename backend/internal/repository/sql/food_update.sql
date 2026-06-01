-- Implements DESIGN-005 FoodItemEntity update query.
UPDATE food_items
SET name = $2,
    physical_state = $3,
    prep_time_minutes = $4,
    average_unit_weight_grams = $5,
    average_serving_volume_milliliters = $6,
    density_grams_per_milliliter = $7,
    density_source_provider = $8,
    density_source_food_id = $9,
    density_source_kind = $10,
    protein_per_100 = $11,
    carbohydrates_per_100 = $12,
    fat_per_100 = $13,
    micronutrients = $14,
    image_url = $15,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
