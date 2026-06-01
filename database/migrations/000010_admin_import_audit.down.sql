-- Implements DESIGN-009 DataImporter.
-- Implements DESIGN-009 AdminController.
DROP INDEX IF EXISTS admin_audit_entity_idx;
DROP INDEX IF EXISTS admin_audit_admin_created_idx;
DROP TABLE IF EXISTS admin_audit_entries;
DROP INDEX IF EXISTS curated_imports_status_idx;
DROP INDEX IF EXISTS curated_imports_food_item_idx;
DROP TABLE IF EXISTS curated_imports;

DELETE FROM schema_migrations WHERE version = 10;
