-- Implements DESIGN-015 ConsentManager account-export list query.
SELECT id, user_id, privacy_policy_version, terms_version, accepted_at
FROM consent_records
WHERE user_id = $1
ORDER BY accepted_at DESC, id;
