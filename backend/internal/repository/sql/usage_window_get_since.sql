-- Implements DESIGN-007 UsageLimiter aggregate query.
SELECT $1::uuid, btrim($2), $3::timestamptz, coalesce(sum(search_count), 0)::integer,
       coalesce(min(created_at), $3::timestamptz), coalesce(max(updated_at), $3::timestamptz)
FROM usage_windows
WHERE user_id = $1 AND feature = btrim($2) AND started_at >= $3;
