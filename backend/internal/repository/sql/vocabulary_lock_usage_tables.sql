-- Implements DESIGN-005 MicronutrientVocabulary concurrent item-write safeguard.
LOCK TABLE food_items, custom_food_items IN SHARE MODE;
