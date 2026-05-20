package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"
	"mealswapp/backend/internal/domain/micronutrient"
	"mealswapp/backend/internal/domain/recipe"
	"mealswapp/backend/internal/domain/tag"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryInterfacesCRUDAndTransactionRollback(t *testing.T) {
	databaseURL := os.Getenv("MEALSWAPP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MEALSWAPP_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	resetRepositoryFoundation(t, ctx, pool)
	defer resetRepositoryFoundation(t, ctx, pool)

	store := NewStore(pool)
	repos := store.Repositories()

	userID := exerciseUserRepository(t, ctx, repos.Users)
	exerciseConsentRepository(t, ctx, repos.Consents, userID)
	exercisePreferenceRepository(t, ctx, repos.Preferences, userID)
	exerciseEntitlementRepository(t, ctx, repos.Entitlements, userID)
	foodID := exerciseFoodItemRepository(t, ctx, repos.FoodItems)
	exerciseMealRepository(t, ctx, repos.Meals, userID, foodID)
	exerciseRecipeRepository(t, ctx, repos.Recipes, userID, foodID)
	exerciseTagRepository(t, ctx, repos.Tags, foodID)
	exerciseMicronutrientRepository(t, ctx, repos.MicronutrientVocabulary)
	exerciseSavedDataRepository(t, ctx, repos.SavedData, userID)
	exerciseAuditLogRepository(t, ctx, repos.AuditLogs, userID)
	exerciseImportRepository(t, ctx, repos.Imports)

	verifyRepositoryRollback(t, ctx, store, pool)
}

func resetRepositoryFoundation(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for i := 9; i >= 1; i-- {
		applyMigration(t, ctx, pool, fmt.Sprintf("%04d_%s.down.sql", i, migrationNames[i]))
	}
	for i := 1; i <= 9; i++ {
		applyMigration(t, ctx, pool, fmt.Sprintf("%04d_%s.up.sql", i, migrationNames[i]))
	}
}

var migrationNames = map[int]string{
	1: "food_items",
	2: "meals",
	3: "recipes",
	4: "tags",
	5: "micronutrient_vocabulary",
	6: "repository_foundation",
	7: "consent_records",
	8: "oauth_identities",
	9: "account_tokens",
}

func exerciseUserRepository(t *testing.T, ctx context.Context, repo repositories.UserRepository) uuid.UUID {
	t.Helper()

	id, err := repo.Create(ctx, repositories.UserEntity{Email: "user@example.com", DisplayName: "User", PasswordHash: "hash", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	user.DisplayName = "Updated User"
	if err := repo.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	user, err = repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if user.DisplayName != "Updated User" {
		t.Fatalf("expected updated user display name, got %q", user.DisplayName)
	}

	return id
}

func exerciseConsentRepository(t *testing.T, ctx context.Context, repo ConsentRepository, userID uuid.UUID) {
	t.Helper()

	_, err := repo.Record(ctx, repositories.ConsentRecordEntity{
		UserID:                     userID,
		PrivacyPolicyVersion:       "privacy-v1",
		TermsVersion:               "terms-v1",
		NutritionDisclaimerVersion: "nutrition-v1",
		IPAddress:                  "203.0.113.10",
		UserAgent:                  "integration-test",
	})
	if err != nil {
		t.Fatal(err)
	}

	hasConsent, err := repo.HasRequiredConsent(ctx, userID, "privacy-v1", "terms-v1", "nutrition-v1")
	if err != nil {
		t.Fatal(err)
	}
	if !hasConsent {
		t.Fatal("expected matching consent versions")
	}
}

func exercisePreferenceRepository(t *testing.T, ctx context.Context, repo repositories.PreferenceRepository, userID uuid.UUID) {
	t.Helper()

	preference := repositories.PreferenceEntity{
		UserID:            userID,
		Theme:             "dark",
		DefaultSearchMode: "replacement",
		EnabledMacros:     map[string]bool{"protein": true, "carbs": false, "fat": true},
	}
	if err := repo.Upsert(ctx, preference); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "dark" || got.EnabledMacros["carbs"] {
		t.Fatalf("unexpected preference: %#v", got)
	}
}

func exerciseEntitlementRepository(t *testing.T, ctx context.Context, repo repositories.EntitlementRepository, userID uuid.UUID) {
	t.Helper()

	if err := repo.Upsert(ctx, repositories.EntitlementEntity{UserID: userID, Plan: "trial", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan != "trial" || got.Status != "active" {
		t.Fatalf("unexpected entitlement: %#v", got)
	}
}

func exerciseFoodItemRepository(t *testing.T, ctx context.Context, repo repositories.FoodItemRepository) uuid.UUID {
	t.Helper()

	id, err := repo.Create(ctx, validRepositoryFood("Repository oats"))
	if err != nil {
		t.Fatal(err)
	}
	item, err := repo.GetByID(ctx, id, repositories.RepositoryContext{})
	if err != nil {
		t.Fatal(err)
	}
	item.Name = "Repository oats updated"
	if err := repo.Update(ctx, item); err != nil {
		t.Fatal(err)
	}
	results, total, err := repo.Search(ctx, repositories.FoodItemQuery{Text: "oats", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 || len(results) == 0 {
		t.Fatal("expected food item search results")
	}
	return id
}

func exerciseMealRepository(t *testing.T, ctx context.Context, repo repositories.MealRepository, userID uuid.UUID, foodID uuid.UUID) {
	t.Helper()

	id, err := repo.Create(ctx, meal.MealEntity{
		UserID: userID,
		Name:   "Repository meal",
		Type:   meal.MealTypeSingle,
		Items:  []meal.MealItemEntity{{FoodItemID: foodID, Quantity: 100, Unit: meal.IngredientUnitGram}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	got.Name = "Repository meal updated"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func exerciseRecipeRepository(t *testing.T, ctx context.Context, repo repositories.RecipeRepository, userID uuid.UUID, foodID uuid.UUID) {
	t.Helper()

	id, err := repo.Create(ctx, recipe.RecipeEntity{
		UserID:      userID,
		Name:        "Repository recipe",
		Ingredients: []recipe.RecipeIngredientEntity{{FoodItemID: foodID, Quantity: 100, Unit: meal.IngredientUnitGram}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.CaloriesTotal == 0 {
		t.Fatal("expected recipe aggregate calories")
	}
	got.Name = "Repository recipe updated"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func exerciseTagRepository(t *testing.T, ctx context.Context, repo repositories.TagRepository, foodID uuid.UUID) {
	t.Helper()

	tagID, err := repo.Upsert(ctx, tag.TagEntity{Name: "Repository vegan", Kind: tag.KindDiet, Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AttachToFoodItem(ctx, foodID, tagID); err != nil {
		t.Fatal(err)
	}
	ids, err := repo.QueryFoodItemIDs(ctx, repositories.FoodItemTagFilter{IncludeTagIDs: []uuid.UUID{tagID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != foodID {
		t.Fatalf("unexpected tag filter result: %#v", ids)
	}
	if err := repo.RemoveFromFoodItem(ctx, foodID, tagID); err != nil {
		t.Fatal(err)
	}
}

func exerciseMicronutrientRepository(t *testing.T, ctx context.Context, repo repositories.MicronutrientVocabularyRepository) {
	t.Helper()

	if err := repo.Upsert(ctx, micronutrient.Entry{Key: "RepositoryZinc", DisplayName: "Repository Zinc", Unit: micronutrient.UnitMilligram, Active: true}); err != nil {
		t.Fatal(err)
	}
	allowed, err := repo.IsAllowed(ctx, "RepositoryZinc")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected RepositoryZinc to be allowed")
	}
}

func exerciseSavedDataRepository(t *testing.T, ctx context.Context, repo repositories.SavedDataRepository, userID uuid.UUID) {
	t.Helper()

	id, err := repo.Create(ctx, repositories.SavedDataEntity{UserID: userID, Kind: "favorite", Label: "Favorite", Payload: []byte(`{"food":"oats"}`)})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	saved.Label = "Updated favorite"
	if err := repo.Update(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func exerciseAuditLogRepository(t *testing.T, ctx context.Context, repo repositories.AuditLogRepository, userID uuid.UUID) {
	t.Helper()

	id, err := repo.Create(ctx, repositories.AuditLogEntity{ActorID: &userID, Action: "repository.test", Target: "test", Metadata: []byte(`{"ok":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func exerciseImportRepository(t *testing.T, ctx context.Context, repo repositories.ImportRepository) {
	t.Helper()

	id, err := repo.Create(ctx, repositories.ImportEntity{Provider: "repository", ExternalID: uuid.NewString(), Status: "draft", Payload: []byte(`{"raw":true}`)})
	if err != nil {
		t.Fatal(err)
	}
	record, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	record.Status = "imported"
	if err := repo.Update(ctx, record); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func verifyRepositoryRollback(t *testing.T, ctx context.Context, store Store, pool *pgxpool.Pool) {
	t.Helper()

	rollbackErr := errors.New("force rollback")
	var rolledBackUserID uuid.UUID
	var rolledBackFoodID uuid.UUID
	var rolledBackTagID uuid.UUID
	var rolledBackImportID uuid.UUID
	err := store.WithTx(ctx, func(repos RepositorySet) error {
		var err error
		rolledBackUserID, err = repos.Users.Create(ctx, repositories.UserEntity{Email: "rollback@example.com", Role: "user"})
		if err != nil {
			return err
		}
		if err := repos.Preferences.Upsert(ctx, repositories.PreferenceEntity{UserID: rolledBackUserID, Theme: "light", DefaultSearchMode: "single", EnabledMacros: map[string]bool{"protein": true}}); err != nil {
			return err
		}
		if err := repos.Entitlements.Upsert(ctx, repositories.EntitlementEntity{UserID: rolledBackUserID, Plan: "free", Status: "active"}); err != nil {
			return err
		}
		rolledBackFoodID, err = repos.FoodItems.Create(ctx, validRepositoryFood("Rollback food"))
		if err != nil {
			return err
		}
		if _, err := repos.Meals.Create(ctx, meal.MealEntity{UserID: rolledBackUserID, Name: "Rollback meal", Type: meal.MealTypeSingle, Items: []meal.MealItemEntity{{FoodItemID: rolledBackFoodID, Quantity: 100, Unit: meal.IngredientUnitGram}}}); err != nil {
			return err
		}
		if _, err := repos.Recipes.Create(ctx, recipe.RecipeEntity{UserID: rolledBackUserID, Name: "Rollback recipe", Ingredients: []recipe.RecipeIngredientEntity{{FoodItemID: rolledBackFoodID, Quantity: 100, Unit: meal.IngredientUnitGram}}}); err != nil {
			return err
		}
		rolledBackTagID, err = repos.Tags.Upsert(ctx, tag.TagEntity{Name: "Rollback tag", Kind: tag.KindCuration, Active: true})
		if err != nil {
			return err
		}
		if err := repos.Tags.AttachToFoodItem(ctx, rolledBackFoodID, rolledBackTagID); err != nil {
			return err
		}
		if err := repos.MicronutrientVocabulary.Upsert(ctx, micronutrient.Entry{Key: "RollbackNutrient", DisplayName: "Rollback Nutrient", Unit: micronutrient.UnitMilligram, Active: true}); err != nil {
			return err
		}
		if _, err := repos.SavedData.Create(ctx, repositories.SavedDataEntity{UserID: rolledBackUserID, Kind: "saved_search", Label: "Rollback saved", Payload: []byte(`{"q":"x"}`)}); err != nil {
			return err
		}
		if _, err := repos.AuditLogs.Create(ctx, repositories.AuditLogEntity{ActorID: &rolledBackUserID, Action: "rollback", Target: "test", Metadata: []byte(`{}`)}); err != nil {
			return err
		}
		rolledBackImportID, err = repos.Imports.Create(ctx, repositories.ImportEntity{Provider: "rollback", ExternalID: uuid.NewString(), Status: "draft", Payload: []byte(`{}`)})
		if err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}

	assertMissing(t, ctx, pool, `SELECT 1 FROM users WHERE id = $1`, rolledBackUserID)
	assertMissing(t, ctx, pool, `SELECT 1 FROM food_items WHERE id = $1`, rolledBackFoodID)
	assertMissing(t, ctx, pool, `SELECT 1 FROM tags WHERE id = $1`, rolledBackTagID)
	assertMissing(t, ctx, pool, `SELECT 1 FROM micronutrient_vocabulary WHERE key = $1`, "RollbackNutrient")
	assertMissing(t, ctx, pool, `SELECT 1 FROM import_records WHERE id = $1`, rolledBackImportID)
}

func validRepositoryFood(name string) food.FoodItemEntity {
	return food.FoodItemEntity{
		Name:           name,
		PhysicalState:  food.PhysicalStateSolid,
		ServingUnit:    food.ServingUnitGram,
		ServingSize:    100,
		CaloriesPer100: 389,
		MacrosPer100: food.MacroValues{
			ProteinGrams: 16.9,
			CarbsGrams:   66.3,
			FatGrams:     6.9,
		},
		Micros:                 map[string]float64{"Sodium": 2},
		AverageUnitWeightGrams: 100,
	}
}

func assertMissing(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	var one int
	err := pool.QueryRow(ctx, query, args...).Scan(&one)
	if err != pgx.ErrNoRows {
		t.Fatalf("expected row to be rolled back for query %q, got %v", query, err)
	}
}
