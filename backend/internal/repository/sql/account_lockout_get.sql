-- Implements DESIGN-006 AccountLockoutTracker lookup query.
SELECT failed_login_count, locked_until
FROM users
WHERE id = $1;
