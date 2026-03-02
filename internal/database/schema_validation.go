// Phase: phase-01 | Task: 16 | Architecture: ARCH-005 | Design: Database
package database

import (
	"context"
	"fmt"
)

type SchemaValidationResult struct {
	Valid   bool
	Errors  []SchemaValidationError
	Checked int
	Passed  int
}

type SchemaValidationError struct {
	Table    string
	Column   string
	Expected string
	Actual   string
	Message  string
}

type TableSchema struct {
	Name    string
	Columns []ColumnSchema
	Indexes []string
}

type ColumnSchema struct {
	Name     string
	Type     string
	Nullable bool
}

var tagTableSchema = TableSchema{
	Name: "tags",
	Columns: []ColumnSchema{
		{Name: "id", Type: "uuid", Nullable: false},
		{Name: "name", Type: "character varying", Nullable: false},
		{Name: "slug", Type: "character varying", Nullable: false},
		{Name: "type", Type: "character varying", Nullable: false},
		{Name: "description", Type: "character varying", Nullable: true},
		{Name: "color_hex", Type: "character varying", Nullable: true},
		{Name: "icon_url", Type: "character varying", Nullable: true},
		{Name: "created_at", Type: "timestamp with time zone", Nullable: false},
		{Name: "updated_at", Type: "timestamp with time zone", Nullable: false},
	},
	Indexes: []string{
		"idx_tags_slug",
		"idx_tags_type",
		"idx_tags_name",
		"idx_tags_type_name",
	},
}

var foodItemTableSchema = TableSchema{
	Name: "food_items",
	Columns: []ColumnSchema{
		{Name: "id", Type: "uuid", Nullable: false},
		{Name: "name", Type: "character varying", Nullable: false},
		{Name: "physical_state", Type: "character varying", Nullable: false},
		{Name: "prep_time", Type: "integer", Nullable: false},
		{Name: "average_unit_weight", Type: "numeric", Nullable: false},
		{Name: "macros", Type: "jsonb", Nullable: false},
		{Name: "micros", Type: "jsonb", Nullable: false},
		{Name: "image_url", Type: "text", Nullable: true},
		{Name: "created_at", Type: "timestamp with time zone", Nullable: false},
		{Name: "updated_at", Type: "timestamp with time zone", Nullable: false},
	},
	Indexes: []string{
		"idx_food_items_name_unique",
		"idx_food_items_physical_state",
		"idx_food_items_macros",
		"idx_food_items_created_at",
	},
}

var foodItemCategoryTagsTableSchema = TableSchema{
	Name: "food_item_category_tags",
	Columns: []ColumnSchema{
		{Name: "food_item_id", Type: "uuid", Nullable: false},
		{Name: "tag_id", Type: "uuid", Nullable: false},
	},
	Indexes: []string{
		"idx_food_item_category_tags_food_item_id",
		"idx_food_item_category_tags_tag_id",
	},
}

var foodItemFunctionalityTagsTableSchema = TableSchema{
	Name: "food_item_functionality_tags",
	Columns: []ColumnSchema{
		{Name: "food_item_id", Type: "uuid", Nullable: false},
		{Name: "tag_id", Type: "uuid", Nullable: false},
	},
	Indexes: []string{
		"idx_food_item_functionality_tags_food_item_id",
		"idx_food_item_functionality_tags_tag_id",
	},
}

var requiredTables = []TableSchema{
	tagTableSchema,
	foodItemTableSchema,
	foodItemCategoryTagsTableSchema,
	foodItemFunctionalityTagsTableSchema,
}

func ValidateSchema(ctx context.Context) (*SchemaValidationResult, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool not initialized")
	}

	result := &SchemaValidationResult{
		Valid:   true,
		Errors:  []SchemaValidationError{},
		Checked: 0,
		Passed:  0,
	}

	for _, tableSchema := range requiredTables {
		tableValid, tableErrors := validateTable(ctx, tableSchema)
		result.Checked += len(tableSchema.Columns) + len(tableSchema.Indexes) + 1

		if !tableValid {
			result.Valid = false
			result.Errors = append(result.Errors, tableErrors...)
		} else {
			result.Passed += len(tableSchema.Columns) + len(tableSchema.Indexes) + 1
		}
	}

	return result, nil
}

func validateTable(ctx context.Context, schema TableSchema) (bool, []SchemaValidationError) {
	errors := []SchemaValidationError{}

	exists, err := tableExists(ctx, schema.Name)
	if err != nil {
		return false, []SchemaValidationError{{
			Table:   schema.Name,
			Message: fmt.Sprintf("failed to check table existence: %v", err),
		}}
	}

	if !exists {
		return false, []SchemaValidationError{{
			Table:   schema.Name,
			Message: fmt.Sprintf("table %q does not exist", schema.Name),
		}}
	}

	for _, col := range schema.Columns {
		colExists, colType, isNullable, err := columnInfo(ctx, schema.Name, col.Name)
		if err != nil {
			errors = append(errors, SchemaValidationError{
				Table:   schema.Name,
				Column:  col.Name,
				Message: fmt.Sprintf("failed to check column: %v", err),
			})
			continue
		}

		if !colExists {
			errors = append(errors, SchemaValidationError{
				Table:    schema.Name,
				Column:   col.Name,
				Expected: col.Type,
				Message:  fmt.Sprintf("column %q does not exist in table %q", col.Name, schema.Name),
			})
			continue
		}

		if colType != col.Type && !typeMatches(colType, col.Type) {
			errors = append(errors, SchemaValidationError{
				Table:    schema.Name,
				Column:   col.Name,
				Expected: col.Type,
				Actual:   colType,
				Message:  fmt.Sprintf("column %q has type %q, expected %q", col.Name, colType, col.Type),
			})
		}

		if col.Nullable != isNullable {
			errors = append(errors, SchemaValidationError{
				Table:    schema.Name,
				Column:   col.Name,
				Expected: fmt.Sprintf("nullable=%v", col.Nullable),
				Actual:   fmt.Sprintf("nullable=%v", isNullable),
				Message:  fmt.Sprintf("column %q nullability mismatch", col.Name),
			})
		}
	}

	for _, idx := range schema.Indexes {
		exists, err := indexExists(ctx, schema.Name, idx)
		if err != nil {
			errors = append(errors, SchemaValidationError{
				Table:   schema.Name,
				Message: fmt.Sprintf("failed to check index %q: %v", idx, err),
			})
			continue
		}

		if !exists {
			errors = append(errors, SchemaValidationError{
				Table:   schema.Name,
				Column:  idx,
				Message: fmt.Sprintf("index %q does not exist on table %q", idx, schema.Name),
			})
		}
	}

	return len(errors) == 0, errors
}

func tableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	err := Pool.QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

func columnInfo(ctx context.Context, tableName, columnName string) (exists bool, dataType string, nullable bool, err error) {
	query := `
		SELECT data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = $1 
		AND column_name = $2
	`
	var nullableStr string
	err = Pool.QueryRow(ctx, query, tableName, columnName).Scan(&dataType, &nullableStr)
	if err != nil {
		return false, "", false, err
	}
	return true, dataType, nullableStr == "YES", nil
}

func indexExists(ctx context.Context, tableName, indexName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE tablename = $1 
			AND indexname = $2
		)
	`
	err := Pool.QueryRow(ctx, query, tableName, indexName).Scan(&exists)
	return exists, err
}

func typeMatches(actual, expected string) bool {
	typeMap := map[string][]string{
		"character varying": {"varchar", "text"},
		"numeric":           {"numeric", "decimal"},
	}

	actualLower := normalizeType(actual)
	expectedLower := normalizeType(expected)

	if actualLower == expectedLower {
		return true
	}

	for base, variants := range typeMap {
		if actualLower == base {
			for _, v := range variants {
				if expectedLower == v {
					return true
				}
			}
		}
	}

	return false
}

func normalizeType(t string) string {
	switch t {
	case "varchar", "text", "character varying":
		return "character varying"
	case "numeric", "decimal":
		return "numeric"
	default:
		return t
	}
}

func ValidateEntitySchemas(ctx context.Context) error {
	result, err := ValidateSchema(ctx)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("schema validation errors: %v", result.Errors)
	}

	fmt.Printf("Schema validation passed: %d/%d checks passed\n", result.Passed, result.Checked)
	return nil
}
