-- Implements DESIGN-009 ItemCurator acceptance proof.
SELECT count(*)
FROM mutation_idempotency_keys
WHERE key = $1;
