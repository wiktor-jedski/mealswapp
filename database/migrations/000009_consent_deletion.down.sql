-- Implements DESIGN-015 ConsentManager.
-- Implements DESIGN-015 DataRetentionPolicy.
DROP INDEX IF EXISTS data_deletion_audit_request_idx;
DROP TABLE IF EXISTS data_deletion_audit_entries;
DROP INDEX IF EXISTS data_deletion_requests_status_idx;
DROP INDEX IF EXISTS data_deletion_requests_active_user_idx;
DROP TABLE IF EXISTS data_deletion_requests;
DROP INDEX IF EXISTS consent_records_user_idx;
DROP TABLE IF EXISTS consent_records;

DELETE FROM schema_migrations WHERE version = 9;
