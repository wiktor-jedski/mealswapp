-- Implements DESIGN-008 SavedDataRepository owner-safe custom Food Object entries.
ALTER TABLE saved_diet_meal_entries
    ADD COLUMN IF NOT EXISTS custom_food_item_id uuid REFERENCES custom_food_items(id);

ALTER TABLE saved_diet_meal_entries
    DROP CONSTRAINT IF EXISTS saved_diet_entry_exactly_one_food_object;
ALTER TABLE saved_diet_meal_entries
    ADD CONSTRAINT saved_diet_entry_exactly_one_food_object
    CHECK (
        (meal_id IS NOT NULL)::integer
        + (food_item_id IS NOT NULL)::integer
        + (custom_food_item_id IS NOT NULL)::integer = 1
    );

CREATE OR REPLACE FUNCTION validate_saved_diet_entry_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    object_state text;
    diet_owner uuid;
    custom_owner uuid;
    custom_deleted_at timestamptz;
BEGIN
    IF NEW.meal_id IS NOT NULL THEN
        SELECT physical_state INTO object_state FROM meals WHERE id = NEW.meal_id;
    ELSIF NEW.food_item_id IS NOT NULL THEN
        SELECT physical_state INTO object_state FROM food_items WHERE id = NEW.food_item_id;
    ELSE
        SELECT user_id INTO diet_owner FROM saved_diets WHERE id = NEW.saved_diet_id;
        SELECT owner_id, physical_state, deleted_at
        INTO custom_owner, object_state, custom_deleted_at
        FROM custom_food_items
        WHERE id = NEW.custom_food_item_id
        FOR UPDATE;
        IF custom_owner IS NULL OR custom_owner <> diet_owner OR custom_deleted_at IS NOT NULL THEN
            RAISE EXCEPTION 'custom Food Item is unavailable for saved diet owner'
                USING ERRCODE = '23503';
        END IF;
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
BEFORE INSERT OR UPDATE OF saved_diet_id, meal_id, food_item_id, custom_food_item_id, unit
ON saved_diet_meal_entries
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_entry_unit_basis();

CREATE OR REPLACE FUNCTION validate_custom_food_saved_diet_unit_basis()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM saved_diet_meal_entries
        WHERE custom_food_item_id = NEW.id
          AND (NEW.physical_state = 'solid' AND unit NOT IN ('g', 'oz')
               OR NEW.physical_state = 'liquid' AND unit NOT IN ('ml', 'fl_oz'))
    ) THEN
        RAISE EXCEPTION 'custom Food Item physical state % conflicts with saved diet quantities', NEW.physical_state
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_custom_food_unit_basis_trigger ON custom_food_items;
CREATE TRIGGER saved_diet_custom_food_unit_basis_trigger
BEFORE UPDATE OF physical_state ON custom_food_items
FOR EACH ROW
EXECUTE FUNCTION validate_custom_food_saved_diet_unit_basis();

INSERT INTO schema_migrations (version)
VALUES (30)
ON CONFLICT (version) DO NOTHING;
