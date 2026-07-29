package repository

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Implements DESIGN-009 AdminController deterministic global catalog export.
//
//go:embed sql/global_catalog_export.sql
var globalCatalogExportSQL string

// Implements DESIGN-009 AdminController read-only consistent snapshot.
const globalCatalogReadOnlyTransactionSQL = "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY"

// CatalogExportClassification identifies one portable global classification relationship.
// Implements DESIGN-009 AdminController global catalog export representation.
type CatalogExportClassification struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CatalogExportCuratedSource reports informational curated-import identity.
// Implements DESIGN-009 AdminController global catalog export representation.
type CatalogExportCuratedSource struct {
	ID         uuid.UUID `json:"id"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"externalId"`
	Status     string    `json:"status"`
}

// CatalogExportItem is one complete ownerless global catalog snapshot row.
// Implements DESIGN-009 AdminController global catalog export representation.
type CatalogExportItem struct {
	ID                              uuid.UUID
	Name                            string
	PhysicalState                   PhysicalState
	PrepTimeMinutes                 int
	AverageUnitWeightGrams          *string
	AverageServingVolumeMilliliters *string
	DensityGramsPerMilliliter       *string
	DensitySourceProvider           *string
	DensitySourceFoodID             *string
	DensitySourceKind               *string
	ProteinPer100                   string
	CarbohydratesPer100             string
	FatPer100                       string
	Micros                          map[string]json.Number
	ImageURL                        *string
	ImageAlt                        *string
	SourceProvider                  *string
	ExternalID                      *string
	DeletedAt                       *time.Time
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
	FoodCategories                  []CatalogExportClassification
	CulinaryRoles                   []CatalogExportClassification
	AllergenKeys                    []string
	CuratedSources                  []CatalogExportCuratedSource
}

// CatalogExportRowConsumer receives one row at a time from the consistent snapshot.
// Implements DESIGN-009 AdminController bounded global catalog export.
type CatalogExportRowConsumer func(CatalogExportItem) error

// PostgresGlobalCatalogExportRepository streams only ownerless global catalog rows.
// Implements DESIGN-009 AdminController global/private export separation.
type PostgresGlobalCatalogExportRepository struct {
	db transactionalExecutor
}

// NewPostgresGlobalCatalogExportRepository creates the read-only export repository.
// Implements DESIGN-009 AdminController.
func NewPostgresGlobalCatalogExportRepository(db transactionalExecutor) *PostgresGlobalCatalogExportRepository {
	return &PostgresGlobalCatalogExportRepository{db: db}
}

// Stream visits UUID-ordered global items in one repeatable-read, read-only snapshot.
// Implements DESIGN-009 AdminController bounded consistent global catalog export.
func (r *PostgresGlobalCatalogExportRepository) Stream(ctx context.Context, includeDeleted bool, consume CatalogExportRowConsumer) (count int, err error) {
	if r == nil || r.db == nil || consume == nil {
		return 0, NewError(ErrorKindConnection, "global catalog export is unavailable", nil)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, mapPostgresError(err, "begin global catalog export")
	}
	completed := false
	defer func() {
		if !completed {
			rollbackContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = tx.Rollback(rollbackContext)
		}
	}()
	if _, err = tx.Exec(ctx, globalCatalogReadOnlyTransactionSQL); err != nil {
		return 0, mapPostgresError(err, "configure global catalog export")
	}
	rows, err := tx.Query(ctx, globalCatalogExportSQL, includeDeleted)
	if err != nil {
		return 0, mapPostgresError(err, "query global catalog export")
	}
	defer rows.Close()
	for rows.Next() {
		item, scanErr := scanGlobalCatalogExportItem(rows)
		if scanErr != nil {
			return count, scanErr
		}
		if consumeErr := consume(item); consumeErr != nil {
			return count, consumeErr
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, mapPostgresError(err, "iterate global catalog export")
	}
	if err := tx.Commit(ctx); err != nil {
		return count, mapPostgresError(err, "complete global catalog export")
	}
	completed = true
	return count, nil
}

// scanGlobalCatalogExportItem decodes one bounded snapshot row.
// Implements DESIGN-009 AdminController global catalog export projection.
func scanGlobalCatalogExportItem(row foodRowScanner) (CatalogExportItem, error) {
	var item CatalogExportItem
	var micros, categories, roles, allergens, curated []byte
	if err := row.Scan(
		&item.ID, &item.Name, &item.PhysicalState, &item.PrepTimeMinutes,
		&item.AverageUnitWeightGrams, &item.AverageServingVolumeMilliliters,
		&item.DensityGramsPerMilliliter, &item.DensitySourceProvider,
		&item.DensitySourceFoodID, &item.DensitySourceKind,
		&item.ProteinPer100, &item.CarbohydratesPer100, &item.FatPer100, &micros,
		&item.ImageURL, &item.ImageAlt, &item.SourceProvider, &item.ExternalID,
		&item.DeletedAt, &item.CreatedAt, &item.UpdatedAt,
		&categories, &roles, &allergens, &curated,
	); err != nil {
		return CatalogExportItem{}, mapPostgresError(err, "scan global catalog export")
	}
	item.Micros = map[string]json.Number{}
	if err := decodeCatalogJSON(micros, &item.Micros); err != nil {
		return CatalogExportItem{}, err
	}
	if err := decodeCatalogJSON(categories, &item.FoodCategories); err != nil {
		return CatalogExportItem{}, err
	}
	if err := decodeCatalogJSON(roles, &item.CulinaryRoles); err != nil {
		return CatalogExportItem{}, err
	}
	if err := decodeCatalogJSON(allergens, &item.AllergenKeys); err != nil {
		return CatalogExportItem{}, err
	}
	if err := decodeCatalogJSON(curated, &item.CuratedSources); err != nil {
		return CatalogExportItem{}, err
	}
	return item, nil
}

// decodeCatalogJSON preserves numeric values while decoding sorted JSON aggregates.
// Implements DESIGN-009 AdminController deterministic global catalog encoding.
func decodeCatalogJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return NewError(ErrorKindValidation, "global catalog data is invalid", err)
	}
	return nil
}
