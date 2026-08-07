-- Implements DESIGN-008 AccountDeleter payload-free create retry tombstones.
WITH keys AS (
    SELECT key
    FROM mutation_idempotency_keys
    WHERE user_id = $1 AND method = 'POST' AND route = '/custom-items'
      AND response_body->>'id' = $2::text
), marked AS (
    INSERT INTO deleted_custom_food_create_keys (user_id, key, expires_at)
    SELECT $1, key, now() + interval '24 hours' FROM keys
    ON CONFLICT (user_id, key) DO UPDATE SET expires_at = EXCLUDED.expires_at
)
DELETE FROM mutation_idempotency_keys
WHERE user_id = $1 AND method = 'POST' AND route = '/custom-items'
  AND response_body->>'id' = $2::text;
