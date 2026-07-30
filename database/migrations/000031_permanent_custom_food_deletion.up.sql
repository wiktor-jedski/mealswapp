-- Implements DESIGN-008 AccountDeleter permanent custom-item deletion.
CREATE TABLE IF NOT EXISTS deleted_custom_food_create_keys (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, key)
);
-- Legacy soft-deleted items cannot remain valid Daily Diet Food Objects. Remove only
-- their owner-scoped entries first so the hard purge is FK-safe and atomic.
DELETE FROM saved_diet_meal_entries e
USING custom_food_items i
WHERE e.custom_food_item_id = i.id AND i.deleted_at IS NOT NULL;
-- Legacy soft-deleted private items are no longer recoverable after this migration.
DELETE FROM custom_food_items i
WHERE i.deleted_at IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM saved_diet_meal_entries e WHERE e.custom_food_item_id = i.id);
CREATE INDEX IF NOT EXISTS deleted_custom_food_create_keys_expiry_idx
    ON deleted_custom_food_create_keys (expires_at);
INSERT INTO schema_migrations (version) VALUES (31) ON CONFLICT (version) DO NOTHING;
