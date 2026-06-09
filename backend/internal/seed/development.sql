-- Implements DESIGN-005 MicronutrientVocabulary deterministic development fixtures.
INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES
    ('VitaminC', 'Vitamin C', 'mg', true),
    ('Calcium', 'Calcium', 'mg', true),
    ('Iron', 'Iron', 'mg', true)
ON CONFLICT (key) DO UPDATE
SET display_name = EXCLUDED.display_name, unit = EXCLUDED.unit, active = EXCLUDED.active;

-- Implements DESIGN-005 ClassificationEntity deterministic development fixtures.
INSERT INTO classifications (id, name, kind)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'Fruit', 'food_category'),
    ('20000000-0000-0000-0000-000000000002', 'Protein', 'food_category'),
    ('20000000-0000-0000-0000-000000000101', 'Quick', 'culinary_role'),
    ('20000000-0000-0000-0000-000000000102', 'Breakfast', 'culinary_role')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- Implements DESIGN-005 FoodItemEntity deterministic development fixtures.
INSERT INTO food_items (
    id, name, physical_state, prep_time_minutes, average_unit_weight_grams, average_serving_volume_milliliters,
    density_grams_per_milliliter, density_source_provider, density_source_food_id, density_source_kind,
    protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients, image_url
)
VALUES
    ('21000000-0000-0000-0000-000000000001', 'Seed Apple', 'solid', 1, 150, NULL, NULL, NULL, NULL, NULL, 0.3, 14, 0.2, '{"VitaminC":4.6}'::jsonb, 'https://example.test/seed-apple.jpg'),
    ('21000000-0000-0000-0000-000000000002', 'Seed Yogurt', 'liquid', 0, NULL, 125, 1, 'fixture', 'seed-yogurt', 'manual', 10, 4, 2, '{"Calcium":120}'::jsonb, 'https://example.test/seed-yogurt.jpg')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name, physical_state = EXCLUDED.physical_state,
    prep_time_minutes = EXCLUDED.prep_time_minutes, average_unit_weight_grams = EXCLUDED.average_unit_weight_grams,
    average_serving_volume_milliliters = EXCLUDED.average_serving_volume_milliliters,
    density_grams_per_milliliter = EXCLUDED.density_grams_per_milliliter,
    density_source_provider = EXCLUDED.density_source_provider, density_source_food_id = EXCLUDED.density_source_food_id,
    density_source_kind = EXCLUDED.density_source_kind,
    protein_per_100 = EXCLUDED.protein_per_100, carbohydrates_per_100 = EXCLUDED.carbohydrates_per_100,
    fat_per_100 = EXCLUDED.fat_per_100, micronutrients = EXCLUDED.micronutrients,
    image_url = EXCLUDED.image_url, deleted_at = NULL;

INSERT INTO food_item_classifications (food_item_id, classification_id)
VALUES
    ('21000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001'),
    ('21000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000101'),
    ('21000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000002'),
    ('21000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000102')
ON CONFLICT DO NOTHING;

-- Implements DESIGN-005 MealEntity deterministic development fixtures.
INSERT INTO meals (
    id, type, name, physical_state, prep_time_minutes, average_unit_weight_grams,
    protein_per_100, carbohydrates_per_100, fat_per_100
)
VALUES
    ('22000000-0000-0000-0000-000000000001', 'single', 'Seed Apple Snack', 'solid', 1, 150, 0.3, 14, 0.2),
    ('22000000-0000-0000-0000-000000000002', 'composite', 'Seed Breakfast Bowl', 'solid', 5, 275, NULL, NULL, NULL)
ON CONFLICT (id) DO UPDATE
SET type = EXCLUDED.type, name = EXCLUDED.name,
    physical_state = EXCLUDED.physical_state, prep_time_minutes = EXCLUDED.prep_time_minutes,
    average_unit_weight_grams = EXCLUDED.average_unit_weight_grams,
    protein_per_100 = EXCLUDED.protein_per_100, carbohydrates_per_100 = EXCLUDED.carbohydrates_per_100,
    fat_per_100 = EXCLUDED.fat_per_100, deleted_at = NULL;

DELETE FROM recipe_ingredients WHERE meal_id = '22000000-0000-0000-0000-000000000002';

INSERT INTO recipe_ingredients (meal_id, food_item_id, quantity, unit, position)
VALUES
    ('22000000-0000-0000-0000-000000000002', '21000000-0000-0000-0000-000000000001', 150, 'g', 0),
    ('22000000-0000-0000-0000-000000000002', '21000000-0000-0000-0000-000000000002', 125, 'ml', 1);

INSERT INTO meal_classifications (meal_id, classification_id)
VALUES
    ('22000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000101'),
    ('22000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000102')
ON CONFLICT DO NOTHING;

-- Implements DESIGN-006 AuthUser deterministic development fixtures.
INSERT INTO users (
    id, email_key_version, email_nonce, email_ciphertext, normalized_email_lookup_key_version, normalized_email_digest,
    role, email_verified, password_hash, password_salt
)
VALUES
    (
        '23000000-0000-0000-0000-000000000001',
        'seed-v1',
        decode('736565642d757365722d6e6f6e6365', 'hex'),
        convert_to('seed.user@example.test', 'UTF8'),
        'seed-v1',
        'seed.user@example.test',
        'user',
        true,
        'fixture-hash-not-secret',
        'fixture-salt-not-secret'
    ),
    (
        '23000000-0000-0000-0000-000000000002',
        'seed-v1',
        decode('736565642d61646d696e2d6e6f6e6365', 'hex'),
        convert_to('seed.admin@example.test', 'UTF8'),
        'seed-v1',
        'seed.admin@example.test',
        'admin',
        true,
        'fixture-hash-not-secret',
        'fixture-salt-not-secret'
    )
ON CONFLICT (id) DO UPDATE
SET email_key_version = EXCLUDED.email_key_version,
    email_nonce = EXCLUDED.email_nonce,
    email_ciphertext = EXCLUDED.email_ciphertext,
    normalized_email_lookup_key_version = EXCLUDED.normalized_email_lookup_key_version,
    normalized_email_digest = EXCLUDED.normalized_email_digest,
    role = EXCLUDED.role,
    email_verified = EXCLUDED.email_verified,
    password_hash = EXCLUDED.password_hash, password_salt = EXCLUDED.password_salt;

-- Implements DESIGN-008 PreferenceManager deterministic development fixtures.
INSERT INTO user_profiles (user_id, display_name, unit_system, theme_preference)
VALUES ('23000000-0000-0000-0000-000000000001', 'Seed User', 'metric', 'system')
ON CONFLICT (user_id) DO UPDATE
SET display_name = EXCLUDED.display_name, unit_system = EXCLUDED.unit_system,
    theme_preference = EXCLUDED.theme_preference;

-- Implements DESIGN-007 EntitlementManager deterministic development fixtures.
INSERT INTO entitlements (user_id, tier, status, search_limit_per_24h, allowed_modes)
SELECT '23000000-0000-0000-0000-000000000001', 'free', 'active', 3, ARRAY['catalog']
WHERE NOT EXISTS (
    SELECT 1 FROM entitlements
    WHERE user_id = '23000000-0000-0000-0000-000000000001' AND tier = 'free' AND status = 'active'
);

-- Implements DESIGN-008 SavedDataRepository deterministic development fixtures.
INSERT INTO saved_items (user_id, item_id, kind)
VALUES
    ('23000000-0000-0000-0000-000000000001', '21000000-0000-0000-0000-000000000001', 'favorite'),
    ('23000000-0000-0000-0000-000000000001', '22000000-0000-0000-0000-000000000002', 'saved_meal')
ON CONFLICT (user_id, item_id, kind) DO NOTHING;

-- Implements DESIGN-008 SearchHistoryRepository deterministic development fixtures.
INSERT INTO search_history (id, user_id, query, mode, filters_hash)
VALUES ('24000000-0000-0000-0000-000000000001', '23000000-0000-0000-0000-000000000001', 'seed apple', 'food', 'seed-filter')
ON CONFLICT (id) DO UPDATE
SET query = EXCLUDED.query, mode = EXCLUDED.mode, filters_hash = EXCLUDED.filters_hash;

-- Implements DESIGN-015 ConsentManager deterministic development fixtures.
INSERT INTO consent_records (user_id, privacy_policy_version, terms_version)
VALUES ('23000000-0000-0000-0000-000000000001', 'seed-privacy-v1', 'seed-terms-v1')
ON CONFLICT (user_id, privacy_policy_version, terms_version) DO NOTHING;

-- Implements DESIGN-009 DataImporter deterministic development fixtures.
INSERT INTO curated_imports (id, source_provider, external_id, food_item_id, status, raw_payload)
VALUES ('25000000-0000-0000-0000-000000000001', 'seed-provider', 'seed-external-1', '21000000-0000-0000-0000-000000000001', 'imported', '{"fixture":true}'::jsonb)
ON CONFLICT (source_provider, external_id) DO UPDATE
SET food_item_id = EXCLUDED.food_item_id, status = EXCLUDED.status, raw_payload = EXCLUDED.raw_payload;

-- Implements DESIGN-009 AdminController deterministic development fixtures.
INSERT INTO admin_audit_entries (id, admin_user_id, action, entity_type, entity_id, before_snapshot, after_snapshot, request_id)
VALUES ('26000000-0000-0000-0000-000000000001', '23000000-0000-0000-0000-000000000002', 'seed_import', 'food_item', '21000000-0000-0000-0000-000000000001', '{}'::jsonb, '{"fixture":true}'::jsonb, 'seed-request')
ON CONFLICT (id) DO NOTHING;
