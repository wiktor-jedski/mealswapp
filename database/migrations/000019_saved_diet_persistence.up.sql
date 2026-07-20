-- Implements DESIGN-008 SavedDataRepository.
CREATE TABLE IF NOT EXISTS saved_diets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL DEFAULT 'Daily Diet',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT saved_diets_name_not_blank CHECK (btrim(name) <> '')
);

CREATE INDEX IF NOT EXISTS saved_diets_user_created_idx
    ON saved_diets (user_id, created_at DESC, id);

CREATE TABLE IF NOT EXISTS saved_diet_meal_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    saved_diet_id uuid NOT NULL REFERENCES saved_diets(id) ON DELETE CASCADE,
    meal_id uuid NOT NULL REFERENCES meals(id),
    quantity numeric NOT NULL,
    unit text NOT NULL,
    position integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT saved_diet_meal_entries_quantity_positive CHECK (quantity > 0),
    CONSTRAINT saved_diet_meal_entries_unit_canonical CHECK (unit IN ('g', 'ml', 'oz', 'fl_oz')),
    CONSTRAINT saved_diet_meal_entries_position_non_negative CHECK (position >= 0),
    UNIQUE (saved_diet_id, position)
);

CREATE INDEX IF NOT EXISTS saved_diet_meal_entries_diet_position_idx
    ON saved_diet_meal_entries (saved_diet_id, position, id);

-- A polymorphic saved_items.item_id cannot use a normal foreign key. These
-- triggers provide the equivalent invariant for the reserved saved_diet kind.
CREATE OR REPLACE FUNCTION validate_saved_diet_saved_item_target()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.kind = 'saved_diet'
       AND NOT EXISTS (
           SELECT 1
           FROM saved_diets
           WHERE id = NEW.item_id
             AND user_id = NEW.user_id
       ) THEN
        RAISE EXCEPTION 'saved_diet target must exist for the saved-item owner'
            USING ERRCODE = '23503';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS saved_items_saved_diet_target_trigger ON saved_items;
CREATE TRIGGER saved_items_saved_diet_target_trigger
BEFORE INSERT OR UPDATE OF item_id, user_id, kind ON saved_items
FOR EACH ROW
EXECUTE FUNCTION validate_saved_diet_saved_item_target();

CREATE OR REPLACE FUNCTION delete_saved_diet_saved_item()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM saved_items
    WHERE user_id = OLD.user_id
      AND item_id = OLD.id
      AND kind = 'saved_diet';
    RETURN OLD;
END;
$$;

DROP TRIGGER IF EXISTS saved_diet_saved_item_cleanup_trigger ON saved_diets;
CREATE TRIGGER saved_diet_saved_item_cleanup_trigger
AFTER DELETE ON saved_diets
FOR EACH ROW
EXECUTE FUNCTION delete_saved_diet_saved_item();

INSERT INTO schema_migrations (version)
VALUES (19)
ON CONFLICT (version) DO NOTHING;
