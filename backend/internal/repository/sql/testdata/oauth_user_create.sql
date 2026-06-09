-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (
    email_key_version,
    email_nonce,
    email_ciphertext,
    normalized_email_lookup_key_version,
    normalized_email_digest
)
VALUES ('test-v1', decode('6f617574682d6e6f6e6365', 'hex'), convert_to('oauth-only@example.test', 'UTF8'), 'test-v1', 'oauth-only@example.test')
RETURNING id;
