-- Implements DESIGN-006 AccountLockoutTracker successful-login reset query.
UPDATE users
SET failed_login_count = 0,
    locked_until = NULL,
    updated_at = now()
WHERE id = $1
RETURNING failed_login_count, locked_until;
