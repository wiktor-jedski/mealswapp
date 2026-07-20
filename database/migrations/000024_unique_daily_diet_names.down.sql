-- Implements DESIGN-008 SavedDataRepository user-scoped Daily Diet naming rollback.
DROP INDEX IF EXISTS saved_diets_user_normalized_name_uidx;

DELETE FROM schema_migrations WHERE version = 24;
