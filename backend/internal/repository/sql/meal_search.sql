-- Implements DESIGN-005 MealEntity search query.
SELECT m.id
FROM meals m
WHERE ($1::boolean OR m.deleted_at IS NULL)
  AND (
      $2::text = ''
      OR lower(btrim(m.name)) LIKE '%' || lower(btrim($2)) || '%'
  )
  AND ($3::integer IS NULL OR m.prep_time_minutes <= $3)
  AND (
      coalesce(cardinality($4::uuid[]), 0) = 0
      OR m.id IN (
          SELECT mt.meal_id
          FROM meal_classifications mt
          JOIN classifications t ON t.id = mt.classification_id
          WHERE t.kind = 'food_category' AND t.id = ANY($4::uuid[])
      )
  )
  AND (
      coalesce(cardinality($5::uuid[]), 0) = 0
      OR m.id IN (
          SELECT mt.meal_id
          FROM meal_classifications mt
          JOIN classifications t ON t.id = mt.classification_id
          WHERE t.kind = 'culinary_role' AND t.id = ANY($5::uuid[])
      )
  )
	AND (coalesce(cardinality($6::text[]), 0) = 0 OR m.physical_state = ANY($6::text[]))
ORDER BY lower(btrim(m.name)), m.id
LIMIT $7 OFFSET $8;
