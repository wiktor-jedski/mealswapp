CREATE TABLE consent_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    privacy_policy_version text NOT NULL,
    terms_version text NOT NULL,
    nutrition_disclaimer_version text NOT NULL,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    ip_address text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT consent_records_versions_present CHECK (
        length(trim(privacy_policy_version)) > 0
        AND length(trim(terms_version)) > 0
        AND length(trim(nutrition_disclaimer_version)) > 0
    )
);

CREATE INDEX consent_records_user_versions_idx ON consent_records (
    user_id,
    privacy_policy_version,
    terms_version,
    nutrition_disclaimer_version
);
