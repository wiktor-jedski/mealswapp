-- Implements DESIGN-008 SavedDataRepository atomic daily-diet creation.
INSERT INTO saved_diets (id, user_id, name)
VALUES ($1, $2, $3)
RETURNING id;
