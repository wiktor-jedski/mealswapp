CREATE TABLE micronutrient_vocabulary (
    key text PRIMARY KEY,
    display_name text NOT NULL,
    unit text NOT NULL CHECK (unit IN ('mg', 'mcg', 'IU')),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX micronutrient_vocabulary_active_idx ON micronutrient_vocabulary (active);

INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES
    ('Calcium', 'Calcium', 'mg', true),
    ('Iron', 'Iron', 'mg', true),
    ('Potassium', 'Potassium', 'mg', true),
    ('Sodium', 'Sodium', 'mg', true),
    ('VitaminA', 'Vitamin A', 'mcg', true),
    ('VitaminC', 'Vitamin C', 'mg', true),
    ('VitaminD', 'Vitamin D', 'IU', true)
ON CONFLICT (key) DO UPDATE
SET display_name = excluded.display_name,
    unit = excluded.unit,
    active = excluded.active,
    updated_at = now();
