-- Implements DESIGN-005 MicronutrientVocabulary.
DROP INDEX IF EXISTS micronutrient_vocabulary_active_idx;
DROP INDEX IF EXISTS micronutrient_vocabulary_key_lower_idx;
DROP TABLE IF EXISTS micronutrient_vocabulary;

DELETE FROM schema_migrations WHERE version = 4;
