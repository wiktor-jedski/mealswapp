-- Implements DESIGN-008 AccountDeleter delayed-create replay prevention.
SELECT 1 FROM deleted_custom_food_create_keys
WHERE user_id = $1 AND key = $2 AND expires_at > now();
