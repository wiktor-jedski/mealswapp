-- Implements DESIGN-005 MicronutrientVocabulary allowed-key query.
SELECT EXISTS (
    SELECT 1
    FROM micronutrient_vocabulary
    WHERE key = $1 AND active
);
