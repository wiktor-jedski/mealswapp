-- Implements DESIGN-006 AuthUser.
CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    role text NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    email_verified boolean NOT NULL DEFAULT false,
    password_hash text,
    password_salt text,
    failed_login_count integer NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
    locked_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_password_pair CHECK (
        (password_hash IS NULL AND password_salt IS NULL)
        OR (password_hash IS NOT NULL AND password_salt IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS oauth_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('google', 'apple')),
    provider_user_id text NOT NULL,
    email text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_identities_provider_user_not_blank CHECK (btrim(provider_user_id) <> ''),
    CONSTRAINT oauth_identities_email_not_blank CHECK (btrim(email) <> ''),
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS oauth_identities_user_idx
    ON oauth_identities (user_id);

CREATE TABLE IF NOT EXISTS user_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash text NOT NULL,
    refresh_family_id uuid NOT NULL,
    access_expires_at timestamptz NOT NULL,
    refresh_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_sessions_refresh_hash_not_blank CHECK (btrim(refresh_token_hash) <> ''),
    CONSTRAINT user_sessions_expiry_order CHECK (refresh_expires_at > access_expires_at)
);

CREATE INDEX IF NOT EXISTS user_sessions_user_idx
    ON user_sessions (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS user_sessions_refresh_hash_idx
    ON user_sessions (refresh_token_hash);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token_hash text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT password_reset_tokens_hash_not_blank CHECK (btrim(token_hash) <> ''),
    CONSTRAINT password_reset_tokens_used_before_expiry CHECK (used_at IS NULL OR used_at <= expires_at)
);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_idx
    ON password_reset_tokens (user_id);

CREATE INDEX IF NOT EXISTS password_reset_tokens_expiry_idx
    ON password_reset_tokens (expires_at);

INSERT INTO schema_migrations (version)
VALUES (6)
ON CONFLICT (version) DO NOTHING;
