-- Implements DESIGN-009 DataImporter cross-process natural-identity serialization.
SELECT pg_advisory_xact_lock(hashtextextended(btrim($1) || chr(31) || btrim($2), 0));
