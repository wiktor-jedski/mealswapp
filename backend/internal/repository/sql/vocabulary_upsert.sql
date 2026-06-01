-- Implements DESIGN-005 MicronutrientVocabulary upsert query.
INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    unit = EXCLUDED.unit,
    active = EXCLUDED.active,
    updated_at = now();
