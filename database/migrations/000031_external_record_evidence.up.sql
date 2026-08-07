-- Implements DESIGN-012 DataNormalizer deployment-safe external evidence storage.
CREATE TABLE IF NOT EXISTS external_record_evidence (
    token text PRIMARY KEY,
    provider text NOT NULL,
    external_id text NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS external_record_evidence_expiry_idx ON external_record_evidence (expires_at);

INSERT INTO schema_migrations (version) VALUES (31) ON CONFLICT (version) DO NOTHING;
