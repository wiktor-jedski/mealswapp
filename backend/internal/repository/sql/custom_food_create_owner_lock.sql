-- Implements DESIGN-008 AccountDeleter create/delete serialization.
SELECT id
FROM users
WHERE id = $1 AND deletion_requested_at IS NULL
FOR UPDATE;
