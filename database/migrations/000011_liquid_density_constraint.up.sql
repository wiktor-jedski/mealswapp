-- Implements DESIGN-005 FoodItemEntity liquid density invariant.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'food_items_liquid_density_required'
          AND conrelid = 'food_items'::regclass
    ) THEN
        ALTER TABLE food_items
            ADD CONSTRAINT food_items_liquid_density_required CHECK (
                (
                    physical_state = 'liquid'
                    AND density_grams_per_milliliter IS NOT NULL
                    AND density_grams_per_milliliter > 0
                    AND density_source_kind IS NOT NULL
                    AND density_source_kind IN ('imported', 'manual', 'estimated')
                )
                OR (
                    physical_state = 'solid'
                    AND average_serving_volume_milliliters IS NULL
                    AND density_grams_per_milliliter IS NULL
                    AND density_source_provider IS NULL
                    AND density_source_food_id IS NULL
                    AND density_source_kind IS NULL
                )
            );
    END IF;
END
$$;

INSERT INTO schema_migrations (version)
VALUES (11)
ON CONFLICT (version) DO NOTHING;
