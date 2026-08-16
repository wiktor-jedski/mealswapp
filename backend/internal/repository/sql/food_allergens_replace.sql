-- Implements DESIGN-009 ItemCurator atomic allergen replacement.
INSERT INTO food_item_allergens (food_item_id, allergen_key)
SELECT $1, key
FROM unnest($2::text[]) AS keys(key)
ORDER BY key;
