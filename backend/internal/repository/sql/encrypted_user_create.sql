-- Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService encrypted create query.
INSERT INTO users (
    email,
    email_key_version,
    email_nonce,
    email_ciphertext,
    normalized_email_lookup_key_version,
    normalized_email_digest,
    role,
    email_verified,
    password_hash,
    password_salt
)
VALUES (
    'encrypted:' || $5,
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING id;
