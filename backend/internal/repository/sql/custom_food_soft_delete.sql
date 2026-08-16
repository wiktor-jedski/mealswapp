-- Implements DESIGN-005 FoodItemEntity and DESIGN-008 AccountDeleter write lockout.
WITH active_owner AS MATERIALIZED (
    SELECT id
    FROM users
    WHERE id = $1 AND deletion_requested_at IS NULL
    FOR NO KEY UPDATE
)
UPDATE custom_food_items
SET deleted_at = now(), updated_at = now()
FROM active_owner
WHERE owner_id = active_owner.id AND custom_food_items.id = $2 AND deleted_at IS NULL;
