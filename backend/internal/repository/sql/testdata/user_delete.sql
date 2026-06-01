-- Implements DESIGN-008 SavedDataRepository integration fixture.
DELETE FROM users WHERE id = $1;
