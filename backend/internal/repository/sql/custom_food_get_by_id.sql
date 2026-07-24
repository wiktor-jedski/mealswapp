-- Implements DESIGN-005 FoodItemEntity owner-scoped custom-item read query.
SELECT id, name, physical_state, prep_time_minutes, average_unit_weight_grams,
       average_serving_volume_milliliters, density_grams_per_milliliter,
       density_source_provider, density_source_food_id, density_source_kind,
       protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients, image_url,
       deleted_at, created_at, updated_at
FROM custom_food_items
WHERE owner_id = $1 AND id = $2 AND ($3::boolean OR deleted_at IS NULL);
