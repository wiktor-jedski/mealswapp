-- Implements DESIGN-008 SavedDataRepository.
SELECT id, saved_diet_id, meal_id, quantity, unit, position, created_at
FROM saved_diet_meal_entries
WHERE saved_diet_id = $1
ORDER BY position, id;
