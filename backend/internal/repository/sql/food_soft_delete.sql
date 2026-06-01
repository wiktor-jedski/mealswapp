-- Implements DESIGN-005 FoodItemEntity soft-delete query.
UPDATE food_items
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
