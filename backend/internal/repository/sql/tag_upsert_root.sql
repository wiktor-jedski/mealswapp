-- Implements DESIGN-005 TagEntity root-tag upsert query.
INSERT INTO tags (name, kind, parent_id)
VALUES ($1, $2, $3)
ON CONFLICT (kind, normalized_name) WHERE parent_id IS NULL AND deleted_at IS NULL
DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING id;
