-- Implements DESIGN-005 FoodItemEntity allergen rollback migration.
DROP TABLE IF EXISTS food_item_allergens;

DELETE FROM schema_migrations WHERE version = 16;
