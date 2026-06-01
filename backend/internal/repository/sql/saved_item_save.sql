-- Implements DESIGN-008 SavedDataRepository save query.
INSERT INTO saved_items (user_id, item_id, kind)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, item_id, kind) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id;
