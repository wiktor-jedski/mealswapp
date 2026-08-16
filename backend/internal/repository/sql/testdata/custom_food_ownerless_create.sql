-- Implements DESIGN-005 FoodItemEntity ownerless-row rejection fixture.
INSERT INTO custom_food_items (
    name, physical_state, protein_per_100, carbohydrates_per_100, fat_per_100
)
VALUES ('Ownerless', 'solid', 0, 0, 0);
