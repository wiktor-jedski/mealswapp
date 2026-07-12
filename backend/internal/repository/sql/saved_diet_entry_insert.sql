-- Implements DESIGN-008 SavedDataRepository.
INSERT INTO saved_diet_meal_entries (saved_diet_id, meal_id, quantity, unit, position)
VALUES ($1, $2, $3, $4, $5);
