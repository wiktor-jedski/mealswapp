-- Implements DESIGN-013 AuditLogger security event insert.
INSERT INTO security_audit_entries (
    request_id, user_id, action, resource, outcome, ip, user_agent, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;
