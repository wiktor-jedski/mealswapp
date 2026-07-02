-- Implements DESIGN-007 UsageLimiter cross-instance atomic usage guard.
SELECT pg_advisory_xact_lock(hashtextextended($1::uuid::text || ':' || btrim($2), 0));
