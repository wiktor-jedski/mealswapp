-- Implements DESIGN-005 ClassificationEntity and DESIGN-009 TagManager deterministic hierarchy query.
WITH RECURSIVE hierarchy AS (
    SELECT id, name, kind, parent_id, ARRAY[normalized_name, id::text] AS sort_path
    FROM classifications
    WHERE kind = $1 AND parent_id IS NULL AND deleted_at IS NULL
    UNION ALL
    SELECT child.id, child.name, child.kind, child.parent_id,
           hierarchy.sort_path || ARRAY[child.normalized_name, child.id::text]
    FROM classifications child
    JOIN hierarchy ON hierarchy.id = child.parent_id
    WHERE child.kind = $1 AND child.deleted_at IS NULL
)
SELECT id, name, kind, parent_id
FROM hierarchy
ORDER BY sort_path;
