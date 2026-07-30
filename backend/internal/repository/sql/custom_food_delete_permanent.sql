-- Implements DESIGN-008 AccountDeleter permanent custom-item deletion.
WITH affected AS (
    SELECT d.id, d.name
    FROM saved_diet_meal_entries e
    JOIN saved_diets d ON d.id = e.saved_diet_id
    WHERE e.custom_food_item_id = $2 AND d.user_id = $1
    ORDER BY d.id
    LIMIT 25
), deleted AS (
    DELETE FROM custom_food_items
    WHERE owner_id = $1 AND id = $2
      AND NOT EXISTS (SELECT 1 FROM affected)
    RETURNING id
)
SELECT affected.id, affected.name, COALESCE(deleted.id IS NOT NULL, false)
FROM affected
RIGHT JOIN deleted ON true;
