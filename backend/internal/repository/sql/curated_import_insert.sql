-- Implements DESIGN-009 DataImporter immutable confirmation insert.
INSERT INTO curated_imports (id, source_provider, external_id, food_item_id, status, raw_payload)
VALUES ($1, btrim($2), btrim($3), $4, 'imported', $5)
RETURNING id;
