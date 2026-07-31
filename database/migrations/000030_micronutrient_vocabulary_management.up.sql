-- Implements DESIGN-005 MicronutrientVocabulary canonical administrator validation.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'micronutrient_vocabulary'::regclass AND conname = 'micronutrient_vocabulary_key_canonical') THEN
        ALTER TABLE micronutrient_vocabulary ADD CONSTRAINT micronutrient_vocabulary_key_canonical
            CHECK (key ~ '^[A-Z][A-Za-z0-9]{2,119}$');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'micronutrient_vocabulary'::regclass AND conname = 'micronutrient_vocabulary_display_normalized') THEN
        ALTER TABLE micronutrient_vocabulary ADD CONSTRAINT micronutrient_vocabulary_display_normalized
            CHECK (display_name = btrim(display_name) AND char_length(display_name) <= 120 AND display_name !~ '[[:cntrl:]]');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'micronutrient_vocabulary'::regclass AND conname = 'micronutrient_vocabulary_unit_supported') THEN
        ALTER TABLE micronutrient_vocabulary ADD CONSTRAINT micronutrient_vocabulary_unit_supported
            CHECK (unit IN ('g', 'mg', 'mcg'));
    END IF;
END
$$;

INSERT INTO schema_migrations (version)
VALUES (30)
ON CONFLICT (version) DO NOTHING;
