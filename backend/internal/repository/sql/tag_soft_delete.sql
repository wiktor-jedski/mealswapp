-- Implements DESIGN-005 TagEntity soft-delete query.
UPDATE tags
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
