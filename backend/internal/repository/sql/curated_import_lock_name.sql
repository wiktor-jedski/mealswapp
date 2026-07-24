-- Implements DESIGN-009 DataImporter cross-process normalized-name serialization.
SELECT pg_advisory_xact_lock(hashtextextended('curated-import-name:' || lower(btrim($1)), 0));
