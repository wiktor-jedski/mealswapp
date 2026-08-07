-- Implements DESIGN-009 ItemCurator acceptance proof.
SELECT count(*)
FROM food_items
WHERE name = $1;
