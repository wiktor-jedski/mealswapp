-- Implements DESIGN-005 MicronutrientVocabulary active vocabulary query.
SELECT key, display_name, unit, active
FROM micronutrient_vocabulary
WHERE active
ORDER BY key;
