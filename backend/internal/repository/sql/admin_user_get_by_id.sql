-- Implements DESIGN-009 UserAdminPanel exact privacy-minimized lookup.
SELECT account.id,
       account.email_key_version,
       account.email_nonce,
       account.email_ciphertext,
       account.email_verified,
       account.created_at,
       deletion.id,
       deletion.status,
       coalesce(deletion.failure_category, ''),
       deletion.retry_count,
       deletion.requested_at
FROM users AS account
LEFT JOIN LATERAL (
    SELECT request.id,
           request.status,
           request.failure_category,
           request.retry_count,
           request.requested_at
    FROM data_deletion_requests AS request
    WHERE request.user_id = account.id
    ORDER BY request.requested_at DESC, request.id DESC
    LIMIT 1
) AS deletion ON true
WHERE account.id = $1;
