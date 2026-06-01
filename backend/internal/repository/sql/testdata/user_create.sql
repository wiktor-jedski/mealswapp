-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (email, password_hash, password_salt)
VALUES ($1, 'fixture-hash', 'fixture-salt')
RETURNING id;
