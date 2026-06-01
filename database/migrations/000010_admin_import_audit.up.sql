-- Implements DESIGN-009 DataImporter.
-- Implements DESIGN-009 AdminController.
CREATE TABLE IF NOT EXISTS curated_imports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_provider text NOT NULL,
    external_id text NOT NULL,
    food_item_id uuid REFERENCES food_items(id) ON DELETE SET NULL,
    status text NOT NULL CHECK (status IN ('draft', 'imported', 'conflict', 'rejected')),
    conflict_reason text,
    raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT curated_imports_provider_not_blank CHECK (btrim(source_provider) <> ''),
    CONSTRAINT curated_imports_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT curated_imports_raw_payload_object CHECK (jsonb_typeof(raw_payload) = 'object'),
    UNIQUE (source_provider, external_id)
);

CREATE INDEX IF NOT EXISTS curated_imports_food_item_idx
    ON curated_imports (food_item_id)
    WHERE food_item_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS curated_imports_status_idx
    ON curated_imports (status, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_audit_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid,
    before_snapshot jsonb,
    after_snapshot jsonb,
    request_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT admin_audit_action_not_blank CHECK (btrim(action) <> ''),
    CONSTRAINT admin_audit_entity_type_not_blank CHECK (btrim(entity_type) <> ''),
    CONSTRAINT admin_audit_before_object CHECK (before_snapshot IS NULL OR jsonb_typeof(before_snapshot) = 'object'),
    CONSTRAINT admin_audit_after_object CHECK (after_snapshot IS NULL OR jsonb_typeof(after_snapshot) = 'object')
);

CREATE INDEX IF NOT EXISTS admin_audit_admin_created_idx
    ON admin_audit_entries (admin_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS admin_audit_entity_idx
    ON admin_audit_entries (entity_type, entity_id, created_at DESC)
    WHERE entity_id IS NOT NULL;

INSERT INTO schema_migrations (version)
VALUES (10)
ON CONFLICT (version) DO NOTHING;
