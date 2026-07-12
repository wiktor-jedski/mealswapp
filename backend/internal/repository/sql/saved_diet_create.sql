-- Implements DESIGN-008 SavedDataRepository.
INSERT INTO saved_diets (user_id, name)
VALUES ($1, $2)
RETURNING id;
