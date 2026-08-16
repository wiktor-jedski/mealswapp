-- Implements DESIGN-009 AdminController serialized first-administrator bootstrap.
SELECT pg_advisory_xact_lock(2760802);
