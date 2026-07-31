-- Implements DESIGN-008 AccountDeleter permanent custom-item deletion.
CREATE TABLE IF NOT EXISTS deleted_custom_food_create_keys (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, key)
);
CREATE TEMP TABLE task298_legacy_custom_food_items ON COMMIT DROP AS
SELECT id, owner_id FROM custom_food_items WHERE deleted_at IS NOT NULL;
-- Preserve only the owner/key/expiry needed to reject a delayed legacy replay.
INSERT INTO deleted_custom_food_create_keys (user_id, key, expires_at)
SELECT m.user_id, m.key, now() + interval '24 hours'
FROM mutation_idempotency_keys m
JOIN task298_legacy_custom_food_items legacy
  ON legacy.id::text = m.response_body->>'id'
 AND legacy.owner_id = m.user_id
WHERE m.method = 'POST' AND m.route = '/custom-items'
ON CONFLICT (user_id, key) DO UPDATE SET expires_at = EXCLUDED.expires_at;
-- Completed create claims for purged legacy items cannot remain replayable.
DELETE FROM mutation_idempotency_keys m
USING task298_legacy_custom_food_items legacy
WHERE m.method = 'POST' AND m.route = '/custom-items'
  AND m.response_body->>'id' = legacy.id::text
  AND m.user_id = legacy.owner_id;
-- Legacy soft-deleted items cannot remain valid Daily Diet Food Objects. Remove only
-- their owner-scoped entries first so the hard purge is FK-safe and atomic.
DELETE FROM saved_diet_meal_entries e
USING custom_food_items i
JOIN task298_legacy_custom_food_items legacy ON legacy.id = i.id
WHERE e.custom_food_item_id = i.id;
-- Legacy soft-deleted private items are no longer recoverable after this migration.
DELETE FROM custom_food_items i
USING task298_legacy_custom_food_items legacy
WHERE i.id = legacy.id
  AND NOT EXISTS (SELECT 1 FROM saved_diet_meal_entries e WHERE e.custom_food_item_id = i.id);
CREATE INDEX IF NOT EXISTS deleted_custom_food_create_keys_expiry_idx
    ON deleted_custom_food_create_keys (expires_at);
INSERT INTO schema_migrations (version) VALUES (31) ON CONFLICT (version) DO NOTHING;
