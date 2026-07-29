-- Implements DESIGN-009 ItemCurator active ownerless global-item discovery count.
SELECT count(*)
FROM food_items
WHERE deleted_at IS NULL
  AND strpos(regexp_replace(normalized_name, '[[:space:]]+', ' ', 'g'), $1::text) > 0;
