-- Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
CREATE TABLE IF NOT EXISTS stripe_dead_letters (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id text NOT NULL,
    event_type text NOT NULL,
    failure_category text NOT NULL,
    error_message text NOT NULL DEFAULT '',
    payload_sha256 text NOT NULL,
    stripe_customer_id text NOT NULL DEFAULT '',
    stripe_subscription_id text NOT NULL DEFAULT '',
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT stripe_dead_letters_event_id_not_blank CHECK (btrim(event_id) <> ''),
    CONSTRAINT stripe_dead_letters_event_type_not_blank CHECK (btrim(event_type) <> ''),
    CONSTRAINT stripe_dead_letters_failure_not_blank CHECK (btrim(failure_category) <> ''),
    CONSTRAINT stripe_dead_letters_payload_hash_hex CHECK (payload_sha256 ~ '^[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS stripe_dead_letters_created_idx
    ON stripe_dead_letters (created_at DESC);

CREATE INDEX IF NOT EXISTS stripe_dead_letters_event_idx
    ON stripe_dead_letters (event_id);

INSERT INTO schema_migrations (version)
VALUES (18)
ON CONFLICT (version) DO NOTHING;
