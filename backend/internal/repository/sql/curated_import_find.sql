-- Implements DESIGN-009 AdminController curated-import lookup query.
SELECT id, source_provider, external_id, food_item_id, status, coalesce(conflict_reason, ''), raw_payload, created_at, updated_at
FROM curated_imports
WHERE source_provider = btrim($1) AND external_id = btrim($2);
