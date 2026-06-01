-- Implements DESIGN-015 ConsentManager required-consent query.
SELECT EXISTS (
    SELECT 1
    FROM consent_records
    WHERE user_id = $1
      AND privacy_policy_version = btrim($2)
      AND terms_version = btrim($3)
);
