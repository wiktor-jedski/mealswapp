-- Implements DESIGN-007 EntitlementManager.
-- Implements DESIGN-007 TrialTracker.
-- Implements DESIGN-007 UsageLimiter.
CREATE TABLE IF NOT EXISTS entitlements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier text NOT NULL CHECK (tier IN ('free', 'trial', 'paid')),
    status text NOT NULL CHECK (status IN ('active', 'expired', 'past_due', 'cancelled')),
    search_limit_per_24h integer NOT NULL CHECK (search_limit_per_24h >= 0),
    allowed_modes text[] NOT NULL,
    expires_at timestamptz,
    stripe_customer_id text,
    stripe_subscription_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entitlements_allowed_modes_not_empty CHECK (array_length(allowed_modes, 1) > 0),
    CONSTRAINT entitlements_allowed_modes_valid CHECK (allowed_modes <@ ARRAY['catalog', 'substitution', 'daily_diet', 'daily_diet_alternative']::text[]),
    CONSTRAINT entitlements_trial_expiry_required CHECK (tier <> 'trial' OR expires_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS entitlements_user_created_idx
    ON entitlements (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS entitlements_trial_expiry_idx
    ON entitlements (expires_at)
    WHERE tier = 'trial' AND status = 'active';

CREATE TABLE IF NOT EXISTS usage_windows (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature text NOT NULL,
    started_at timestamptz NOT NULL,
    search_count integer NOT NULL DEFAULT 0 CHECK (search_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT usage_windows_feature_not_blank CHECK (btrim(feature) <> ''),
    UNIQUE (user_id, feature, started_at)
);

CREATE INDEX IF NOT EXISTS usage_windows_user_started_idx
    ON usage_windows (user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS processed_stripe_events (
    event_id text PRIMARY KEY,
    event_type text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('success', 'duplicate', 'failed')),
    processed_at timestamptz NOT NULL DEFAULT now(),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT processed_stripe_events_id_not_blank CHECK (btrim(event_id) <> ''),
    CONSTRAINT processed_stripe_events_type_not_blank CHECK (btrim(event_type) <> ''),
    CONSTRAINT processed_stripe_events_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

INSERT INTO schema_migrations (version)
VALUES (8)
ON CONFLICT (version) DO NOTHING;
