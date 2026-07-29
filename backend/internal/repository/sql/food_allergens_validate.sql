-- Implements DESIGN-009 ItemCurator active allergen validation.
SELECT count(*) = COALESCE(cardinality($1::text[]), 0)
FROM allergen_vocabulary
WHERE deleted_at IS NULL
  AND key = ANY($1::text[]);
