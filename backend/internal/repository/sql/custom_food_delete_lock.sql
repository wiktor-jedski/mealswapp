-- Implements DESIGN-008 AccountDeleter serialized permanent custom-item deletion.
SELECT c.id
FROM users u
JOIN custom_food_items c ON c.owner_id = u.id
WHERE u.id = $1 AND u.deletion_requested_at IS NULL AND c.id = $2
FOR UPDATE OF u, c;
