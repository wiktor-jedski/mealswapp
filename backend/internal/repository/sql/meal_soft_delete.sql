-- Implements DESIGN-005 MealEntity soft-delete query.
UPDATE meals
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
