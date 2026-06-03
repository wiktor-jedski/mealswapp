-- Implements DESIGN-005 RepositoryInterfaces search count query.
SELECT count(*)
FROM food_items
WHERE ($1::boolean OR deleted_at IS NULL)
  AND ($2::text = '' OR normalized_name LIKE lower(btrim($2)) || '%')
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
      OR id IN (
          SELECT fit.food_item_id
          FROM food_item_classifications fit
          JOIN classifications t ON t.id = fit.classification_id
          WHERE t.kind = 'culinary_role' AND t.id = ANY($5::uuid[])
      )
  );
