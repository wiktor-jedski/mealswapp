-- Implements DESIGN-006 AuthController verification projection query.
UPDATE users
SET email_verified = true,
    updated_at = now()
WHERE id = $1
RETURNING id;
