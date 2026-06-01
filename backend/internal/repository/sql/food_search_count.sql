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
          FROM food_item_tags fit
          JOIN tags t ON t.id = fit.tag_id
          WHERE t.kind = 'category' AND t.id = ANY($4::uuid[])
      )
  )
  AND (
      coalesce(cardinality($5::uuid[]), 0) = 0
      OR id IN (
          SELECT fit.food_item_id
          FROM food_item_tags fit
          JOIN tags t ON t.id = fit.tag_id
          WHERE t.kind = 'functionality' AND t.id = ANY($5::uuid[])
      )
  );
