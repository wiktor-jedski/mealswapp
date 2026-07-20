-- Implements DESIGN-008 SavedDataRepository Food Object entries.
ALTER TABLE saved_diet_meal_entries
    ALTER COLUMN meal_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS food_item_id uuid REFERENCES food_items(id);

ALTER TABLE saved_diet_meal_entries
    DROP CONSTRAINT IF EXISTS saved_diet_entry_exactly_one_food_object;
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

DROP TRIGGER IF EXISTS saved_diet_entry_unit_basis_trigger ON saved_diet_meal_entries;
CREATE TRIGGER saved_diet_entry_unit_basis_trigger
BEFORE INSERT OR UPDATE OF meal_id, food_item_id, unit ON saved_diet_meal_entries
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_entry_unit_basis();

CREATE OR REPLACE FUNCTION validate_food_saved_diet_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM saved_diet_meal_entries
        WHERE food_item_id = NEW.id
          AND (NEW.physical_state = 'solid' AND unit NOT IN ('g', 'oz')
               OR NEW.physical_state = 'liquid' AND unit NOT IN ('ml', 'fl_oz'))
    ) THEN
        RAISE EXCEPTION 'Food Item physical state % conflicts with saved diet quantities', NEW.physical_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_food_item_unit_basis_trigger ON food_items;
CREATE TRIGGER saved_diet_food_item_unit_basis_trigger
BEFORE UPDATE OF physical_state ON food_items
FOR EACH ROW
EXECUTE FUNCTION validate_food_saved_diet_unit_basis();

INSERT INTO schema_migrations (version)
VALUES (23)
ON CONFLICT (version) DO NOTHING;
