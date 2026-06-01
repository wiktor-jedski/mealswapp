-- Implements DESIGN-005 FoodItemEntity hydrate-tags query.
SELECT t.id, t.name, t.kind, t.parent_id
FROM food_item_tags fit
JOIN tags t ON t.id = fit.tag_id
WHERE fit.food_item_id = $1 AND t.deleted_at IS NULL
ORDER BY t.kind, t.normalized_name, t.id;
