-- Implements DESIGN-008 SavedDataRepository.
SELECT id, user_id, name, created_at, updated_at
FROM saved_diets
WHERE user_id = $1
ORDER BY created_at DESC, id;
