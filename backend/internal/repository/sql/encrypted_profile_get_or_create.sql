-- Implements DESIGN-008 PreferenceManager and DESIGN-013 EncryptionService encrypted profile read query.
WITH inserted AS (
    INSERT INTO user_profiles (user_id)
    VALUES ($1)
    ON CONFLICT (user_id) DO NOTHING
    RETURNING user_id,
              display_name_key_version,
              display_name_nonce,
              display_name_ciphertext,
              unit_system,
              theme_preference,
              created_at,
              updated_at
)
SELECT user_id,
       display_name_key_version,
       display_name_nonce,
       display_name_ciphertext,
       unit_system,
       theme_preference,
       created_at,
       updated_at
FROM inserted
UNION ALL
SELECT user_id,
       display_name_key_version,
       display_name_nonce,
       display_name_ciphertext,
       unit_system,
       theme_preference,
       created_at,
       updated_at
FROM user_profiles
WHERE user_id = $1
LIMIT 1;
