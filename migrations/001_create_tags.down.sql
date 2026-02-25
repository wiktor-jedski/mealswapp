-- Phase: phase-01 | Task: 3 | Architecture: ARCH-005 | Design: TagEntity

-- Drop indexes for tags table
DROP INDEX IF EXISTS idx_tags_type_name;
DROP INDEX IF EXISTS idx_tags_name;
DROP INDEX IF EXISTS idx_tags_type;
DROP INDEX IF EXISTS idx_tags_slug;

-- Drop tags table
DROP TABLE IF EXISTS tags;
