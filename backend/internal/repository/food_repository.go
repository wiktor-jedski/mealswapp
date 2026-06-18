package repository

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/google/uuid"
)

// Implements DESIGN-005 FoodItemEntity search query.
//
//go:embed sql/food_search.sql
var foodSearchSQL string

// Implements DESIGN-005 RepositoryInterfaces search count query.
//
//go:embed sql/food_search_count.sql
var foodSearchCountSQL string

// Implements DESIGN-005 FoodItemEntity create query.
//
//go:embed sql/food_create.sql
var foodCreateSQL string

// Implements DESIGN-005 FoodItemEntity update query.
//
//go:embed sql/food_update.sql
var foodUpdateSQL string

// Implements DESIGN-005 FoodItemEntity soft-delete query.
//
//go:embed sql/food_soft_delete.sql
var foodSoftDeleteSQL string

// Implements DESIGN-005 FoodItemEntity classification validation query.
//
//go:embed sql/food_validate_classification.sql
var foodValidateClassificationSQL string

// Implements DESIGN-005 FoodItemEntity clear-classifications query.
//
//go:embed sql/food_clear_classifications.sql
var foodClearClassificationsSQL string

// Implements DESIGN-005 FoodItemEntity attach-classification query.
//
//go:embed sql/food_attach_classification.sql
var foodAttachClassificationSQL string

// Implements DESIGN-005 FoodItemEntity get-by-id query.
//
//go:embed sql/food_get_by_id.sql
var foodGetByIDSQL string

// Implements DESIGN-005 FoodItemEntity hydrate-classifications query.
//
//go:embed sql/food_list_classifications.sql
var foodListClassificationsSQL string

// PostgresFoodItemRepository persists normalized food items in PostgreSQL.
// Implements DESIGN-005 FoodItemEntity.
type PostgresFoodItemRepository struct {
	db transactionalExecutor
}

// Implements DESIGN-005 FoodItemEntity compile-time repository contract.
var _ FoodItemRepository = (*PostgresFoodItemRepository)(nil)

// NewPostgresFoodItemRepository creates a PostgreSQL-backed food item repository.
// Implements DESIGN-005 FoodItemEntity.
func NewPostgresFoodItemRepository(db transactionalExecutor) *PostgresFoodItemRepository {
	return &PostgresFoodItemRepository{db: db}
}

// GetByID loads one food item with hydrated classifications.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) GetByID(ctx context.Context, id uuid.UUID, rc RepositoryContext) (FoodItemEntity, error) {
	item, err := r.getFoodByID(ctx, id, rc.IncludeDeleted)
	if err != nil {
		return FoodItemEntity{}, err
	}
	if err := r.hydrateFoodClassifications(ctx, &item); err != nil {
		return FoodItemEntity{}, err
	}
	convertFoodItemForUnitSystem(&item, rc.UnitSystem)
	return item, nil
}

// Search returns matching food items and total count for deterministic pagination.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) Search(ctx context.Context, q RepositoryQuery) ([]FoodItemEntity, int, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	foodObjectTypes := physicalStatesToStrings(q.FoodObjectTypes)
	excludedFoodObjectTypes := physicalStatesToStrings(q.ExcludedFoodObjectTypes)
	if err := r.db.QueryRow(ctx, foodSearchCountSQL, q.IncludeDeleted, q.Name, q.MaxPrepMinutes, q.FoodCategoryIDs, q.ExcludedFoodCategoryIDs, q.CulinaryRoleIDs, q.ExcludedCulinaryRoleIDs, q.AllergenIDs, q.ExcludedAllergenIDs, q.AllergenKeys, q.ExcludedAllergenKeys, foodObjectTypes, excludedFoodObjectTypes).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err, "count food items")
	}

	rows, err := r.db.Query(ctx, foodSearchSQL, q.IncludeDeleted, q.Name, q.MaxPrepMinutes, q.FoodCategoryIDs, q.ExcludedFoodCategoryIDs, q.CulinaryRoleIDs, q.ExcludedCulinaryRoleIDs, q.AllergenIDs, q.ExcludedAllergenIDs, q.AllergenKeys, q.ExcludedAllergenKeys, foodObjectTypes, excludedFoodObjectTypes, limit, offset)
	if err != nil {
		return nil, 0, mapPostgresError(err, "search food items")
	}
	defer rows.Close()

	items := []FoodItemEntity{}
	for rows.Next() {
		item, err := scanFoodItem(rows)
		if err != nil {
			if IsKind(err, ErrorKindValidation) {
				return nil, 0, err
			}
			return nil, 0, mapPostgresError(err, "scan food item")
		}
		if err := r.hydrateFoodClassifications(ctx, &item); err != nil {
			return nil, 0, err
		}
		convertFoodItemForUnitSystem(&item, q.UnitSystem)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err, "iterate food items")
	}
	return items, total, nil
}

// physicalStatesToStrings converts domain physical-state filters to SQL parameters.
// Implements DESIGN-005 FoodItemEntity search query.
func physicalStatesToStrings(states []PhysicalState) []string {
	if len(states) == 0 {
		return nil
	}
	values := make([]string, 0, len(states))
	for _, state := range states {
		values = append(values, string(state))
	}
	return values
}

// Create validates and persists a food item.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) Create(ctx context.Context, item FoodItemEntity) (uuid.UUID, error) {
	if err := r.validateFoodItem(ctx, item); err != nil {
		return uuid.Nil, err
	}
	micros := marshalMicros(item.Micros)

	var id uuid.UUID
	err := withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		txRepo := NewPostgresFoodItemRepository(db)
		err := db.QueryRow(ctx, foodCreateSQL, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, micros, nullableString(item.ImageURL)).Scan(&id)
		if err != nil {
			return mapPostgresError(err, "create food item")
		}
		return txRepo.replaceFoodClassifications(ctx, id, item.FoodCategories, item.CulinaryRoles)
	})
	return id, err
}

// Update validates and replaces a food item and its classification assignments.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) Update(ctx context.Context, item FoodItemEntity) error {
	if item.ID == uuid.Nil {
		return validationError("food item id is required")
	}
	if err := r.validateFoodItem(ctx, item); err != nil {
		return err
	}
	micros := marshalMicros(item.Micros)

	return withTransaction(ctx, r.db, func(db transactionalExecutor) error {
		txRepo := NewPostgresFoodItemRepository(db)
		result, err := db.Exec(ctx, foodUpdateSQL, item.ID, item.Name, string(item.PhysicalState), item.PrepTimeMinutes, nullablePositiveFloat(item.AverageUnitWeightGrams), nullablePositiveFloat(item.AverageServingVolumeMilliliters), nullablePositiveFloat(item.DensityGramsPerMilliliter), nullableString(item.DensitySourceProvider), nullableString(item.DensitySourceFoodID), nullableString(item.DensitySourceKind), item.MacrosPer100.Protein, item.MacrosPer100.Carbohydrates, item.MacrosPer100.Fat, micros, nullableString(item.ImageURL))
		if err != nil {
			return mapPostgresError(err, "update food item")
		}
		if result.RowsAffected() == 0 {
			return NewError(ErrorKindNotFound, "food item not found", nil)
		}
		return txRepo.replaceFoodClassifications(ctx, item.ID, item.FoodCategories, item.CulinaryRoles)
	})
}

// Delete soft-deletes a food item.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, foodSoftDeleteSQL, id)
	if err != nil {
		return mapPostgresError(err, "delete food item")
	}
	if result.RowsAffected() == 0 {
		return NewError(ErrorKindNotFound, "food item not found", nil)
	}
	return nil
}

// validateFoodItem checks food item fields before persistence.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) validateFoodItem(ctx context.Context, item FoodItemEntity) error {
	if item.Name == "" {
		return validationError("food item name is required")
	}
	if item.PrepTimeMinutes < 0 {
		return validationError("prep time cannot be negative")
	}
	if err := ValidatePhysicalState(item.PhysicalState); err != nil {
		return err
	}
	if err := ValidateMacrosPer100(item.MacrosPer100, item.PhysicalState); err != nil {
		return err
	}
	if err := validateFoodDensity(item); err != nil {
		return err
	}
	if err := r.validateFoodClassifications(ctx, item.FoodCategories, ClassificationKindFoodCategory); err != nil {
		return err
	}
	if err := r.validateFoodClassifications(ctx, item.CulinaryRoles, ClassificationKindCulinaryRole); err != nil {
		return err
	}
	return r.validateMicronutrients(ctx, item.Micros)
}

// validateFoodClassifications verifies that food item classifications exist with the required kinds.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) validateFoodClassifications(ctx context.Context, classifications []ClassificationEntity, kind ClassificationKind) error {
	for _, classification := range classifications {
		if classification.ID == uuid.Nil {
			return validationError("classification id is required")
		}
		var exists bool
		err := r.db.QueryRow(ctx, foodValidateClassificationSQL, classification.ID, string(kind)).Scan(&exists)
		if err != nil {
			return mapPostgresError(err, "validate food classification")
		}
		if !exists {
			return validationError("classification does not exist for required kind")
		}
	}
	return nil
}

// validateMicronutrients verifies that micronutrient keys are active vocabulary entries.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) validateMicronutrients(ctx context.Context, micros MicroValues) error {
	repo := NewPostgresMicronutrientVocabularyRepository(r.db)
	entries, err := repo.ListActive(ctx)
	if err != nil {
		return err
	}
	return ValidateMicronutrientKeys(micros, entries)
}

// replaceFoodClassifications replaces persisted classification associations for a food item.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) replaceFoodClassifications(ctx context.Context, foodID uuid.UUID, foodCategories []ClassificationEntity, culinaryRoles []ClassificationEntity) error {
	if _, err := r.db.Exec(ctx, foodClearClassificationsSQL, foodID); err != nil {
		return mapPostgresError(err, "clear food classifications")
	}
	for _, classification := range append(foodCategories, culinaryRoles...) {
		if _, err := r.db.Exec(ctx, foodAttachClassificationSQL, foodID, classification.ID); err != nil {
			return mapPostgresError(err, "replace food classifications")
		}
	}
	return nil
}

// getFoodByID loads one food item using the provided SQL executor.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) getFoodByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (FoodItemEntity, error) {
	row := r.db.QueryRow(ctx, foodGetByIDSQL, id, includeDeleted)
	item, err := scanFoodItem(row)
	if err != nil {
		if IsKind(err, ErrorKindValidation) {
			return FoodItemEntity{}, err
		}
		return FoodItemEntity{}, mapPostgresError(err, "food item not found")
	}
	return item, nil
}

// hydrateFoodClassifications loads classification IDs onto food item entities.
// Implements DESIGN-005 FoodItemEntity.
func (r *PostgresFoodItemRepository) hydrateFoodClassifications(ctx context.Context, item *FoodItemEntity) error {
	rows, err := r.db.Query(ctx, foodListClassificationsSQL, item.ID)
	if err != nil {
		return mapPostgresError(err, "load food classifications")
	}
	defer rows.Close()

	item.FoodCategories = nil
	item.CulinaryRoles = nil
	for rows.Next() {
		var classification ClassificationEntity
		if err := rows.Scan(&classification.ID, &classification.Name, &classification.Kind, &classification.ParentID); err != nil {
			return mapPostgresError(err, "scan food classification")
		}
		switch classification.Kind {
		case ClassificationKindFoodCategory:
			item.FoodCategories = append(item.FoodCategories, classification)
		case ClassificationKindCulinaryRole:
			item.CulinaryRoles = append(item.CulinaryRoles, classification)
		}
	}
	if err := rows.Err(); err != nil {
		return mapPostgresError(err, "iterate food classifications")
	}
	return nil
}

// foodRowScanner describes a PostgreSQL row that can populate scanned values.
// Implements DESIGN-005 FoodItemEntity.
type foodRowScanner interface {
	Scan(dest ...any) error
}

// scanFoodItem reads a food item from a PostgreSQL row.
// Implements DESIGN-005 FoodItemEntity.
func scanFoodItem(row foodRowScanner) (FoodItemEntity, error) {
	var item FoodItemEntity
	var averageUnitWeight *float64
	var averageServingVolume *float64
	var density *float64
	var densitySourceProvider *string
	var densitySourceFoodID *string
	var densitySourceKind *string
	var microsBytes []byte
	var imageURL *string
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.PhysicalState,
		&item.PrepTimeMinutes,
		&averageUnitWeight,
		&averageServingVolume,
		&density,
		&densitySourceProvider,
		&densitySourceFoodID,
		&densitySourceKind,
		&item.MacrosPer100.Protein,
		&item.MacrosPer100.Carbohydrates,
		&item.MacrosPer100.Fat,
		&microsBytes,
		&imageURL,
		&item.DeletedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return FoodItemEntity{}, err
	}
	if averageUnitWeight != nil {
		item.AverageUnitWeightGrams = *averageUnitWeight
	}
	setFoodDensityFields(&item, averageServingVolume, density, densitySourceProvider, densitySourceFoodID, densitySourceKind)
	if imageURL != nil {
		item.ImageURL = *imageURL
	}
	if len(microsBytes) > 0 {
		if err := json.Unmarshal(microsBytes, &item.Micros); err != nil {
			return FoodItemEntity{}, validationError("micronutrients must be a JSON object")
		}
	}
	if item.Micros == nil {
		item.Micros = MicroValues{}
	}
	return item, nil
}

// convertFoodItemForUnitSystem converts display values to the requested unit system.
// Implements DESIGN-005 FoodItemEntity.
func convertFoodItemForUnitSystem(item *FoodItemEntity, unitSystem UnitSystem) {
	if unitSystem != UnitSystemImperial {
		return
	}
	switch item.PhysicalState {
	case PhysicalStateSolid:
		item.AverageUnitWeightGrams, _ = ConvertUnit(item.AverageUnitWeightGrams, "g", "oz")
	case PhysicalStateLiquid:
		item.AverageServingVolumeMilliliters, _ = ConvertUnit(item.AverageServingVolumeMilliliters, "ml", "fl_oz")
	}
}

// validateFoodDensity checks required liquid density metadata.
// Implements DESIGN-005 FoodItemEntity.
func validateFoodDensity(item FoodItemEntity) error {
	if item.AverageServingVolumeMilliliters < 0 || item.DensityGramsPerMilliliter < 0 {
		return validationError("liquid serving volume and density cannot be negative")
	}
	if item.PhysicalState == PhysicalStateSolid && (item.AverageServingVolumeMilliliters > 0 || item.DensityGramsPerMilliliter > 0) {
		return validationError("liquid serving volume and density require liquid physical state")
	}
	if item.DensityGramsPerMilliliter == 0 {
		if item.DensitySourceProvider != "" || item.DensitySourceFoodID != "" || item.DensitySourceKind != "" {
			return validationError("density provenance requires density")
		}
		if item.PhysicalState == PhysicalStateLiquid {
			return validationError("liquid density is required")
		}
		return nil
	}
	if item.DensitySourceKind != "imported" && item.DensitySourceKind != "manual" && item.DensitySourceKind != "estimated" {
		return validationError("density source kind is invalid")
	}
	return nil
}

// setFoodDensityFields copies nullable liquid metadata from PostgreSQL.
// Implements DESIGN-005 FoodItemEntity.
func setFoodDensityFields(item *FoodItemEntity, servingVolume *float64, density *float64, provider *string, foodID *string, kind *string) {
	if servingVolume != nil {
		item.AverageServingVolumeMilliliters = *servingVolume
	}
	if density != nil {
		item.DensityGramsPerMilliliter = *density
	}
	if provider != nil {
		item.DensitySourceProvider = *provider
	}
	if foodID != nil {
		item.DensitySourceFoodID = *foodID
	}
	if kind != nil {
		item.DensitySourceKind = *kind
	}
}

// nullablePositiveFloat converts non-positive optional values to SQL null.
// Implements DESIGN-005 FoodItemEntity.
func nullablePositiveFloat(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	return &value
}

// nullableString converts blank optional strings to SQL null.
// Implements DESIGN-005 FoodItemEntity.
func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// marshalMicros serializes micronutrient values for PostgreSQL storage.
// Implements DESIGN-005 FoodItemEntity.
func marshalMicros(micros MicroValues) []byte {
	if micros == nil {
		micros = MicroValues{}
	}
	encoded, _ := json.Marshal(micros)
	return encoded
}
