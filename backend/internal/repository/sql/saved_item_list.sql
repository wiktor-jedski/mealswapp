-- Implements DESIGN-008 SavedDataRepository list query.
SELECT id, user_id, item_id, kind, created_at
FROM saved_items
WHERE user_id = $1 AND ($2::text = '' OR kind = $2)
ORDER BY created_at DESC, id;
