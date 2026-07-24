-- Implements DESIGN-009 TagManager hierarchy integrity under concurrent administration.
CREATE OR REPLACE FUNCTION validate_classification_hierarchy()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    parent_kind text;
    cycle_found boolean;
BEGIN
	PERFORM pg_advisory_xact_lock(hashtext('mealswapp:global-classifications'));
	IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT kind INTO parent_kind
    FROM classifications
    WHERE id = NEW.parent_id AND deleted_at IS NULL
    FOR SHARE;

    IF parent_kind IS NULL OR parent_kind <> NEW.kind THEN
		RAISE EXCEPTION 'classification parent must be active and have the same kind' USING ERRCODE = '23514', CONSTRAINT = 'classification_parent_invalid';
    END IF;

    WITH RECURSIVE descendants AS (
        SELECT id FROM classifications WHERE parent_id = NEW.id AND deleted_at IS NULL
        UNION ALL
        SELECT child.id
        FROM classifications child
        JOIN descendants ON descendants.id = child.parent_id
        WHERE child.deleted_at IS NULL
    )
    SELECT EXISTS (SELECT 1 FROM descendants WHERE id = NEW.parent_id) INTO cycle_found;

    IF NEW.parent_id = NEW.id OR cycle_found THEN
		RAISE EXCEPTION 'classification hierarchy cycle' USING ERRCODE = '23514', CONSTRAINT = 'classification_hierarchy_cycle';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS classifications_hierarchy_guard ON classifications;
CREATE TRIGGER classifications_hierarchy_guard
BEFORE INSERT OR UPDATE OF parent_id, kind ON classifications
FOR EACH ROW EXECUTE FUNCTION validate_classification_hierarchy();

CREATE OR REPLACE FUNCTION validate_active_classification_assignment()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	PERFORM pg_advisory_xact_lock(hashtext('mealswapp:global-classifications'));
	IF NOT EXISTS (SELECT 1 FROM classifications WHERE id = NEW.classification_id AND deleted_at IS NULL) THEN
		RAISE EXCEPTION 'classification assignment requires an active classification' USING ERRCODE = '23503', CONSTRAINT = 'classification_assignment_inactive';
	END IF;
	RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION block_in_use_classification_delete()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
		PERFORM pg_advisory_xact_lock(hashtext('mealswapp:global-classifications'));
		IF EXISTS (SELECT 1 FROM food_item_classifications WHERE classification_id = NEW.id)
			OR EXISTS (SELECT 1 FROM meal_classifications WHERE classification_id = NEW.id)
			OR EXISTS (SELECT 1 FROM custom_food_item_classifications WHERE classification_id = NEW.id)
			OR EXISTS (SELECT 1 FROM classifications WHERE parent_id = NEW.id AND deleted_at IS NULL) THEN
			RAISE EXCEPTION 'classification is in use' USING ERRCODE = '23503', CONSTRAINT = 'classification_in_use';
		END IF;
	END IF;
	RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS classifications_in_use_guard ON classifications;
CREATE TRIGGER classifications_in_use_guard
BEFORE UPDATE OF deleted_at ON classifications
FOR EACH ROW EXECUTE FUNCTION block_in_use_classification_delete();

DROP TRIGGER IF EXISTS food_item_classification_active_guard ON food_item_classifications;
CREATE TRIGGER food_item_classification_active_guard
BEFORE INSERT OR UPDATE OF classification_id ON food_item_classifications
FOR EACH ROW EXECUTE FUNCTION validate_active_classification_assignment();

DROP TRIGGER IF EXISTS meal_classification_active_guard ON meal_classifications;
CREATE TRIGGER meal_classification_active_guard
BEFORE INSERT OR UPDATE OF classification_id ON meal_classifications
FOR EACH ROW EXECUTE FUNCTION validate_active_classification_assignment();

DROP TRIGGER IF EXISTS custom_food_item_classification_active_guard ON custom_food_item_classifications;
CREATE TRIGGER custom_food_item_classification_active_guard
BEFORE INSERT OR UPDATE OF classification_id ON custom_food_item_classifications
FOR EACH ROW EXECUTE FUNCTION validate_active_classification_assignment();

INSERT INTO schema_migrations (version)
VALUES (28)
ON CONFLICT (version) DO NOTHING;
