-- Implements DESIGN-005 MicronutrientVocabulary display-name update with immutable key.
UPDATE micronutrient_vocabulary
SET display_name = $2, updated_at = now()
WHERE key = $1
RETURNING key, display_name, unit, active;
