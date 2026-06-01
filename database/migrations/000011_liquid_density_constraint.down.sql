-- Implements DESIGN-005 FoodItemEntity liquid density invariant rollback.
ALTER TABLE food_items
    DROP CONSTRAINT IF EXISTS food_items_liquid_density_required;

DELETE FROM schema_migrations WHERE version = 11;
