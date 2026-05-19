package seed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	AdminUserID      = "00000000-0000-4000-8000-000000000001"
	OatsFoodID       = "00000000-0000-4000-8000-000000000101"
	MilkFoodID       = "00000000-0000-4000-8000-000000000102"
	TofuFoodID       = "00000000-0000-4000-8000-000000000103"
	VeganTagID       = "00000000-0000-4000-8000-000000000201"
	DairyTagID       = "00000000-0000-4000-8000-000000000202"
	HighProteinTagID = "00000000-0000-4000-8000-000000000203"
	PorridgeRecipeID = "00000000-0000-4000-8000-000000000301"
)

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	statements := []string{
		`
		INSERT INTO micronutrient_vocabulary (key, display_name, unit, active)
		VALUES
			('Calcium', 'Calcium', 'mg', true),
			('Iron', 'Iron', 'mg', true),
			('Sodium', 'Sodium', 'mg', true),
			('VitaminC', 'Vitamin C', 'mg', true)
		ON CONFLICT (key) DO UPDATE
		SET display_name = excluded.display_name, unit = excluded.unit, active = excluded.active, updated_at = now()
		`,
		`
		INSERT INTO tags (id, name, kind, active)
		VALUES
			('` + VeganTagID + `', 'Vegan', 'diet', true),
			('` + DairyTagID + `', 'Dairy', 'allergen', true),
			('` + HighProteinTagID + `', 'High protein', 'functionality', true)
		ON CONFLICT (kind, normalized_name) DO UPDATE
		SET active = excluded.active, updated_at = now()
		`,
		`
		INSERT INTO food_items (
			id, name, physical_state, serving_unit, serving_size, calories_per_100,
			protein_grams_per_100, carbs_grams_per_100, fat_grams_per_100, micronutrients,
			curation_state, average_unit_weight_grams
		)
		VALUES
			('` + OatsFoodID + `', 'Rolled oats', 'solid', 'gram', 100, 389, 16.9, 66.3, 6.9, '{"Iron": 4.7}'::jsonb, 'approved', 100),
			('` + MilkFoodID + `', 'Whole milk', 'liquid', 'milliliter', 100, 61, 3.2, 4.8, 3.3, '{"Calcium": 113}'::jsonb, 'approved', 100),
			('` + TofuFoodID + `', 'Firm tofu', 'solid', 'gram', 100, 144, 17.3, 2.8, 8.7, '{"Calcium": 350}'::jsonb, 'approved', 100)
		ON CONFLICT (id) DO UPDATE
		SET name = excluded.name,
			physical_state = excluded.physical_state,
			serving_unit = excluded.serving_unit,
			serving_size = excluded.serving_size,
			calories_per_100 = excluded.calories_per_100,
			protein_grams_per_100 = excluded.protein_grams_per_100,
			carbs_grams_per_100 = excluded.carbs_grams_per_100,
			fat_grams_per_100 = excluded.fat_grams_per_100,
			micronutrients = excluded.micronutrients,
			curation_state = excluded.curation_state,
			average_unit_weight_grams = excluded.average_unit_weight_grams,
			updated_at = now()
		`,
		`
		INSERT INTO food_item_tags (food_item_id, tag_id)
		VALUES
			('` + OatsFoodID + `', '` + VeganTagID + `'),
			('` + TofuFoodID + `', '` + VeganTagID + `'),
			('` + TofuFoodID + `', '` + HighProteinTagID + `'),
			('` + MilkFoodID + `', '` + DairyTagID + `')
		ON CONFLICT DO NOTHING
		`,
		`
		INSERT INTO users (id, email, display_name, password_hash, role)
		VALUES ('` + AdminUserID + `', 'admin@mealswapp.local', 'Admin', 'dev-only', 'admin')
		ON CONFLICT (id) DO UPDATE
		SET email = excluded.email, display_name = excluded.display_name, role = excluded.role, updated_at = now()
		`,
		`
		INSERT INTO recipes (
			id, user_id, name, calories_total, protein_grams_total, carbs_grams_total, fat_grams_total, source_provider, source_external_id
		)
		VALUES ('` + PorridgeRecipeID + `', '` + AdminUserID + `', 'Seed porridge', 433.1, 19.92, 62.64, 12.12, 'seed', 'porridge')
		ON CONFLICT (id) DO UPDATE
		SET name = excluded.name,
			calories_total = excluded.calories_total,
			protein_grams_total = excluded.protein_grams_total,
			carbs_grams_total = excluded.carbs_grams_total,
			fat_grams_total = excluded.fat_grams_total,
			updated_at = now()
		`,
		`
		DELETE FROM recipe_ingredients WHERE recipe_id = '` + PorridgeRecipeID + `'
		`,
		`
		INSERT INTO recipe_ingredients (recipe_id, food_item_id, quantity, unit, position)
		VALUES
			('` + PorridgeRecipeID + `', '` + OatsFoodID + `', 80, 'gram', 0),
			('` + PorridgeRecipeID + `', '` + MilkFoodID + `', 200, 'milliliter', 1)
		`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
