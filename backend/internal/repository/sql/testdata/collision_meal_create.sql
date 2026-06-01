-- Implements DESIGN-005 MealEntity integration fixture.
INSERT INTO meals (id, type, name, physical_state)
VALUES ($1, 'composite', 'Cycle Recipe', 'solid');
