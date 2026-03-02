-- Phase: phase-01 | Task: 20 | Architecture: ARCH-005 | Design: MealEntity
-- Create meals table with constraints and indexes

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE meals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('single', 'recipe')),
    physical_state VARCHAR(20) NOT NULL CHECK (physical_state IN ('solid', 'liquid')),
    prep_time INTEGER NOT NULL DEFAULT 0,
    average_unit_weight DECIMAL(10, 2) NOT NULL DEFAULT 0,
    macros JSONB NOT NULL DEFAULT '{"protein": 0, "carbs": 0, "fat": 0}',
    micros JSONB NOT NULL DEFAULT '{}',
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_meals_name_unique ON meals(name);
CREATE INDEX idx_meals_type ON meals(type);
CREATE INDEX idx_meals_physical_state ON meals(physical_state);
CREATE INDEX idx_meals_macros ON meals USING GIN (macros);
CREATE INDEX idx_meals_created_at ON meals(created_at DESC);

CREATE TRIGGER update_meals_updated_at
    BEFORE UPDATE ON meals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
