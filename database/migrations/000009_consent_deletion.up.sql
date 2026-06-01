-- Implements DESIGN-015 ConsentManager.
-- Implements DESIGN-015 DataRetentionPolicy.
CREATE TABLE IF NOT EXISTS consent_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    privacy_policy_version text NOT NULL,
    terms_version text NOT NULL,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT consent_records_privacy_not_blank CHECK (btrim(privacy_policy_version) <> ''),
    CONSTRAINT consent_records_terms_not_blank CHECK (btrim(terms_version) <> ''),
    UNIQUE (user_id, privacy_policy_version, terms_version)
);

CREATE INDEX IF NOT EXISTS consent_records_user_idx
    ON consent_records (user_id, accepted_at DESC);

CREATE TABLE IF NOT EXISTS data_deletion_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    requested_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    failure_reason text,
    CONSTRAINT data_deletion_requests_completed_status CHECK (
        (status = 'completed' AND completed_at IS NOT NULL)
        OR (status <> 'completed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS data_deletion_requests_active_user_idx
    ON data_deletion_requests (user_id)
    WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS data_deletion_requests_status_idx
    ON data_deletion_requests (status, requested_at);

CREATE TABLE IF NOT EXISTS data_deletion_audit_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id uuid NOT NULL REFERENCES data_deletion_requests(id) ON DELETE CASCADE,
    from_status text,
    to_status text NOT NULL CHECK (to_status IN ('pending', 'processing', 'completed', 'failed')),
    note text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS data_deletion_audit_request_idx
    ON data_deletion_audit_entries (request_id, created_at);

INSERT INTO schema_migrations (version)
VALUES (9)
ON CONFLICT (version) DO NOTHING;
