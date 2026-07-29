-- Implements DESIGN-005 UnitConverter byte-for-byte metric persistence verification.
SELECT row_to_json(food_items)::text
FROM food_items
WHERE id = $1;
