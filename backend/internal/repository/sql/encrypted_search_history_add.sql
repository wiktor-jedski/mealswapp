-- Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService encrypted query insert.
WITH inserted AS (
    INSERT INTO search_history (
    user_id,
    query,
    query_key_version,
    query_nonce,
    query_ciphertext,
    mode,
    filters_hash
)
    VALUES ($1, 'encrypted', $2, $3, $4, btrim($5), $6)
    RETURNING id
),
pruned AS (
    DELETE FROM search_history
    WHERE user_id = $1
      AND id NOT IN (
          SELECT id
          FROM search_history
          WHERE user_id = $1
          ORDER BY created_at DESC, id DESC
          LIMIT 99
      )
)
SELECT id FROM inserted;
