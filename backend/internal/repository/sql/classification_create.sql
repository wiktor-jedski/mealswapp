-- Implements DESIGN-009 TagManager classification create query.
INSERT INTO classifications (name, kind, parent_id)
VALUES ($1, $2, $3)
RETURNING id, name, kind, parent_id;
