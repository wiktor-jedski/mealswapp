-- Implements DESIGN-008 SavedDataRepository immutable daily-diet create snapshot.
INSERT INTO saved_diets (id, user_id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);
