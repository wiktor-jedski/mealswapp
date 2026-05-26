-- Implements DESIGN-005 RepositoryInterfaces migration rollback bookkeeping.
DELETE FROM schema_migrations WHERE version = 1;
