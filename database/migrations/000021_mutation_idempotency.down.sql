-- Implements DESIGN-007 SubscriptionController and DESIGN-008 SavedDataRepository shared mutation idempotency storage rollback.
DO $$
BEGIN
    IF to_regclass('public.mutation_idempotency_keys') IS NOT NULL THEN
        IF to_regclass('public.checkout_idempotency_keys') IS NULL THEN
            ALTER TABLE mutation_idempotency_keys RENAME TO checkout_idempotency_keys;
        ELSE
            INSERT INTO checkout_idempotency_keys (id, user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at)
            SELECT id, user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at
            FROM mutation_idempotency_keys
            ON CONFLICT (user_id, method, route, key) DO NOTHING;
            DROP TABLE mutation_idempotency_keys;
        END IF;
    END IF;
END
$$;

DELETE FROM schema_migrations WHERE version = 21;
