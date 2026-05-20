CREATE TABLE oauth_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    provider_user_id text NOT NULL,
    email text NOT NULL DEFAULT '',
    display_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_identities_provider_user_unique UNIQUE (provider, provider_user_id)
);

CREATE INDEX oauth_identities_user_id_idx ON oauth_identities (user_id);
CREATE INDEX oauth_identities_email_idx ON oauth_identities (lower(email));
