// Package catalogexport encodes deterministic ownerless global catalog snapshots.
package catalogexport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Schema is the shared Task 277/278 global catalog representation.
// Implements DESIGN-009 AdminController global catalog export contract.
const Schema = "mealswapp.global-catalog.v1"

// Store streams one consistent PostgreSQL snapshot.
// Implements DESIGN-009 AdminController bounded global catalog export.
type Store interface {
	Stream(context.Context, bool, repository.CatalogExportRowConsumer) (int, error)
}

// Service writes canonical global catalog JSON without retaining the catalog in memory.
// Implements DESIGN-009 AdminController deterministic global catalog export.
type Service struct {
	store Store
}

// NewService creates a global catalog export service.
// Implements DESIGN-009 AdminController.
func NewService(store Store) *Service { return &Service{store: store} }

// catalogEntry binds stable replay identity to one global item.
// Implements DESIGN-009 AdminController global catalog export contract.
type catalogEntry struct {
	IdempotencyKey string      `json:"idempotencyKey"`
	Item           catalogItem `json:"item"`
}

// catalogItem contains importable fields and informational catalog metadata.
// Implements DESIGN-009 AdminController global catalog export contract.
type catalogItem struct {
	ID                              string                                  `json:"id"`
	Name                            string                                  `json:"name"`
	PhysicalState                   repository.PhysicalState                `json:"physicalState"`
	PrepTimeMinutes                 int                                     `json:"prepTimeMinutes"`
	AverageUnitWeightGrams          *json.Number                            `json:"averageUnitWeightGrams"`
	AverageServingVolumeMilliliters *json.Number                            `json:"averageServingVolumeMilliliters"`
	DensityGramsPerMilliliter       *json.Number                            `json:"densityGramsPerMilliliter"`
	DensitySourceProvider           *string                                 `json:"densitySourceProvider"`
	DensitySourceFoodID             *string                                 `json:"densitySourceFoodId"`
	DensitySourceKind               *string                                 `json:"densitySourceKind"`
	MacrosPer100                    catalogMacros                           `json:"macrosPer100"`
	Micros                          map[string]json.Number                  `json:"micros"`
	FoodCategoryNames               []string                                `json:"foodCategoryNames"`
	CulinaryRoleNames               []string                                `json:"culinaryRoleNames"`
	AllergenKeys                    []string                                `json:"allergenKeys"`
	ImageURL                        *string                                 `json:"imageUrl"`
	ImageAlt                        *string                                 `json:"imageAlt"`
	SourceProvider                  *string                                 `json:"sourceProvider"`
	ExternalID                      *string                                 `json:"externalId"`
	CreatedAt                       string                                  `json:"createdAt"`
	UpdatedAt                       string                                  `json:"updatedAt"`
	DeletedAt                       *string                                 `json:"deletedAt"`
	Classifications                 catalogClassifications                  `json:"classifications"`
	CuratedSources                  []repository.CatalogExportCuratedSource `json:"curatedSources"`
}

// catalogMacros preserves exact PostgreSQL numeric text as JSON numbers.
// Implements DESIGN-009 AdminController deterministic numeric encoding.
type catalogMacros struct {
	Protein       json.Number `json:"protein"`
	Carbohydrates json.Number `json:"carbohydrates"`
	Fat           json.Number `json:"fat"`
}

// catalogClassifications reports UUID/name relationship metadata.
// Implements DESIGN-009 AdminController global catalog relationship export.
type catalogClassifications struct {
	FoodCategories []repository.CatalogExportClassification `json:"foodCategories"`
	CulinaryRoles  []repository.CatalogExportClassification `json:"culinaryRoles"`
}

// Write streams one compact canonical document and returns its item count.
// Implements DESIGN-009 AdminController deterministic bounded export.
func (s *Service) Write(ctx context.Context, output io.Writer, includeDeleted bool) (int, error) {
	if s == nil || s.store == nil || output == nil {
		return 0, repository.NewError(repository.ErrorKindConnection, "global catalog export is unavailable", nil)
	}
	if _, err := io.WriteString(output, `{"schema":"`+Schema+`","items":[`); err != nil {
		return 0, err
	}
	first := true
	count, err := s.store.Stream(ctx, includeDeleted, func(row repository.CatalogExportItem) error {
		if !first {
			if _, err := io.WriteString(output, ","); err != nil {
				return err
			}
		}
		first = false
		entry := catalogEntry{IdempotencyKey: "global-catalog:" + row.ID.String(), Item: project(row)}
		encoded, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("encode global catalog item: %w", err)
		}
		_, err = output.Write(encoded)
		return err
	})
	if err != nil {
		return count, err
	}
	if _, err := io.WriteString(output, "]}\n"); err != nil {
		return count, err
	}
	return count, nil
}

// project converts one snapshot row to the shared interchange representation.
// Implements DESIGN-009 AdminController global catalog export contract.
func project(row repository.CatalogExportItem) catalogItem {
	deletedAt := (*string)(nil)
	if row.DeletedAt != nil {
		value := timestamp(*row.DeletedAt)
		deletedAt = &value
	}
	micros := row.Micros
	if micros == nil {
		micros = map[string]json.Number{}
	}
	allergens := row.AllergenKeys
	if allergens == nil {
		allergens = []string{}
	}
	curated := row.CuratedSources
	if curated == nil {
		curated = []repository.CatalogExportCuratedSource{}
	}
	categories := row.FoodCategories
	if categories == nil {
		categories = []repository.CatalogExportClassification{}
	}
	roles := row.CulinaryRoles
	if roles == nil {
		roles = []repository.CatalogExportClassification{}
	}
	return catalogItem{
		ID: row.ID.String(), Name: row.Name, PhysicalState: row.PhysicalState, PrepTimeMinutes: row.PrepTimeMinutes,
		AverageUnitWeightGrams: number(row.AverageUnitWeightGrams), AverageServingVolumeMilliliters: number(row.AverageServingVolumeMilliliters),
		DensityGramsPerMilliliter: number(row.DensityGramsPerMilliliter), DensitySourceProvider: row.DensitySourceProvider,
		DensitySourceFoodID: row.DensitySourceFoodID, DensitySourceKind: row.DensitySourceKind,
		MacrosPer100: catalogMacros{Protein: canonicalNumber(row.ProteinPer100), Carbohydrates: canonicalNumber(row.CarbohydratesPer100), Fat: canonicalNumber(row.FatPer100)},
		Micros:       micros, FoodCategoryNames: classificationNames(categories), CulinaryRoleNames: classificationNames(roles),
		AllergenKeys: allergens, ImageURL: row.ImageURL, ImageAlt: row.ImageAlt, SourceProvider: row.SourceProvider,
		ExternalID: row.ExternalID, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: deletedAt,
		Classifications: catalogClassifications{FoodCategories: categories, CulinaryRoles: roles},
		CuratedSources:  curated,
	}
}

// number preserves nullable database numerics as JSON numbers.
// Implements DESIGN-009 AdminController deterministic numeric encoding.
func number(value *string) *json.Number {
	if value == nil {
		return nil
	}
	result := canonicalNumber(*value)
	return &result
}

// canonicalNumber removes insignificant fractional zeroes.
// Implements DESIGN-009 AdminController deterministic numeric encoding.
func canonicalNumber(value string) json.Number {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "" {
		value = "0"
	}
	return json.Number(value)
}

// classificationNames projects ordered relationships to portable import names.
// Implements DESIGN-009 AdminController global catalog relationship export.
func classificationNames(values []repository.CatalogExportClassification) []string {
	if values == nil {
		return []string{}
	}
	names := make([]string, len(values))
	for index, value := range values {
		names[index] = value.Name
	}
	return names
}

// timestamp canonicalizes PostgreSQL times to UTC without adding volatile time.
// Implements DESIGN-009 AdminController deterministic timestamp encoding.
func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
