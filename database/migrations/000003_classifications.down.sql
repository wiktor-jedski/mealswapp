-- Implements DESIGN-005 ClassificationEntity.
DROP INDEX IF EXISTS meal_classifications_classification_idx;
DROP TABLE IF EXISTS meal_classifications;
DROP INDEX IF EXISTS food_item_classifications_classification_idx;
DROP TABLE IF EXISTS food_item_classifications;
DROP INDEX IF EXISTS meal_tags_tag_idx;
DROP TABLE IF EXISTS meal_tags;
DROP INDEX IF EXISTS food_item_tags_tag_idx;
DROP TABLE IF EXISTS food_item_tags;
DROP INDEX IF EXISTS classifications_kind_parent_idx;
DROP INDEX IF EXISTS classifications_active_child_name_idx;
DROP INDEX IF EXISTS classifications_active_root_name_idx;
DROP TABLE IF EXISTS classifications;
DROP INDEX IF EXISTS tags_kind_parent_idx;
DROP INDEX IF EXISTS tags_active_child_name_idx;
DROP INDEX IF EXISTS tags_active_root_name_idx;
DROP TABLE IF EXISTS tags;

DELETE FROM schema_migrations WHERE version = 3;
