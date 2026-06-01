-- Implements DESIGN-008 PreferenceManager update query.
UPDATE user_profiles
SET display_name = nullif(btrim($2), ''),
    unit_system = $3,
    theme_preference = $4,
    updated_at = now()
WHERE user_id = $1
RETURNING user_id, coalesce(display_name, ''), unit_system, theme_preference, created_at, updated_at;
