-- Implements DESIGN-008 SavedDataRepository.
UPDATE saved_diets
SET name = $3, updated_at = now()
WHERE id = $1 AND user_id = $2;
