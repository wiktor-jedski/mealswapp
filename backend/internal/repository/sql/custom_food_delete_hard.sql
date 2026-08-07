-- Implements DESIGN-008 AccountDeleter permanent custom-item deletion.
DELETE FROM custom_food_items WHERE owner_id = $1 AND id = $2;
