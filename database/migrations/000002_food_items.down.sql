-- Implements DESIGN-005 FoodItemEntity.
DROP INDEX IF EXISTS food_items_prep_time_idx;
DROP INDEX IF EXISTS food_items_physical_state_idx;
DROP INDEX IF EXISTS food_items_name_search_idx;
DROP INDEX IF EXISTS food_items_active_normalized_name_idx;
DROP TABLE IF EXISTS food_items;

DELETE FROM schema_migrations WHERE version = 2;
