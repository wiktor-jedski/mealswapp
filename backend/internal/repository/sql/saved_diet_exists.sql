-- Implements DESIGN-008 SavedDataRepository ownership-aware daily-diet deletion.
SELECT EXISTS (SELECT 1 FROM saved_diets WHERE id = $1);
