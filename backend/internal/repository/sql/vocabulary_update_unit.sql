-- Implements DESIGN-005 MicronutrientVocabulary safe unit update.
UPDATE micronutrient_vocabulary
SET unit = $2, updated_at = now()
WHERE key = $1
RETURNING key, display_name, unit, active;
