-- Implements DESIGN-008 AccountDeleter expired retry-marker purge.
DELETE FROM deleted_custom_food_create_keys WHERE expires_at <= now();
