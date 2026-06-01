-- Implements DESIGN-005 TagEntity usage query.
SELECT EXISTS (SELECT 1 FROM food_item_tags WHERE tag_id = $1)
    OR EXISTS (SELECT 1 FROM meal_tags WHERE tag_id = $1);
