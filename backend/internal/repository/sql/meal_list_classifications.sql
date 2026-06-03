-- Implements DESIGN-005 MealEntity hydrate-classifications query.
SELECT t.id, t.name, t.kind, t.parent_id
FROM meal_classifications mt
JOIN classifications t ON t.id = mt.classification_id
WHERE mt.meal_id = $1 AND t.deleted_at IS NULL
ORDER BY t.kind, t.normalized_name, t.id;
