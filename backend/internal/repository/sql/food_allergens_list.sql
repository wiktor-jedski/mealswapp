-- Implements DESIGN-009 ItemCurator deterministic allergen hydration.
SELECT allergen_key
FROM food_item_allergens
WHERE food_item_id = $1
ORDER BY allergen_key;
