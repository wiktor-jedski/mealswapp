-- Implements DESIGN-008 SavedDataRepository custom Food Object rollback.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM saved_diet_meal_entries WHERE custom_food_item_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove Daily Diet custom Food Item support while entries exist';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
DROP TRIGGER IF EXISTS saved_diet_custom_food_unit_basis_trigger ON custom_food_items;
DROP FUNCTION IF EXISTS validate_custom_food_saved_diet_unit_basis();
ALTER TABLE saved_diet_meal_entries DROP CONSTRAINT IF EXISTS saved_diet_entry_exactly_one_food_object;
ALTER TABLE saved_diet_meal_entries DROP COLUMN IF EXISTS custom_food_item_id;
ALTER TABLE saved_diet_meal_entries
    ADD CONSTRAINT saved_diet_entry_exactly_one_food_object
    CHECK ((meal_id IS NOT NULL)::integer + (food_item_id IS NOT NULL)::integer = 1);

CREATE OR REPLACE FUNCTION validate_saved_diet_entry_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    object_state text;
BEGIN
    IF NEW.meal_id IS NOT NULL THEN
        SELECT physical_state INTO object_state FROM meals WHERE id = NEW.meal_id;
    ELSE
        SELECT physical_state INTO object_state FROM food_items WHERE id = NEW.food_item_id;
    END IF;
    IF object_state = 'solid' AND NEW.unit NOT IN ('g', 'oz')
       OR object_state = 'liquid' AND NEW.unit NOT IN ('ml', 'fl_oz') THEN
        RAISE EXCEPTION 'saved diet unit % does not match Food Object physical state %', NEW.unit, object_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER saved_diet_entry_unit_basis_trigger
BEFORE INSERT OR UPDATE OF meal_id, food_item_id, unit ON saved_diet_meal_entries
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_entry_unit_basis();

DELETE FROM schema_migrations WHERE version = 30;
