-- Implements DESIGN-008 PreferenceManager get-or-create query.
INSERT INTO user_profiles (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING user_id, coalesce(display_name, ''), unit_system, theme_preference, created_at, updated_at;
