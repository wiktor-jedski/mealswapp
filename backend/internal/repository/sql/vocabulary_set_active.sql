-- Implements DESIGN-005 MicronutrientVocabulary deactivate/reactivate lifecycle.
UPDATE micronutrient_vocabulary
SET active = $2, updated_at = now()
WHERE key = $1
RETURNING key, display_name, unit, active;
