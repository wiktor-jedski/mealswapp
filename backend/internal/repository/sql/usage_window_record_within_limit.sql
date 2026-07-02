-- Implements DESIGN-007 UsageLimiter atomic limit-enforced record query.
WITH usage_lock AS (
    SELECT pg_advisory_xact_lock(hashtextextended($1::uuid::text || ':' || btrim($2), 0))
),
current_usage AS (
    SELECT coalesce(sum(w.search_count), 0)::integer AS search_count
    FROM usage_lock
    LEFT JOIN usage_windows w
        ON w.user_id = $1::uuid
       AND w.feature = btrim($2)
       AND w.started_at >= $4
),
recorded_usage AS (
    INSERT INTO usage_windows (user_id, feature, started_at, search_count)
    SELECT $1::uuid, btrim($2), $3, 1
    FROM current_usage
    WHERE current_usage.search_count < $5
    ON CONFLICT (user_id, feature, started_at)
    DO UPDATE SET search_count = usage_windows.search_count + 1, updated_at = now()
    RETURNING user_id
)
SELECT
    $1::uuid,
    btrim($2),
    $4::timestamptz,
    CASE
        WHEN EXISTS (SELECT 1 FROM recorded_usage) THEN current_usage.search_count + 1
        ELSE current_usage.search_count
    END,
    now(),
    now(),
    EXISTS (SELECT 1 FROM recorded_usage)
FROM current_usage;
