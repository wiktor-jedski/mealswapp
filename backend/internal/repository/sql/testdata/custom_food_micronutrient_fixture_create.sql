-- Implements DESIGN-005 MicronutrientVocabulary private-use integration fixture.
INSERT INTO custom_food_items (
    owner_id, name, physical_state, protein_per_100, carbohydrates_per_100, fat_per_100, micronutrients
)
VALUES ($1, 'Private vocabulary fixture', 'solid', 1, 1, 1, jsonb_build_object($2::text, 1));
