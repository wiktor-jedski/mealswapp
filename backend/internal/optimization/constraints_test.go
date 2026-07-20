// Implements DESIGN-004 ConstraintBuilder verification.
package optimization

import (
	"context"
	"math"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

var (
	constraintUser  = uuid.MustParse("00000000-0000-4000-8000-000000000010")
	constraintDiet  = uuid.MustParse("00000000-0000-4000-8000-000000000011")
	constraintMealA = uuid.MustParse("00000000-0000-4000-8000-000000000001")
	constraintMealB = uuid.MustParse("00000000-0000-4000-8000-000000000002")
	constraintMealC = uuid.MustParse("00000000-0000-4000-8000-000000000003")
)

func TestConstraintDomainUsesOneProductionVocabulary(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		fields []string
	}{
		{name: "macro target", value: MacroTarget{}, fields: []string{"Protein", "Carbohydrates", "Fat"}},
		{name: "saved diet request", value: DietOptimizationRequest{}, fields: []string{"OriginalDiet", "TolerancePercent", "ExcludedMealIDs"}},
		{name: "LP variable", value: LPVariable{}, fields: []string{"ItemID", "LowerBound", "UpperBound", "CaloriesPerUnit", "DiversityPenalty", "ProteinPerUnit", "CarbohydratesPerUnit", "FatPerUnit"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeOf := reflect.TypeOf(tt.value)
			fields := make([]string, typeOf.NumField())
			for index := range typeOf.NumField() {
				fields[index] = typeOf.Field(index).Name
			}
			if !reflect.DeepEqual(fields, tt.fields) {
				t.Fatalf("%s fields = %v, want exactly %v", typeOf.Name(), fields, tt.fields)
			}
		})
	}
}

func TestBuildConstraintsUsesAuthoritativeSavedDietAndCanonicalDomain(t *testing.T) {
	mealA := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 30, Fat: 2})
	mealB := eligibleConstraintMeal(constraintMealB, repository.PhysicalStateLiquid, MacroTarget{Protein: 20, Carbohydrates: 40, Fat: 5})
	req := savedDietConstraintRequest(mealA, 100, "g", 10)
	req.ExcludedMealIDs = []uuid.UUID{constraintMealB}

	model, err := BuildConstraints(req, []repository.MealEntity{mealB, mealA}, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	if got, want := len(model.Variables), 1; got != want {
		t.Fatalf("variable count = %d, want %d; excluded meals must be omitted", got, want)
	}
	variable := model.Variables[0]
	if variable.ItemID != constraintMealA.String() || variable.UpperBound != MaximumMealQuantity || variable.CarbohydratesPerUnit != 0.3 {
		t.Fatalf("variable = %+v", variable)
	}
	if got, want := len(model.Constraints), 3; got != want {
		t.Fatalf("constraint count = %d, want macro rows only (%d)", got, want)
	}
	assertConstraintBounds(t, model.Constraints[0], 9, 11)
	assertConstraintBounds(t, model.Constraints[1], 27, 33)
	assertConstraintBounds(t, model.Constraints[2], 1.8, 2.2)
	for _, constraint := range model.Constraints {
		if len(constraint.Name) >= 9 && (constraint.Name[:9] == "quantity_" || constraint.Name[:8] == "exclude_") {
			t.Fatalf("redundant bound row remains: %s", constraint.Name)
		}
	}
}

func TestBuildConstraintsNormalizesSupportedOriginalDietBases(t *testing.T) {
	tests := []struct {
		name           string
		state          repository.PhysicalState
		metricQuantity float64
		metricUnit     string
		imperialUnit   string
	}{
		{name: "solid ounces", state: repository.PhysicalStateSolid, metricQuantity: 28.3495, metricUnit: "g", imperialUnit: "oz"},
		{name: "liquid fluid ounces", state: repository.PhysicalStateLiquid, metricQuantity: 29.5735, metricUnit: "ml", imperialUnit: "fl_oz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meal := eligibleConstraintMeal(constraintMealA, tt.state, MacroTarget{Protein: 20, Carbohydrates: 10, Fat: 2})
			metric, err := BuildConstraints(savedDietConstraintRequest(meal, tt.metricQuantity, tt.metricUnit, 0), []repository.MealEntity{meal}, nil)
			if err != nil {
				t.Fatalf("BuildConstraints(metric) error = %v", err)
			}
			imperial, err := BuildConstraints(savedDietConstraintRequest(meal, 1, tt.imperialUnit, 0), []repository.MealEntity{meal}, nil)
			if err != nil {
				t.Fatalf("BuildConstraints(imperial) error = %v", err)
			}
			if !reflect.DeepEqual(imperial, metric) {
				t.Fatalf("imperial model differs from metric model:\nimperial=%+v\nmetric=%+v", imperial, metric)
			}
		})
	}
}

func TestBuildConstraintsEligibilityPolicy(t *testing.T) {
	original := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	ineligibleState := eligibleConstraintMeal(constraintMealB, "", MacroTarget{Protein: 10})
	unavailable := eligibleConstraintMeal(constraintMealC, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	unavailable.NormalizedMacrosAvailable = false
	invalidBasis := eligibleConstraintMeal(uuid.MustParse("00000000-0000-4000-8000-000000000004"), repository.PhysicalStateSolid, MacroTarget{Protein: -1})
	zeroInformation := eligibleConstraintMeal(uuid.MustParse("00000000-0000-4000-8000-000000000005"), repository.PhysicalStateSolid, MacroTarget{})

	model, err := BuildConstraints(savedDietConstraintRequest(original, 100, "g", 0), []repository.MealEntity{unavailable, zeroInformation, invalidBasis, original, ineligibleState}, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	if len(model.Variables) != 1 || model.Variables[0].ItemID != original.ID.String() {
		t.Fatalf("eligible variables = %+v, want only original", model.Variables)
	}

	for _, mutation := range []func(*repository.MealEntity){
		func(meal *repository.MealEntity) { meal.PhysicalState = "" },
		func(meal *repository.MealEntity) { meal.PhysicalState = "gas" },
		func(meal *repository.MealEntity) { meal.NormalizedMacrosAvailable = false },
		func(meal *repository.MealEntity) { meal.MacrosPer100.Protein = -1 },
	} {
		bad := original
		mutation(&bad)
		if _, err := BuildConstraints(savedDietConstraintRequest(bad, 100, "g", 0), []repository.MealEntity{bad}, nil); err == nil {
			t.Fatal("BuildConstraints() accepted an unusable authoritative original meal")
		}
	}

	req := savedDietConstraintRequest(original, 100, "g", 0)
	req.OriginalDiet.Entries = append(req.OriginalDiet.Entries, repository.SavedDietMealEntry{MealID: zeroInformation.ID, Quantity: 100, Unit: "g"})
	if _, err := BuildConstraints(req, []repository.MealEntity{original, zeroInformation}, nil); err == nil {
		t.Fatal("BuildConstraints() filtered a zero-information original meal instead of failing safely")
	}
}

func TestBuildConstraintsRejectsAllZeroAuthoritativeTargetAndInvalidIdentity(t *testing.T) {
	zero := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{})
	if _, err := BuildConstraints(savedDietConstraintRequest(zero, 100, "g", 0), []repository.MealEntity{zero}, nil); err == nil {
		t.Fatal("BuildConstraints() accepted an all-zero target")
	}

	valid := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 1})
	for _, req := range []DietOptimizationRequest{
		{},
		{OriginalDiet: repository.SavedDiet{ID: constraintDiet, Entries: []repository.SavedDietMealEntry{{MealID: valid.ID, Quantity: 100, Unit: "g"}}}},
		{OriginalDiet: repository.SavedDiet{ID: constraintDiet, UserID: constraintUser}},
	} {
		if _, err := BuildConstraints(req, []repository.MealEntity{valid}, nil); err == nil {
			t.Fatal("BuildConstraints() accepted incomplete saved-diet identity")
		}
	}
}

func TestBuildConstraintsMealSetDistinctnessCannotUseQuantityDrift(t *testing.T) {
	mealA := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	mealB := eligibleConstraintMeal(constraintMealB, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	previous := LPSolution{mealA.ID.String(): 25, mealB.ID.String(): 100}
	req := savedDietConstraintRequest(mealA, 100, "g", 5)

	first, err := BuildConstraints(req, []repository.MealEntity{mealB, mealA}, []LPSolution{previous})
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	reversed, err := BuildConstraints(req, []repository.MealEntity{mealA, mealB}, []LPSolution{previous})
	if err != nil {
		t.Fatalf("BuildConstraints(reversed) error = %v", err)
	}
	if !reflect.DeepEqual(first, reversed) {
		t.Fatalf("matrix changes with candidate order:\nfirst=%+v\nreversed=%+v", first, reversed)
	}
	alternative := findConstraint(t, first, "alternative_1")
	assertConstraintBounds(t, alternative, 0, 0)
	if !reflect.DeepEqual(alternative.Coefficients, map[string]float64{mealB.ID.String(): 1}) {
		t.Fatalf("alternative coefficients = %v, want deterministic highest-quantity exclusion", alternative.Coefficients)
	}
	if got := constraintValue(alternative, map[string]float64{mealA.ID.String(): 23.75, mealB.ID.String(): 99}); got <= alternative.UpperBound {
		t.Fatalf("same-set quantity drift value = %v, want rejected", got)
	}
	if got := constraintValue(alternative, map[string]float64{mealA.ID.String(): 100}); got > alternative.UpperBound {
		t.Fatalf("changed meal set value = %v, want accepted", got)
	}
}

func TestBuildConstraintsDeterministicFeasibleAndInfeasibleFixtures(t *testing.T) {
	meal := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5})
	tests := []struct {
		name       string
		quantity   float64
		assignment LPSolution
		feasible   bool
	}{
		{name: "feasible", quantity: 100, assignment: LPSolution{meal.ID.String(): 100}, feasible: true},
		{name: "bounded infeasible", quantity: MaximumMealQuantity + 1, feasible: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := BuildConstraints(savedDietConstraintRequest(meal, tt.quantity, "g", 0), []repository.MealEntity{meal}, nil)
			if err != nil {
				t.Fatalf("BuildConstraints() error = %v", err)
			}
			got := modelAccepts(model, tt.assignment)
			if tt.assignment == nil {
				got = matrixCanReachLowerBounds(model)
			}
			if got != tt.feasible {
				t.Fatalf("fixture feasibility = %v, want %v", got, tt.feasible)
			}
		})
	}
}

func TestConstraintBuilderLoadsEligibleCatalogInBoundedPages(t *testing.T) {
	meals := make([]repository.MealEntity, 201)
	for index := range meals {
		meals[index] = eligibleConstraintMeal(uuid.New(), repository.PhysicalStateSolid, MacroTarget{Protein: 1})
	}
	ineligible := eligibleConstraintMeal(uuid.New(), repository.PhysicalStateSolid, MacroTarget{Protein: 1})
	ineligible.NormalizedMacrosAvailable = false
	meals = append(meals, ineligible)
	mealRepo := &pagedConstraintMealRepository{meals: meals}
	dietRepo := &constraintDietRepository{diet: repository.SavedDiet{
		ID: constraintDiet, UserID: constraintUser,
		Entries: []repository.SavedDietMealEntry{{MealID: meals[0].ID, Quantity: 100, Unit: "g"}},
	}}

	inputs, err := NewConstraintBuilder(mealRepo, dietRepo).LoadFromSavedDiet(context.Background(), constraintUser, constraintDiet, DietOptimizationRequest{})
	if err != nil {
		t.Fatalf("LoadFromSavedDiet() error = %v", err)
	}
	if len(inputs.Meals) != 201 || mealRepo.searchCalls != 3 || mealRepo.getCalls != 1 {
		t.Fatalf("loaded=%d searchCalls=%d getCalls=%d, want 201, 3, 1", len(inputs.Meals), mealRepo.searchCalls, mealRepo.getCalls)
	}
	if !mealRepo.sawSupportedStates {
		t.Fatal("repository query did not restrict candidates to supported physical states")
	}
	if inputs.Request.OriginalDiet.ID != constraintDiet || inputs.Request.OriginalDiet.UserID != constraintUser {
		t.Fatalf("loaded request diet = %+v", inputs.Request.OriginalDiet)
	}
}

func TestConstraintBuilderUsesFoodItemsAsSourceOnlyNutrition(t *testing.T) {
	foodID, candidateID := uuid.New(), uuid.New()
	candidate := eligibleConstraintMeal(candidateID, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	diet := repository.SavedDiet{ID: constraintDiet, UserID: constraintUser, Entries: []repository.SavedDietMealEntry{{
		FoodObjectID: foodID, FoodObjectType: repository.FoodObjectTypeFoodItem, Quantity: 100, Unit: "g",
	}}}
	foods := &constraintFoodRepository{foods: map[uuid.UUID]repository.FoodItemEntity{
		foodID: {ID: foodID, Name: "Source Food", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}},
	}}
	inputs, err := NewConstraintBuilder(&pagedConstraintMealRepository{meals: []repository.MealEntity{candidate}}, &constraintDietRepository{diet: diet}, foods).LoadFromSavedDiet(context.Background(), constraintUser, constraintDiet, DietOptimizationRequest{})
	if err != nil {
		t.Fatalf("LoadFromSavedDiet() error = %v", err)
	}
	model, err := BuildConstraints(inputs.Request, inputs.Meals, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	if len(model.Variables) != 1 || model.Variables[0].ItemID != candidateID.String() {
		t.Fatalf("variables = %+v, want Meal candidate only", model.Variables)
	}
}

func TestConstraintBuilderRequiresExactRepositoryDietIdentity(t *testing.T) {
	meal := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 1})
	for _, diet := range []repository.SavedDiet{
		{ID: uuid.Nil, UserID: constraintUser, Entries: []repository.SavedDietMealEntry{{MealID: meal.ID, Quantity: 100, Unit: "g"}}},
		{ID: constraintDiet, UserID: uuid.Nil, Entries: []repository.SavedDietMealEntry{{MealID: meal.ID, Quantity: 100, Unit: "g"}}},
		{ID: constraintDiet, UserID: uuid.New(), Entries: []repository.SavedDietMealEntry{{MealID: meal.ID, Quantity: 100, Unit: "g"}}},
	} {
		builder := NewConstraintBuilder(&pagedConstraintMealRepository{meals: []repository.MealEntity{meal}}, &constraintDietRepository{diet: diet})
		if _, err := builder.LoadFromSavedDiet(context.Background(), constraintUser, constraintDiet, DietOptimizationRequest{}); err == nil {
			t.Fatalf("LoadFromSavedDiet() accepted diet identity %+v", diet)
		}
	}
}

func TestBuildConstraintsRejectsInvalidNumericAndTypedExclusionInputs(t *testing.T) {
	meal := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10})
	base := savedDietConstraintRequest(meal, 100, "g", 0)
	tests := []DietOptimizationRequest{base, base, base, base}
	tests[0].TolerancePercent = math.Inf(1)
	tests[1].TolerancePercent = -1
	tests[2].ExcludedMealIDs = []uuid.UUID{uuid.Nil}
	tests[3].ExcludedMealIDs = []uuid.UUID{constraintMealB, constraintMealB}
	for _, req := range tests {
		if _, err := BuildConstraints(req, []repository.MealEntity{meal}, nil); err == nil {
			t.Fatal("BuildConstraints() accepted invalid request input")
		}
	}
	if _, err := BuildConstraints(base, []repository.MealEntity{meal}, []LPSolution{{meal.ID.String(): math.Inf(1)}}); err == nil {
		t.Fatal("BuildConstraints() accepted non-finite prior quantity")
	}
}

func TestTask218PackagedCLPConstraintFixture(t *testing.T) {
	executable := os.Getenv("MEALSWAPP_CLP_PATH")
	if executable == "" {
		executable, _ = exec.LookPath(DefaultCLPExecutable)
	}
	if executable == "" {
		t.Skip("native CLP executable is not installed; packaged worker CI supplies it")
	}
	meal := eligibleConstraintMeal(constraintMealA, repository.PhysicalStateSolid, MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5})
	model, err := BuildConstraints(savedDietConstraintRequest(meal, 100, "g", 0), []repository.MealEntity{meal}, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	policy, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}
	solver := NewLPSolverWrapper(CLPConfig{Executable: executable})
	solution, err := solver.Solve(context.Background(), model, policy.Primary)
	if err != nil {
		t.Fatalf("packaged CLP solve: %v", err)
	}
	if quantity := solution[meal.ID.String()]; math.Abs(quantity-100) > SolutionValidationEpsilon {
		t.Fatalf("packaged CLP quantity = %v, want 100", quantity)
	}
}

func eligibleConstraintMeal(id uuid.UUID, state repository.PhysicalState, macros MacroTarget) repository.MealEntity {
	return repository.MealEntity{
		ID: id, Type: repository.MealTypeSingle, Name: id.String(), PhysicalState: state,
		MacrosPer100:              repository.MacroValues{Protein: macros.Protein, Carbohydrates: macros.Carbohydrates, Fat: macros.Fat},
		NormalizedMacrosAvailable: true,
	}
}

func savedDietConstraintRequest(original repository.MealEntity, quantity float64, unit string, tolerance float64) DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalDiet: repository.SavedDiet{
			ID: constraintDiet, UserID: constraintUser,
			Entries: []repository.SavedDietMealEntry{{MealID: original.ID, Quantity: quantity, Unit: unit}},
		},
		TolerancePercent: tolerance,
	}
}

func assertConstraintBounds(t *testing.T, constraint LPConstraint, lower, upper float64) {
	t.Helper()
	if math.Abs(constraint.LowerBound-lower) > 1e-12 || math.Abs(constraint.UpperBound-upper) > 1e-12 {
		t.Fatalf("%s bounds = [%v, %v], want [%v, %v]", constraint.Name, constraint.LowerBound, constraint.UpperBound, lower, upper)
	}
}

func findConstraint(t *testing.T, model LPModel, name string) LPConstraint {
	t.Helper()
	for _, constraint := range model.Constraints {
		if constraint.Name == name {
			return constraint
		}
	}
	t.Fatalf("constraint %q not found in %+v", name, model.Constraints)
	return LPConstraint{}
}

func constraintValue(constraint LPConstraint, quantities map[string]float64) float64 {
	value := 0.0
	for itemID, coefficient := range constraint.Coefficients {
		value += coefficient * quantities[itemID]
	}
	return value
}

func modelAccepts(model LPModel, quantities map[string]float64) bool {
	for _, variable := range model.Variables {
		quantity := quantities[variable.ItemID]
		if quantity < variable.LowerBound || quantity > variable.UpperBound {
			return false
		}
	}
	for _, constraint := range model.Constraints {
		value := constraintValue(constraint, quantities)
		if value < constraint.LowerBound-1e-9 || value > constraint.UpperBound+1e-9 {
			return false
		}
	}
	return true
}

func matrixCanReachLowerBounds(model LPModel) bool {
	for _, constraint := range model.Constraints {
		maximum := 0.0
		for _, variable := range model.Variables {
			coefficient := constraint.Coefficients[variable.ItemID]
			if coefficient > 0 {
				maximum += coefficient * variable.UpperBound
			}
		}
		if maximum < constraint.LowerBound-1e-9 {
			return false
		}
	}
	return true
}

type pagedConstraintMealRepository struct {
	meals                 []repository.MealEntity
	searchCalls, getCalls int
	sawSupportedStates    bool
}

func (r *pagedConstraintMealRepository) GetByID(_ context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.MealEntity, error) {
	r.getCalls++
	for _, meal := range r.meals {
		if meal.ID == id {
			return meal, nil
		}
	}
	return repository.MealEntity{}, repository.NewError(repository.ErrorKindNotFound, "missing meal", nil)
}

func (r *pagedConstraintMealRepository) Search(_ context.Context, query repository.RepositoryQuery) ([]repository.MealEntity, int, error) {
	r.searchCalls++
	r.sawSupportedStates = reflect.DeepEqual(query.FoodObjectTypes, []repository.PhysicalState{repository.PhysicalStateSolid, repository.PhysicalStateLiquid})
	if query.Offset >= len(r.meals) {
		return []repository.MealEntity{}, len(r.meals), nil
	}
	end := min(query.Offset+query.Limit, len(r.meals))
	return append([]repository.MealEntity(nil), r.meals[query.Offset:end]...), len(r.meals), nil
}

func (*pagedConstraintMealRepository) CalculateMacros(context.Context, uuid.UUID) (repository.MacroValues, error) {
	return repository.MacroValues{}, nil
}
func (*pagedConstraintMealRepository) Create(context.Context, repository.MealEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (*pagedConstraintMealRepository) Update(context.Context, repository.MealEntity) error {
	return nil
}
func (*pagedConstraintMealRepository) Delete(context.Context, uuid.UUID) error { return nil }

type constraintDietRepository struct{ diet repository.SavedDiet }

type constraintFoodRepository struct {
	foods map[uuid.UUID]repository.FoodItemEntity
}

func (r *constraintFoodRepository) GetByID(_ context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.FoodItemEntity, error) {
	food, ok := r.foods[id]
	if !ok {
		return repository.FoodItemEntity{}, repository.NewError(repository.ErrorKindNotFound, "Food Item not found", nil)
	}
	return food, nil
}
func (*constraintFoodRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	return nil, 0, nil
}
func (*constraintFoodRepository) Create(context.Context, repository.FoodItemEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (*constraintFoodRepository) Update(context.Context, repository.FoodItemEntity) error { return nil }
func (*constraintFoodRepository) Delete(context.Context, uuid.UUID) error                 { return nil }

func (*constraintDietRepository) Create(context.Context, uuid.UUID, repository.SavedDiet) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (r *constraintDietRepository) Get(context.Context, uuid.UUID, uuid.UUID) (repository.SavedDiet, error) {
	return r.diet, nil
}
func (*constraintDietRepository) List(context.Context, uuid.UUID) ([]repository.SavedDiet, error) {
	return nil, nil
}
func (*constraintDietRepository) Replace(context.Context, uuid.UUID, repository.SavedDiet) error {
	return nil
}
func (*constraintDietRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
