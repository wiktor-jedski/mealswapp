-- Implements DESIGN-015 DataRetentionPolicy transition-audit query.
INSERT INTO data_deletion_audit_entries (request_id, from_status, to_status, note)
VALUES ($1, $2, $3, nullif(btrim($4), ''));
