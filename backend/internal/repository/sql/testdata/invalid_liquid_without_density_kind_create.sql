-- Implements DESIGN-005 FoodItemEntity integration fixture.
INSERT INTO food_items (name, physical_state, density_grams_per_milliliter, protein_per_100, carbohydrates_per_100, fat_per_100)
VALUES ('Invalid Liquid Without Density Kind', 'liquid', 1, 0, 0, 0);
