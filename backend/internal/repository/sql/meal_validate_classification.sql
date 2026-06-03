-- Implements DESIGN-005 MealEntity classification validation query.
SELECT EXISTS (
    SELECT 1
    FROM classifications
    WHERE id = $1 AND deleted_at IS NULL
);
