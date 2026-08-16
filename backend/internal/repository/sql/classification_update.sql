-- Implements DESIGN-009 TagManager classification update query.
UPDATE classifications
SET name = $2, parent_id = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, name, kind, parent_id;
