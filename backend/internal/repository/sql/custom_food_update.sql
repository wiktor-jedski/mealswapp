-- Implements DESIGN-005 FoodItemEntity and DESIGN-008 AccountDeleter write lockout.
WITH active_owner AS MATERIALIZED (
    SELECT id
    FROM users
    WHERE id = $1 AND deletion_requested_at IS NULL
    FOR NO KEY UPDATE
)
UPDATE custom_food_items
SET name = $3,
    physical_state = $4,
    prep_time_minutes = $5,
    average_unit_weight_grams = $6,
    average_serving_volume_milliliters = $7,
    density_grams_per_milliliter = $8,
    density_source_provider = $9,
    density_source_food_id = $10,
    density_source_kind = $11,
    protein_per_100 = $12,
    carbohydrates_per_100 = $13,
    fat_per_100 = $14,
    micronutrients = $15,
    image_url = $16,
    updated_at = now()
FROM active_owner
WHERE owner_id = active_owner.id AND custom_food_items.id = $2 AND deleted_at IS NULL;
