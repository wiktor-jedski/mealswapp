-- Implements DESIGN-009 AdminController deterministic global catalog export.
SELECT
    f.id,
    f.name,
    f.physical_state,
    f.prep_time_minutes,
    f.average_unit_weight_grams::text,
    f.average_serving_volume_milliliters::text,
    f.density_grams_per_milliliter::text,
    f.density_source_provider,
    f.density_source_food_id,
    f.density_source_kind,
    f.protein_per_100::text,
    f.carbohydrates_per_100::text,
    f.fat_per_100::text,
    f.micronutrients,
    f.image_url,
    f.image_alt,
    f.source_provider,
    f.external_id,
    f.deleted_at,
    f.created_at,
    f.updated_at,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('id', c.id, 'name', c.name) ORDER BY c.name COLLATE "C", c.id)
        FROM food_item_classifications fic
        JOIN classifications c ON c.id = fic.classification_id
        WHERE fic.food_item_id = f.id AND c.kind = 'food_category'
    ), '[]'::jsonb),
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('id', c.id, 'name', c.name) ORDER BY c.name COLLATE "C", c.id)
        FROM food_item_classifications fic
        JOIN classifications c ON c.id = fic.classification_id
        WHERE fic.food_item_id = f.id AND c.kind = 'culinary_role'
    ), '[]'::jsonb),
    COALESCE((
        SELECT jsonb_agg(fia.allergen_key ORDER BY fia.allergen_key COLLATE "C")
        FROM food_item_allergens fia
        WHERE fia.food_item_id = f.id
    ), '[]'::jsonb),
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object(
            'id', ci.id,
            'provider', ci.source_provider,
            'externalId', ci.external_id,
            'status', ci.status
        ) ORDER BY ci.source_provider COLLATE "C", ci.external_id COLLATE "C", ci.id)
        FROM curated_imports ci
        WHERE ci.food_item_id = f.id
    ), '[]'::jsonb)
FROM food_items f
WHERE ($1::boolean OR f.deleted_at IS NULL)
ORDER BY f.id;
