-- Implements DESIGN-005 FoodItemEntity tag validation query.
SELECT EXISTS (
    SELECT 1
    FROM tags
    WHERE id = $1 AND kind = $2 AND deleted_at IS NULL
);
