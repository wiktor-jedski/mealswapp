-- Implements DESIGN-005 MicronutrientVocabulary integration fixture.
INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
VALUES ('Legacy', 'Legacy', 'mg', false)
ON CONFLICT (key) DO UPDATE SET active = false;
