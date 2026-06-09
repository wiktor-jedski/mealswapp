-- Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService encrypted query list.
SELECT id,
       user_id,
       query_key_version,
       query_nonce,
       query_ciphertext,
       mode,
       filters_hash,
       created_at
FROM search_history
WHERE user_id = $1
  AND query_key_version IS NOT NULL
ORDER BY created_at DESC, id
LIMIT $2;
