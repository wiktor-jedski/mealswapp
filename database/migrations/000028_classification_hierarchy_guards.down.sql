-- Implements DESIGN-009 TagManager hierarchy integrity rollback.
DROP TRIGGER IF EXISTS custom_food_item_classification_active_guard ON custom_food_item_classifications;
DROP TRIGGER IF EXISTS meal_classification_active_guard ON meal_classifications;
DROP TRIGGER IF EXISTS food_item_classification_active_guard ON food_item_classifications;
DROP TRIGGER IF EXISTS classifications_in_use_guard ON classifications;
DROP TRIGGER IF EXISTS classifications_hierarchy_guard ON classifications;
DROP FUNCTION IF EXISTS block_in_use_classification_delete();
DROP FUNCTION IF EXISTS validate_active_classification_assignment();
DROP FUNCTION IF EXISTS validate_classification_hierarchy();
DELETE FROM schema_migrations WHERE version = 28;
