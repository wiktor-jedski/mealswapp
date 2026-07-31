-- Implements DESIGN-005 MicronutrientVocabulary global-use fixture cleanup.
UPDATE food_items
SET micronutrients = '{}'::jsonb
WHERE id = '10000000-0000-0000-0000-000000000201';
