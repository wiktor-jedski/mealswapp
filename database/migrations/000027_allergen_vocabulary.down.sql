-- Implements DESIGN-009 TagManager allergen vocabulary rollback.
DROP TABLE IF EXISTS allergen_vocabulary;

DELETE FROM schema_migrations WHERE version = 27;
