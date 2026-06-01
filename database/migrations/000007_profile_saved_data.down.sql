-- Implements DESIGN-008 SavedDataRepository.
-- Implements DESIGN-008 PreferenceManager.
DROP INDEX IF EXISTS search_history_user_created_idx;
DROP TABLE IF EXISTS search_history;
DROP INDEX IF EXISTS saved_items_user_kind_idx;
DROP TABLE IF EXISTS saved_items;
DROP TABLE IF EXISTS user_profiles;

DELETE FROM schema_migrations WHERE version = 7;
