-- Implements DESIGN-008 SavedDataRepository.
-- Implements DESIGN-008 PreferenceManager.
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name text,
    unit_system text NOT NULL DEFAULT 'metric' CHECK (unit_system IN ('metric', 'imperial')),
    theme_preference text NOT NULL DEFAULT 'system' CHECK (theme_preference IN ('system', 'light', 'dark')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS saved_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id uuid NOT NULL,
    kind text NOT NULL CHECK (kind IN ('favorite', 'saved_meal', 'saved_diet')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, item_id, kind)
);

CREATE INDEX IF NOT EXISTS saved_items_user_kind_idx
    ON saved_items (user_id, kind, created_at DESC);

CREATE TABLE IF NOT EXISTS search_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    query text NOT NULL,
    mode text NOT NULL,
    filters_hash text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT search_history_query_not_blank CHECK (btrim(query) <> ''),
    CONSTRAINT search_history_mode_not_blank CHECK (btrim(mode) <> '')
);

CREATE INDEX IF NOT EXISTS search_history_user_created_idx
    ON search_history (user_id, created_at DESC);

INSERT INTO schema_migrations (version)
VALUES (7)
ON CONFLICT (version) DO NOTHING;
