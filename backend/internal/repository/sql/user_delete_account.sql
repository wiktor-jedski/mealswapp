-- Implements DESIGN-008 AccountDeleter production account delete query.
DELETE FROM users
WHERE id = $1;
