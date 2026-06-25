-- Implements DESIGN-005 FoodItemEntity search query.
SELECT id, name, physical_state, prep_time_minutes, average_unit_weight_grams, average_serving_volume_milliliters,
       density_grams_per_milliliter, density_source_provider, density_source_food_id, density_source_kind,
       protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients, image_url,
       deleted_at, created_at, updated_at
FROM food_items
WHERE ($1::boolean OR deleted_at IS NULL)
  AND ($2::text = '' OR normalized_name LIKE '%' || lower(btrim($2)) || '%')
  AND ($3::integer IS NULL OR prep_time_minutes <= $3)
  AND (
      coalesce(cardinality($4::uuid[]), 0) = 0
      OR id IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          JOIN classifications t ON t.id = fit.classification_id
          WHERE t.kind = 'food_category' AND t.id = ANY($4::uuid[])
      )
  )
  AND (
      coalesce(cardinality($5::uuid[]), 0) = 0
      OR id NOT IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          JOIN classifications t ON t.id = fit.classification_id
          WHERE t.kind = 'food_category' AND t.id = ANY($5::uuid[])
      )
  )
  AND (
      coalesce(cardinality($6::uuid[]), 0) = 0
      OR id IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          JOIN classifications t ON t.id = fit.classification_id
          WHERE t.kind = 'culinary_role' AND t.id = ANY($6::uuid[])
      )
  )
  AND (
      coalesce(cardinality($7::uuid[]), 0) = 0
      OR id NOT IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          JOIN classifications t ON t.id = fit.classification_id
          WHERE t.kind = 'culinary_role' AND t.id = ANY($7::uuid[])
      )
  )
  AND (
      coalesce(cardinality($8::uuid[]), 0) = 0
      OR id IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          WHERE fit.classification_id = ANY($8::uuid[])
      )
  )
  AND (
      coalesce(cardinality($9::uuid[]), 0) = 0
      OR id NOT IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          WHERE fit.classification_id = ANY($9::uuid[])
      )
  )
  AND (
      coalesce(cardinality($10::text[]), 0) = 0
      OR id IN (
          SELECT fia.food_item_id
          FROM food_item_allergens fia
          WHERE fia.allergen_key = ANY($10::text[])
      )
  )
  AND (
      coalesce(cardinality($11::text[]), 0) = 0
      OR id NOT IN (
          SELECT fia.food_item_id
          FROM food_item_allergens fia
          WHERE fia.allergen_key = ANY($11::text[])
      )
  )
  AND (coalesce(cardinality($12::text[]), 0) = 0 OR physical_state = ANY($12::text[]))
  AND (coalesce(cardinality($13::text[]), 0) = 0 OR physical_state <> ALL($13::text[]))
ORDER BY normalized_name, id
LIMIT $14 OFFSET $15;
