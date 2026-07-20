-- Implements DESIGN-008 SavedDataRepository immutable daily-diet entry snapshot.
INSERT INTO saved_diet_meal_entries (id, saved_diet_id, meal_id, quantity, unit, position, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);
