-- Implements DESIGN-009 ItemCurator active ownerless global-item discovery count.
SELECT count(*)
FROM food_items
WHERE deleted_at IS NULL
  AND strpos(normalized_name, $1::text) > 0;
