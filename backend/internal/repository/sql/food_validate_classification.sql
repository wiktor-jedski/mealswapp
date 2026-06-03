-- Implements DESIGN-005 FoodItemEntity classification validation query.
SELECT EXISTS (
    SELECT 1
    FROM classifications
    WHERE id = $1 AND kind = $2 AND deleted_at IS NULL
);
