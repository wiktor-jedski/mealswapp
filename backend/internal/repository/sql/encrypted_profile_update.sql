-- Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService encrypted display-name query.
UPDATE user_profiles
SET display_name = CASE WHEN $2::text IS NULL THEN NULL ELSE 'encrypted' END,
    display_name_key_version = $2,
    display_name_nonce = $3,
    display_name_ciphertext = $4,
    unit_system = $5,
    theme_preference = $6,
    updated_at = now()
WHERE user_id = $1
RETURNING user_id, display_name_key_version, display_name_nonce, display_name_ciphertext, unit_system, theme_preference, created_at, updated_at;
