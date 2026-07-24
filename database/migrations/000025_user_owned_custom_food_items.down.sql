-- Implements DESIGN-005 FoodItemEntity owner-scoped custom-item persistence rollback.
DROP INDEX IF EXISTS custom_food_item_classifications_classification_idx;
DROP TABLE IF EXISTS custom_food_item_classifications;
DROP INDEX IF EXISTS custom_food_items_owner_idx;
DROP INDEX IF EXISTS custom_food_items_owner_active_name_idx;
DROP TABLE IF EXISTS custom_food_items;

DELETE FROM schema_migrations WHERE version = 25;
