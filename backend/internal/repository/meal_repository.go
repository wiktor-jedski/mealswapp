package repository

import (
	"context"
	_ "embed"
	"strings"

	"github.com/google/uuid"
)

// Implements DESIGN-005 MealEntity search query.
//
//go:embed sql/meal_search.sql
var mealSearchSQL string

// Implements DESIGN-005 MealEntity create query.
//
//go:embed sql/meal_create.sql
var mealCreateSQL string

// Implements DESIGN-005 MealEntity update query.
//
//go:embed sql/meal_update.sql
var mealUpdateSQL string

// Implements DESIGN-005 MealEntity soft-delete query.
//
//go:embed sql/meal_soft_delete.sql
var mealSoftDeleteSQL string

// Implements DESIGN-005 MealEntity tag validation query.
//
//go:embed sql/meal_validate_tag.sql
var mealValidateTagSQL string

// Implements DESIGN-005 MealEntity clear-ingredients query.
//
//go:embed sql/meal_clear_ingredients.sql
var mealClearIngredientsSQL string

// Implements DESIGN-005 MealEntity attach-ingredient query.
//
//go:embed sql/meal_attach_ingredient.sql
var mealAttachIngredientSQL string

// Implements DESIGN-005 MealEntity clear-tags query.
//
//go:embed sql/meal_clear_tags.sql
var mealClearTagsSQL string

// Implements DESIGN-005 MealEntity attach-tag query.
//
//go:embed sql/meal_attach_tag.sql
var mealAttachTagSQL string

// Implements DESIGN-005 MealEntity get-by-id query.
//
//go:embed sql/meal_get_by_id.sql
var mealGetByIDSQL string

// Implements DESIGN-005 MealEntity ingredient query.
//
//go:embed sql/meal_list_ingredients.sql
var mealListIngredientsSQL string

// Implements DESIGN-005 MealEntity hydrate-tags query.
//
//go:embed sql/meal_list_tags.sql
var mealListTagsSQL string

// PostgresMealRepository persists opaque single and composite meals in PostgreSQL.
// Implements DESIGN-005 MealEntity.
type PostgresMealRepository struct {
	db transactionalExecutor
}

// NewPostgresMealRepository creates a PostgreSQL-backed meal repository.
// Implements DESIGN-005 MealEntity.
func NewPostgresMealRepository(db transactionalExecutor) *PostgresMealRepository {
	return &PostgresMealRepository{db: db}
}

// GetByID loads an opaque single or composite meal with ingredients and hydrated tags.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) GetByID(ctx context.Context, id uuid.UUID, rc RepositoryContext) (MealEntity, error) {
	meal, err := r.getMealByID(ctx, id, rc.IncludeDeleted)
	if err != nil {
		return MealEntity{}, err
	}
	if meal.Type == MealTypeComposite {
		ingredients, err := r.loadIngredients(ctx, meal.ID)
		if err != nil {
			return MealEntity{}, err
		}
		meal.RecipeItems = ingredients
		macros, available, err := r.calculateCompositeMacros(ctx, ingredients)
		if err != nil {
			return MealEntity{}, err
		}
		meal.MacrosPer100 = macros
		meal.NormalizedMacrosAvailable = available
	} else {
		meal.NormalizedMacrosAvailable = true
	}
	if err := r.hydrateMealTags(ctx, &meal); err != nil {
		return MealEntity{}, err
	}
	convertMealForUnitSystem(&meal, rc.UnitSystem)
	return meal, nil
}

// Search returns matching meals and total count for deterministic pagination.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) Search(ctx context.Context, q RepositoryQuery) ([]MealEntity, int, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.Query(ctx, mealSearchSQL, q.IncludeDeleted, q.Name, q.MaxPrepMinutes, q.CategoryTagIDs, q.FunctionalityIDs)
	if err != nil {
		return nil, 0, mapPostgresError(err, "search meals")
	}
	defer rows.Close()

	matches := []MealEntity{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, mapPostgresError(err, "scan meal search id")
		}
		meal, err := r.GetByID(ctx, id, q.RepositoryContext)
		if err != nil {
			return nil, 0, err
		}
		matches = append(matches, meal)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err, "iterate meal search")
	}

	total := len(matches)
	if offset >= total {
		return []MealEntity{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matches[offset:end], total, nil
}

// CalculateMacros returns aggregate macro values for a meal.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) CalculateMacros(ctx context.Context, mealID uuid.UUID) (MacroValues, error) {
	return r.calculateMacros(ctx, mealID, RepositoryContext{UnitSystem: UnitSystemMetric})
}

// calculateMacros derives meal macros from persisted ingredients.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) calculateMacros(ctx context.Context, mealID uuid.UUID, rc RepositoryContext) (MacroValues, error) {
	rc.UnitSystem = UnitSystemMetric
	meal, err := r.GetByID(ctx, mealID, rc)
	if err != nil {
		return MacroValues{}, err
	}
	switch meal.Type {
	case MealTypeSingle:
		return meal.MacrosPer100, nil
	case MealTypeComposite:
		return meal.MacrosPer100, nil
	default:
		return MacroValues{}, validationError("unsupported meal type")
	}
}

// calculateCompositeMacros derives normalized per-100 macros from persisted ingredients.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) calculateCompositeMacros(ctx context.Context, ingredients []RecipeIngredientEntity) (MacroValues, bool, error) {
	total := MacroValues{}
	totalMassGrams := 0.0
	for _, ingredient := range ingredients {
		food, err := NewPostgresFoodItemRepository(r.db).GetByID(ctx, ingredient.FoodItemID, RepositoryContext{UnitSystem: UnitSystemMetric})
		if err != nil {
			return MacroValues{}, false, err
		}
		basis, err := ingredientBasisQuantity(ingredient, food)
		if err != nil {
			return MacroValues{}, false, err
		}
		scaled := ScaleMacros(food.MacrosPer100, basis, 100)
		massGrams, available := ingredientMassGrams(basis, food)
		if !available {
			return MacroValues{}, false, nil
		}
		totalMassGrams += massGrams
		total.Protein += scaled.Protein
		total.Carbohydrates += scaled.Carbohydrates
		total.Fat += scaled.Fat
	}
	if totalMassGrams == 0 {
		return MacroValues{}, false, validationError("composite meal requires positive ingredient basis")
	}
	return ScaleMacros(total, 100, totalMassGrams), true, nil
}

// Create validates and persists a meal with tags and recipe ingredients.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) Create(ctx context.Context, meal MealEntity) (uuid.UUID, error) {
	if err := r.validateMeal(ctx, meal); err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	err := withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		txRepo := NewPostgresMealRepository(db)
		err := db.QueryRow(ctx, mealCreateSQL, string(meal.Type), strings.TrimSpace(meal.Name), string(meal.PhysicalState), meal.PrepTimeMinutes, nullablePositiveFloat(meal.AverageUnitWeightGrams), nullableMealMacro(meal, meal.MacrosPer100.Protein), nullableMealMacro(meal, meal.MacrosPer100.Carbohydrates), nullableMealMacro(meal, meal.MacrosPer100.Fat)).Scan(&id)
		if err != nil {
			return mapPostgresError(err, "create meal")
		}
		if err := txRepo.replaceIngredients(ctx, id, meal.RecipeItems); err != nil {
			return err
		}
		return txRepo.replaceMealTags(ctx, id, meal.Tags)
	})
	return id, err
}

// Update validates and replaces a meal with tags and recipe ingredients.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) Update(ctx context.Context, meal MealEntity) error {
	if meal.ID == uuid.Nil {
		return validationError("meal id is required")
	}
	if err := r.validateMeal(ctx, meal); err != nil {
		return err
	}
	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		txRepo := NewPostgresMealRepository(db)
		result, err := db.Exec(ctx, mealUpdateSQL, meal.ID, string(meal.Type), strings.TrimSpace(meal.Name), string(meal.PhysicalState), meal.PrepTimeMinutes, nullablePositiveFloat(meal.AverageUnitWeightGrams), nullableMealMacro(meal, meal.MacrosPer100.Protein), nullableMealMacro(meal, meal.MacrosPer100.Carbohydrates), nullableMealMacro(meal, meal.MacrosPer100.Fat))
		if err != nil {
			return mapPostgresError(err, "update meal")
		}
		if result.RowsAffected() == 0 {
			return NewError(ErrorKindNotFound, "meal not found", nil)
		}
		if err := txRepo.replaceIngredients(ctx, meal.ID, meal.RecipeItems); err != nil {
			return err
		}
		return txRepo.replaceMealTags(ctx, meal.ID, meal.Tags)
	})
}

// Delete soft-deletes a meal.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, mealSoftDeleteSQL, id)
	if err != nil {
		return mapPostgresError(err, "delete meal")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "meal not found", nil)
	}
	return nil
}

// validateMeal checks meal fields before persistence.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) validateMeal(ctx context.Context, meal MealEntity) error {
	if meal.PrepTimeMinutes < 0 {
		return validationError("prep time cannot be negative")
	}
	if err := ValidatePhysicalState(meal.PhysicalState); err != nil {
		return err
	}
	if strings.TrimSpace(meal.Name) == "" {
		return validationError("meal name is required")
	}
	switch meal.Type {
	case MealTypeSingle:
		if len(meal.RecipeItems) > 0 {
			return validationError("single meal cannot include recipe ingredients")
		}
		if err := ValidateMacrosPer100(meal.MacrosPer100, meal.PhysicalState); err != nil {
			return err
		}
	case MealTypeComposite:
		if len(meal.RecipeItems) == 0 {
			return validationError("composite meal requires ingredients")
		}
		if err := r.validateIngredients(ctx, meal.ID, meal.RecipeItems); err != nil {
			return err
		}
	default:
		return validationError("meal type must be single or composite")
	}
	return r.validateMealTags(ctx, meal.Tags)
}

// validateIngredients checks recipe ingredient fields and ordering.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) validateIngredients(ctx context.Context, mealID uuid.UUID, ingredients []RecipeIngredientEntity) error {
	seenPositions := map[int]struct{}{}
	for _, ingredient := range ingredients {
		if ingredient.FoodItemID == uuid.Nil {
			return validationError("ingredient food item id is required")
		}
		if ingredient.Quantity <= 0 {
			return validationError("ingredient quantity must be positive")
		}
		if _, ok := seenPositions[ingredient.Position]; ok {
			return validationError("ingredient positions must be unique")
		}
		seenPositions[ingredient.Position] = struct{}{}
		food, err := NewPostgresFoodItemRepository(r.db).GetByID(ctx, ingredient.FoodItemID, RepositoryContext{UnitSystem: UnitSystemMetric})
		if err != nil {
			return err
		}
		if _, err := ingredientBasisQuantity(ingredient, food); err != nil {
			return err
		}
	}
	return nil
}

// validateMealTags verifies that referenced meal tags exist.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) validateMealTags(ctx context.Context, tags []TagEntity) error {
	for _, tag := range tags {
		if tag.ID == uuid.Nil {
			return validationError("meal tag id is required")
		}
		var exists bool
		err := r.db.QueryRow(ctx, mealValidateTagSQL, tag.ID).Scan(&exists)
		if err != nil {
			return mapPostgresError(err, "validate meal tag")
		}
		if !exists {
			return validationError("meal tag does not exist")
		}
	}
	return nil
}

// replaceIngredients replaces persisted ingredients for a meal.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) replaceIngredients(ctx context.Context, mealID uuid.UUID, ingredients []RecipeIngredientEntity) error {
	if _, err := r.db.Exec(ctx, mealClearIngredientsSQL, mealID); err != nil {
		return mapPostgresError(err, "clear recipe ingredients")
	}
	for _, ingredient := range ingredients {
		if _, err := r.db.Exec(ctx, mealAttachIngredientSQL, mealID, ingredient.FoodItemID, ingredient.Quantity, ingredient.Unit, ingredient.Position); err != nil {
			return mapPostgresError(err, "replace recipe ingredients")
		}
	}
	return nil
}

// replaceMealTags replaces persisted tag associations for a meal.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) replaceMealTags(ctx context.Context, mealID uuid.UUID, tags []TagEntity) error {
	if _, err := r.db.Exec(ctx, mealClearTagsSQL, mealID); err != nil {
		return mapPostgresError(err, "clear meal tags")
	}
	for _, tag := range tags {
		if _, err := r.db.Exec(ctx, mealAttachTagSQL, mealID, tag.ID); err != nil {
			return mapPostgresError(err, "replace meal tags")
		}
	}
	return nil
}

// getMealByID loads one meal using the provided SQL executor.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) getMealByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (MealEntity, error) {
	var meal MealEntity
	var averageUnitWeight *float64
	var protein *float64
	var carbohydrates *float64
	var fat *float64
	if err := r.db.QueryRow(ctx, mealGetByIDSQL, id, includeDeleted).Scan(&meal.ID, &meal.Type, &meal.Name, &meal.PhysicalState, &meal.PrepTimeMinutes, &averageUnitWeight, &protein, &carbohydrates, &fat, &meal.CreatedAt, &meal.UpdatedAt); err != nil {
		return MealEntity{}, mapPostgresError(err, "meal not found")
	}
	if averageUnitWeight != nil {
		meal.AverageUnitWeightGrams = *averageUnitWeight
	}
	if protein != nil {
		meal.MacrosPer100.Protein = *protein
	}
	if carbohydrates != nil {
		meal.MacrosPer100.Carbohydrates = *carbohydrates
	}
	if fat != nil {
		meal.MacrosPer100.Fat = *fat
	}
	return meal, nil
}

// loadIngredients loads persisted recipe ingredients for a meal.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) loadIngredients(ctx context.Context, mealID uuid.UUID) ([]RecipeIngredientEntity, error) {
	rows, err := r.db.Query(ctx, mealListIngredientsSQL, mealID)
	if err != nil {
		return nil, mapPostgresError(err, "load recipe ingredients")
	}
	defer rows.Close()

	ingredients := []RecipeIngredientEntity{}
	for rows.Next() {
		var ingredient RecipeIngredientEntity
		if err := rows.Scan(&ingredient.FoodItemID, &ingredient.Quantity, &ingredient.Unit, &ingredient.Position); err != nil {
			return nil, mapPostgresError(err, "scan recipe ingredient")
		}
		ingredients = append(ingredients, ingredient)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err, "iterate recipe ingredients")
	}
	return ingredients, nil
}

// hydrateMealTags loads tag IDs onto meal entities.
// Implements DESIGN-005 MealEntity.
func (r *PostgresMealRepository) hydrateMealTags(ctx context.Context, meal *MealEntity) error {
	rows, err := r.db.Query(ctx, mealListTagsSQL, meal.ID)
	if err != nil {
		return mapPostgresError(err, "load meal tags")
	}
	defer rows.Close()

	meal.Tags = nil
	for rows.Next() {
		var tag TagEntity
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Kind, &tag.ParentID); err != nil {
			return mapPostgresError(err, "scan meal tag")
		}
		meal.Tags = append(meal.Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return mapPostgresError(err, "iterate meal tags")
	}
	return nil
}

// ingredientBasisQuantity converts an ingredient quantity to its macro-calculation basis.
// Implements DESIGN-005 MealEntity.
func ingredientBasisQuantity(ingredient RecipeIngredientEntity, food FoodItemEntity) (float64, error) {
	switch ingredient.Unit {
	case "g":
		return ingredient.Quantity, nil
	case "oz":
		return ConvertUnit(ingredient.Quantity, "oz", "g")
	case "ml":
		return ingredient.Quantity, nil
	case "fl_oz":
		return ConvertUnit(ingredient.Quantity, "fl_oz", "ml")
	case "serving":
		quantity, _, err := ConvertServingToBase(ingredient.Quantity, food.AverageUnitWeightGrams, food.AverageServingVolumeMilliliters, food.PhysicalState)
		return quantity, err
	default:
		return 0, unitConversionError("unsupported ingredient unit %q", ingredient.Unit)
	}
}

// ingredientMassGrams returns comparable recipe mass when normalization data exists.
// Implements DESIGN-005 MacroNormalizer.
func ingredientMassGrams(nativeBasis float64, food FoodItemEntity) (float64, bool) {
	if food.PhysicalState == PhysicalStateSolid {
		return nativeBasis, true
	}
	if food.DensityGramsPerMilliliter <= 0 {
		return 0, false
	}
	return nativeBasis * food.DensityGramsPerMilliliter, true
}

// convertMealForUnitSystem converts meal display values to the requested unit system.
// Implements DESIGN-005 MealEntity.
func convertMealForUnitSystem(meal *MealEntity, unitSystem UnitSystem) {
	if unitSystem != UnitSystemImperial {
		return
	}
	switch meal.PhysicalState {
	case PhysicalStateSolid:
		meal.AverageUnitWeightGrams, _ = ConvertUnit(meal.AverageUnitWeightGrams, "g", "oz")
	case PhysicalStateLiquid:
		meal.AverageUnitWeightGrams, _ = ConvertUnit(meal.AverageUnitWeightGrams, "ml", "fl_oz")
	}
}

// nullableMealMacro stores direct macros only for opaque single meals.
// Implements DESIGN-005 MealEntity.
func nullableMealMacro(meal MealEntity, value float64) *float64 {
	if meal.Type != MealTypeSingle {
		return nil
	}
	return &value
}
