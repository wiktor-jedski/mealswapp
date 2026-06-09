-- Implements DESIGN-006 AuthController password-reset token create query.
INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);
