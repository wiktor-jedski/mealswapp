-- Implements DESIGN-009 ItemCurator active ownerless global-item discovery.
SELECT id, name, physical_state, prep_time_minutes, average_unit_weight_grams, average_serving_volume_milliliters,
       density_grams_per_milliliter, density_source_provider, density_source_food_id, density_source_kind,
       protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients, image_url,
       deleted_at, created_at, updated_at
FROM food_items
WHERE deleted_at IS NULL
  AND strpos(normalized_name, $1::text) > 0
ORDER BY normalized_name, id
LIMIT $2 OFFSET $3;
