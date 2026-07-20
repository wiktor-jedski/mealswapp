-- Implements DESIGN-007 SubscriptionController checkout idempotency query.
INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
VALUES ($1, $2, $3, $4, $5, $6, $7);
