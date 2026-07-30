-- Implements DESIGN-008 AccountDeleter serialized permanent custom-item deletion.
SELECT id FROM custom_food_items WHERE owner_id = $1 AND id = $2 FOR UPDATE;
