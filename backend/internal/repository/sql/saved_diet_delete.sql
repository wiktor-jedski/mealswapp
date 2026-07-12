-- Implements DESIGN-008 SavedDataRepository.
DELETE FROM saved_diets
WHERE id = $1 AND user_id = $2;
