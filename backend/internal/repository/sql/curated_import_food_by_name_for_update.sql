-- Implements DESIGN-009 DataImporter explicit normalized-name conflict locking.
SELECT id
FROM food_items
WHERE normalized_name = lower(btrim($1)) AND deleted_at IS NULL
FOR UPDATE;
