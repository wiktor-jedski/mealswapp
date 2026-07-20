-- Implements DESIGN-008 SavedDataRepository Food Object entries rollback.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'saved_diet_meal_entries'
          AND column_name = 'food_item_id'
    ) THEN
        IF EXISTS (SELECT 1 FROM saved_diet_meal_entries WHERE food_item_id IS NOT NULL) THEN
            RAISE EXCEPTION 'cannot remove Daily Diet Food Item support while Food Item entries exist';
        END IF;
        DROP TRIGGER IF EXISTS saved_diet_food_item_unit_basis_trigger ON food_items;
        DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
        ALTER TABLE saved_diet_meal_entries DROP CONSTRAINT IF EXISTS saved_diet_entry_exactly_one_food_object;
        ALTER TABLE saved_diet_meal_entries DROP COLUMN food_item_id;
        ALTER TABLE saved_diet_meal_entries ALTER COLUMN meal_id SET NOT NULL;
    END IF;
END;
$$;

DROP FUNCTION IF EXISTS validate_food_saved_diet_unit_basis();

CREATE OR REPLACE FUNCTION validate_saved_diet_entry_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    meal_state text;
BEGIN
    SELECT physical_state INTO meal_state FROM meals WHERE id = NEW.meal_id;
    IF meal_state = 'solid' AND NEW.unit NOT IN ('g', 'oz')
       OR meal_state = 'liquid' AND NEW.unit NOT IN ('ml', 'fl_oz') THEN
        RAISE EXCEPTION 'saved diet unit % does not match meal physical state %', NEW.unit, meal_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DO $$
BEGIN
    IF to_regclass('public.saved_diet_meal_entries') IS NOT NULL THEN
        DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
        CREATE TRIGGER saved_diet_entry_unit_basis_trigger
        BEFORE INSERT OR UPDATE OF meal_id, unit ON saved_diet_meal_entries
        FOR EACH ROW
        EXECUTE FUNCTION validate_saved_diet_entry_unit_basis();
    END IF;
END;
$$;

DO $$
BEGIN
    IF to_regclass('public.schema_migrations') IS NOT NULL THEN
        DELETE FROM schema_migrations WHERE version = 23;
    END IF;
END;
$$;
