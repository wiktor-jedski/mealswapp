-- Implements DESIGN-008 SavedDataRepository.
SELECT id, user_id, name, created_at, updated_at
FROM saved_diets
WHERE id = $1 AND user_id = $2;
