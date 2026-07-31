-- Implements DESIGN-005 MicronutrientVocabulary in-use integration fixture.
UPDATE food_items
SET micronutrients = jsonb_build_object($1::text, 1)
WHERE id = '10000000-0000-0000-0000-000000000201';
