-- Implements DESIGN-008 SearchHistoryRepository and DESIGN-013 EncryptionService encrypted query insert.
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
RETURNING id;
