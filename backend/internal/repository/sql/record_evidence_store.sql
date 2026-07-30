-- Implements DESIGN-012 DataNormalizer deployment-safe external evidence storage.
DELETE FROM external_record_evidence WHERE expires_at <= $4;
INSERT INTO external_record_evidence (token, provider, external_id, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (token) DO NOTHING;
