-- Implements DESIGN-009 TagManager active allergen filter-option vocabulary.
SELECT key, name, label_key
FROM allergen_vocabulary
WHERE deleted_at IS NULL
ORDER BY lower(name), key;
