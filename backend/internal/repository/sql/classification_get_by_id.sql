-- Implements DESIGN-009 TagManager classification lookup query.
SELECT id, name, kind, parent_id
FROM classifications
WHERE id = $1 AND deleted_at IS NULL;
