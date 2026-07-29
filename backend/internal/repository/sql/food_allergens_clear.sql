-- Implements DESIGN-009 ItemCurator atomic allergen replacement.
DELETE FROM food_item_allergens
WHERE food_item_id = $1;
