-- Implements DESIGN-008 PreferenceManager unit-system query.
SELECT unit_system
FROM user_profiles
WHERE user_id = $1;
