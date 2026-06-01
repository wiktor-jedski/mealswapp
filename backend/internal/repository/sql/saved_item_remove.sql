-- Implements DESIGN-008 SavedDataRepository remove query.
DELETE FROM saved_items
WHERE user_id = $1 AND item_id = $2 AND kind = $3;
