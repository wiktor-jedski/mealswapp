-- Implements DESIGN-005 FoodItemEntity hydrate-classifications query.
SELECT t.id, t.name, t.kind, t.parent_id
FROM food_item_classifications fit
JOIN classifications t ON t.id = fit.classification_id
WHERE fit.food_item_id = $1 AND t.deleted_at IS NULL
ORDER BY t.kind, t.normalized_name, t.id;
