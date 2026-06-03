-- Implements DESIGN-005 ClassificationEntity soft-delete query.
UPDATE classifications
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
