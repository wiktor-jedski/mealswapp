-- Implements DESIGN-005 FoodItemEntity custom-item classification hydration query.
SELECT c.id, c.name, c.kind, c.parent_id
FROM custom_food_item_classifications assignment
JOIN classifications c ON c.id = assignment.classification_id
WHERE assignment.custom_food_item_id = $1 AND c.deleted_at IS NULL
ORDER BY c.kind, c.normalized_name, c.id;
