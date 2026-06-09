-- Implements DESIGN-008 SearchHistoryRepository clear query.
DELETE FROM search_history
WHERE user_id = $1;
