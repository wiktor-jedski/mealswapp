-- Implements DESIGN-005 MicronutrientVocabulary migration rollback.
DELETE FROM schema_migrations WHERE version = 30;

ALTER TABLE micronutrient_vocabulary
    DROP CONSTRAINT IF EXISTS micronutrient_vocabulary_unit_supported,
    DROP CONSTRAINT IF EXISTS micronutrient_vocabulary_display_normalized,
    DROP CONSTRAINT IF EXISTS micronutrient_vocabulary_key_canonical;
