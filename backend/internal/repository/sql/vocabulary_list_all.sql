-- Implements DESIGN-005 MicronutrientVocabulary deterministic administrator listing.
SELECT key, display_name, unit, active
FROM micronutrient_vocabulary
ORDER BY lower(key), key
LIMIT 1000;
