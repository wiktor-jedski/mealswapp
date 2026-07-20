-- Implements DESIGN-008 SavedDataRepository.
SELECT EXISTS (
    SELECT 1
    FROM saved_diets
    WHERE id = $1 AND user_id = $2
);
