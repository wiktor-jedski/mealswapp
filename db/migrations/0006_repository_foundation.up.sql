CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    normalized_email text GENERATED ALWAYS AS (lower(trim(email))) STORED,
    display_name text NOT NULL DEFAULT '',
    password_hash text NOT NULL DEFAULT '',
    role text NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    disabled boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_normalized_email_unique UNIQUE (normalized_email)
);

CREATE TABLE user_preferences (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme text NOT NULL DEFAULT 'system' CHECK (theme IN ('system', 'light', 'dark')),
    default_search_mode text NOT NULL DEFAULT 'single' CHECK (default_search_mode IN ('single', 'replacement', 'diet')),
    enabled_macros jsonb NOT NULL DEFAULT '{"protein": true, "carbs": true, "fat": true}'::jsonb,
    excluded_tag_ids uuid[] NOT NULL DEFAULT '{}',
    dietary_filter_ids uuid[] NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_preferences_enabled_macros_object CHECK (jsonb_typeof(enabled_macros) = 'object')
);

CREATE TABLE entitlements (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan text NOT NULL CHECK (plan IN ('free', 'trial', 'paid')),
    status text NOT NULL CHECK (status IN ('active', 'expired', 'canceled')),
    expires_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE saved_data (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('favorite', 'saved_search', 'search_history')),
    label text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT saved_data_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action text NOT NULL,
    target text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_logs_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE TABLE import_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('draft', 'imported', 'rejected', 'failed')),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT import_records_provider_external_unique UNIQUE (provider, external_id),
    CONSTRAINT import_records_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX saved_data_user_kind_idx ON saved_data (user_id, kind);
CREATE INDEX audit_logs_actor_id_idx ON audit_logs (actor_id);
CREATE INDEX audit_logs_action_idx ON audit_logs (action);
CREATE INDEX import_records_status_idx ON import_records (status);
