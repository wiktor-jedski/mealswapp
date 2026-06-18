-- Implements DESIGN-008 SearchHistoryRepository bounded retention policy.
DELETE FROM search_history
WHERE user_id = $1
  AND id NOT IN (
      SELECT id
      FROM search_history
      WHERE user_id = $1
      ORDER BY created_at DESC, id DESC
      LIMIT 100
  );
