-- Implements DESIGN-006 AuthController password-reset password update query.
UPDATE users
SET password_hash = $2,
    password_salt = $3,
    updated_at = now()
WHERE id = $1
RETURNING id;
