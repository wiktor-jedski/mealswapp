-- Implements DESIGN-015 DataRetentionPolicy status query.
SELECT status
FROM data_deletion_requests
WHERE id = $1;
