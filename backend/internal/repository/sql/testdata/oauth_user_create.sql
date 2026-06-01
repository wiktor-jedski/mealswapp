-- Implements DESIGN-006 AuthController integration fixture.
INSERT INTO users (email)
VALUES ('oauth-only@example.test')
RETURNING id;
