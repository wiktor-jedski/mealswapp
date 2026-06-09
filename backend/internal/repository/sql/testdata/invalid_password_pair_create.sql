-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (
    email_key_version,
    email_nonce,
    email_ciphertext,
    normalized_email_lookup_key_version,
    normalized_email_digest,
    password_hash
)
VALUES ('test-v1', decode('696e76616c69642d6e6f6e6365', 'hex'), convert_to('invalid-password-pair@example.test', 'UTF8'), 'test-v1', 'invalid-password-pair@example.test', 'hash-without-salt');
