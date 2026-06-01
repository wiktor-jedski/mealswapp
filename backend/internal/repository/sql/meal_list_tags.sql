-- Implements DESIGN-005 MealEntity hydrate-tags query.
SELECT t.id, t.name, t.kind, t.parent_id
FROM meal_tags mt
JOIN tags t ON t.id = mt.tag_id
WHERE mt.meal_id = $1 AND t.deleted_at IS NULL
ORDER BY t.kind, t.normalized_name, t.id;
