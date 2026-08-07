-- Implements DESIGN-012 DataNormalizer fail-closed external evidence.
SELECT provider, external_id
FROM external_record_evidence
WHERE token = $1 AND expires_at > $2
