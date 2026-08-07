-- Implements DESIGN-005 MicronutrientVocabulary retry-safe creation.
INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES ($1, $2, $3, true)
ON CONFLICT (key) DO UPDATE SET key = EXCLUDED.key
WHERE micronutrient_vocabulary.display_name = EXCLUDED.display_name
  AND micronutrient_vocabulary.unit = EXCLUDED.unit
  AND micronutrient_vocabulary.active
RETURNING key, display_name, unit, active;
