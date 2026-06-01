-- Implements DESIGN-005 TagEntity active tag query.
SELECT id, name, kind, parent_id
FROM tags
WHERE kind = $1 AND deleted_at IS NULL
ORDER BY parent_id NULLS FIRST, normalized_name, id;
