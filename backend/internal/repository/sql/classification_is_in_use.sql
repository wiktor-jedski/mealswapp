-- Implements DESIGN-005 ClassificationEntity usage query.
SELECT EXISTS (SELECT 1 FROM food_item_classifications WHERE classification_id = $1)
    OR EXISTS (SELECT 1 FROM meal_classifications WHERE classification_id = $1)
    OR EXISTS (SELECT 1 FROM custom_food_item_classifications WHERE classification_id = $1)
    OR EXISTS (SELECT 1 FROM classifications WHERE parent_id = $1 AND deleted_at IS NULL);
