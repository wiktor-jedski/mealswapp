-- Implements DESIGN-005 ClassificationEntity active classification query.
SELECT id, name, kind, parent_id
FROM classifications
WHERE kind = $1 AND deleted_at IS NULL
ORDER BY parent_id NULLS FIRST, normalized_name, id;
