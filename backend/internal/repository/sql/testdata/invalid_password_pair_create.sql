-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (email, password_hash)
VALUES ('invalid-password-pair@example.test', 'hash-without-salt');
