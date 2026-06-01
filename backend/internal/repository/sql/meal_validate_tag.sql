-- Implements DESIGN-005 MealEntity tag validation query.
SELECT EXISTS (
    SELECT 1
    FROM tags
    WHERE id = $1 AND deleted_at IS NULL
);
