-- Implements DESIGN-009 AdminController global catalog export integration fixture.
WITH foods_inserted AS (
    INSERT INTO food_items (
        id, name, physical_state, prep_time_minutes, average_unit_weight_grams,
        average_serving_volume_milliliters, density_grams_per_milliliter,
        density_source_provider, density_source_food_id, density_source_kind,
        protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients,
        image_url, image_alt, source_provider, external_id, deleted_at, created_at, updated_at
    )
    VALUES
        ($1, 'Żółty płyn', 'liquid', 9, NULL, 250.5000, 1.030000,
         'usda', 'food-42', 'imported', 3.2500, 4.5000, 1.1250,
         '{"VitaminC":4.5000,"Iron":1.2500}'::jsonb,
         'https://example.test/image.png', 'Żółty obraz', 'usda', 'food-42',
         NULL, '2026-07-01T00:00:00Z', '2026-07-02T00:00:00Z'),
        ($2, 'Deleted solid', 'solid', 0, 80.2500, NULL, NULL,
         NULL, NULL, NULL, 10.0000, 20.0000, 5.0000, '{}'::jsonb,
         NULL, NULL, NULL, NULL,
         '2026-07-03T00:00:00Z', '2026-07-01T00:00:00Z', '2026-07-03T00:00:00Z')
    RETURNING id
),
relationships_inserted AS (
    INSERT INTO food_item_classifications (food_item_id, classification_id)
    VALUES ($1, $3), ($1, $4), ($1, $5)
    RETURNING food_item_id
),
allergens_inserted AS (
    INSERT INTO food_item_allergens (food_item_id, allergen_key)
    VALUES ($1, 'gluten'), ($1, 'dairy')
    RETURNING food_item_id
),
curated_inserted AS (
    INSERT INTO curated_imports (id, source_provider, external_id, food_item_id, status)
    VALUES ($6, 'usda', 'food-42', $1, 'imported')
    RETURNING id
)
SELECT 1
FROM foods_inserted
LIMIT 1;
