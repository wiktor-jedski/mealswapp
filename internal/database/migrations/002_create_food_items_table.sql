-- Phase: phase-01 | Task: 4 | Architecture: ARCH-005 | Design: FoodItemEntity
-- Create food_items table with constraints and indexes

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE food_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    physical_state VARCHAR(20) NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time INTEGER NOT NULL DEFAULT 0,
    average_unit_weight DECIMAL(10, 2) NOT NULL DEFAULT 0,
    macros JSONB NOT NULL DEFAULT '{"protein": 0, "carbs": 0, "fat": 0}',
    micros JSONB NOT NULL DEFAULT '{}',
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_food_items_name_unique ON food_items(name);
CREATE INDEX idx_food_items_physical_state ON food_items(physical_state);
CREATE INDEX idx_food_items_macros ON food_items USING GIN (macros);
CREATE INDEX idx_food_items_created_at ON food_items(created_at DESC);

CREATE TRIGGER update_food_items_updated_at
    BEFORE UPDATE ON food_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
