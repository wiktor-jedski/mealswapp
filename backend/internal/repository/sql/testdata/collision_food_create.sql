-- Implements DESIGN-005 FoodItemEntity integration fixture.
INSERT INTO food_items (id, name, physical_state, protein_per_100, carbohydrates_per_100, fat_per_100)
VALUES ($1, 'Cycle Food', 'solid', 1, 1, 1);
