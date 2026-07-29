-- Implements DESIGN-005 MicronutrientVocabulary administrator lookup.
SELECT key, display_name, unit, active
FROM micronutrient_vocabulary
WHERE key = $1
FOR UPDATE;
