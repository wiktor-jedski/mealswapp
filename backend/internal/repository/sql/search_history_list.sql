-- Implements DESIGN-008 SearchHistoryRepository list query.
SELECT id, user_id, query, mode, filters_hash, created_at
FROM search_history
WHERE user_id = $1
ORDER BY created_at DESC, id
LIMIT $2;
