-- Implements DESIGN-007 SubscriptionController checkout idempotency.
CREATE TABLE IF NOT EXISTS checkout_idempotency_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method text NOT NULL,
    route text NOT NULL,
    key text NOT NULL,
    body_hash text NOT NULL,
    status_code integer NOT NULL CHECK (status_code BETWEEN 100 AND 599),
    response_body jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT checkout_idempotency_method_not_blank CHECK (btrim(method) <> ''),
    CONSTRAINT checkout_idempotency_route_not_blank CHECK (btrim(route) <> ''),
    CONSTRAINT checkout_idempotency_key_not_blank CHECK (btrim(key) <> ''),
    CONSTRAINT checkout_idempotency_body_hash_not_blank CHECK (btrim(body_hash) <> ''),
    UNIQUE (user_id, method, route, key)
);

CREATE INDEX IF NOT EXISTS checkout_idempotency_user_created_idx
    ON checkout_idempotency_keys (user_id, created_at DESC);

INSERT INTO schema_migrations (version)
VALUES (17)
ON CONFLICT (version) DO NOTHING;
