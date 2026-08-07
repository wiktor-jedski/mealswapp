-- Implements DESIGN-012 DataNormalizer deployment-safe external evidence cleanup.
DROP TABLE IF EXISTS external_record_evidence;
DELETE FROM schema_migrations WHERE version = 31;
