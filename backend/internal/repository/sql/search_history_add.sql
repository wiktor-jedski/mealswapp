-- Implements DESIGN-008 SearchHistoryRepository add query.
INSERT INTO search_history (user_id, query, mode, filters_hash)
VALUES ($1, btrim($2), btrim($3), $4)
RETURNING id;
