-- Implements DESIGN-015 ConsentManager record query.
INSERT INTO consent_records (user_id, privacy_policy_version, terms_version)
VALUES ($1, btrim($2), btrim($3))
ON CONFLICT (user_id, privacy_policy_version, terms_version)
DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id;
