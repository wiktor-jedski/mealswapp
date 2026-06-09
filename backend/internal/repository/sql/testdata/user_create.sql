-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (
    email_key_version,
    email_nonce,
    email_ciphertext,
    normalized_email_lookup_key_version,
    normalized_email_digest,
    password_hash,
    password_salt
)
VALUES ('test-v1', decode('746573742d6e6f6e6365', 'hex'), convert_to($1, 'UTF8'), 'test-v1', lower($1), 'fixture-hash', 'fixture-salt')
RETURNING id;
