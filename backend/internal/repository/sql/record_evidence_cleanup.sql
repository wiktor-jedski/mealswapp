-- Implements DESIGN-012 DataNormalizer expired evidence cleanup.
DELETE FROM external_record_evidence WHERE expires_at <= $1;
