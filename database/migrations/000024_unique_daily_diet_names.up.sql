-- Implements DESIGN-008 SavedDataRepository user-scoped Daily Diet naming.
CREATE UNIQUE INDEX IF NOT EXISTS saved_diets_user_normalized_name_uidx
    ON saved_diets (user_id, lower(btrim(name)));

INSERT INTO schema_migrations (version)
VALUES (24)
ON CONFLICT (version) DO NOTHING;
