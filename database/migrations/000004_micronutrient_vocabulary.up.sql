-- Implements DESIGN-005 MicronutrientVocabulary.
CREATE TABLE IF NOT EXISTS micronutrient_vocabulary (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    unit text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT micronutrient_vocabulary_key_not_blank CHECK (btrim(key) <> ''),
    CONSTRAINT micronutrient_vocabulary_display_not_blank CHECK (btrim(display_name) <> ''),
    CONSTRAINT micronutrient_vocabulary_unit_not_blank CHECK (btrim(unit) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS micronutrient_vocabulary_key_lower_idx
    ON micronutrient_vocabulary (lower(key));

CREATE INDEX IF NOT EXISTS micronutrient_vocabulary_active_idx
    ON micronutrient_vocabulary (active);

INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES
    ('Sodium', 'Sodium', 'mg', true),
    ('Potassium', 'Potassium', 'mg', true),
    ('Calcium', 'Calcium', 'mg', true),
    ('Iron', 'Iron', 'mg', true),
    ('VitaminC', 'Vitamin C', 'mg', true),
    ('VitaminD', 'Vitamin D', 'mcg', true),
    ('Fiber', 'Fiber', 'g', true),
    ('Sugar', 'Sugar', 'g', true)
ON CONFLICT (key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    unit = EXCLUDED.unit,
    active = EXCLUDED.active,
    updated_at = now();

INSERT INTO schema_migrations (version)
VALUES (4)
ON CONFLICT (version) DO NOTHING;
