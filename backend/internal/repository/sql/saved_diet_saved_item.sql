-- Implements DESIGN-008 SavedDataRepository.
INSERT INTO saved_items (user_id, item_id, kind)
VALUES ($1, $2, 'saved_diet')
ON CONFLICT (user_id, item_id, kind) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id;
