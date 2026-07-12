-- Implements DESIGN-008 SavedDataRepository.
DROP TRIGGER IF EXISTS saved_diet_saved_item_cleanup_trigger ON saved_diets;
DROP TRIGGER IF EXISTS saved_items_saved_diet_target_trigger ON saved_items;
DROP FUNCTION IF EXISTS delete_saved_diet_saved_item();
DROP FUNCTION IF EXISTS validate_saved_diet_saved_item_target();
DROP TABLE IF EXISTS saved_diet_meal_entries;
DROP TABLE IF EXISTS saved_diets;

DELETE FROM schema_migrations WHERE version = 19;
