-- Implements DESIGN-007 SubscriptionController and DESIGN-008 SavedDataRepository shared mutation idempotency storage.
DO $$
BEGIN
    IF to_regclass('public.checkout_idempotency_keys') IS NOT NULL
       AND to_regclass('public.mutation_idempotency_keys') IS NULL THEN
        ALTER TABLE checkout_idempotency_keys RENAME TO mutation_idempotency_keys;
    END IF;
END
$$;
INSERT INTO schema_migrations (version) VALUES (21) ON CONFLICT (version) DO NOTHING;
