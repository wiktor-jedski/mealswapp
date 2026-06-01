-- Implements DESIGN-009 AdminController curated-import upsert query.
INSERT INTO curated_imports (source_provider, external_id, food_item_id, status, conflict_reason, raw_payload)
VALUES (btrim($1), btrim($2), $3, $4, nullif(btrim($5), ''), $6)
ON CONFLICT (source_provider, external_id) DO UPDATE
SET food_item_id = EXCLUDED.food_item_id,
    status = EXCLUDED.status,
    conflict_reason = EXCLUDED.conflict_reason,
    raw_payload = EXCLUDED.raw_payload,
    updated_at = now()
RETURNING id;
