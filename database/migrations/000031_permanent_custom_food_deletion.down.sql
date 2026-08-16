-- Implements DESIGN-008 AccountDeleter permanent custom-item deletion rollback.
DROP TABLE IF EXISTS deleted_custom_food_create_keys;
DELETE FROM schema_migrations WHERE version = 31;
