-- Implements DESIGN-005 FoodItemEntity integration fixture.
SELECT EXISTS (SELECT 1 FROM food_items WHERE name = $1);
