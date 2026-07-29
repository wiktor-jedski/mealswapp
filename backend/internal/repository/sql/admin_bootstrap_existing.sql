-- Implements DESIGN-009 AdminController first-administrator-only invariant.
SELECT id
FROM users
WHERE role = 'admin'
ORDER BY id
LIMIT 1
FOR UPDATE;
