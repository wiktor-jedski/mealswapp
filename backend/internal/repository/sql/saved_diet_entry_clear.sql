-- Implements DESIGN-008 SavedDataRepository.
DELETE FROM saved_diet_meal_entries
WHERE saved_diet_id = $1;
