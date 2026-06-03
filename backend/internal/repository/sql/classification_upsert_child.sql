-- Implements DESIGN-005 ClassificationEntity child-classification upsert query.
INSERT INTO classifications (name, kind, parent_id)
VALUES ($1, $2, $3)
ON CONFLICT (kind, parent_id, normalized_name) WHERE parent_id IS NOT NULL AND deleted_at IS NULL
DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING id;
