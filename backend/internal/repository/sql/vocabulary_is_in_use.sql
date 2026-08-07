-- Implements DESIGN-005 MicronutrientVocabulary global/private item usage guard.
SELECT EXISTS (
    SELECT 1 FROM food_items WHERE micronutrients ? $1
    UNION ALL
    SELECT 1 FROM custom_food_items WHERE micronutrients ? $1
);
