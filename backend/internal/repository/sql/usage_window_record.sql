-- Implements DESIGN-007 UsageLimiter record query.
INSERT INTO usage_windows (user_id, feature, started_at, search_count)
VALUES ($1, btrim($2), $3, 1)
ON CONFLICT (user_id, feature, started_at)
DO UPDATE SET search_count = usage_windows.search_count + 1, updated_at = now()
RETURNING user_id, feature, started_at, search_count, created_at, updated_at;
