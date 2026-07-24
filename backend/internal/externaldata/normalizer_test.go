package externaldata

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-012 DataNormalizer aliases, per-100 conversion, warnings, density, and vocabulary-query verification.

func TestNormalizeUSDAAliasesAndTrustedDensityPriority(t *testing.T) {
	record := ExternalFoodRecord{
		Provider: "usda", ExternalID: "100", Name: "Milk",
		ServingSize: float64Pointer(1), ServingUnit: "fluid ounces",
		Nutrients: map[string]float64{
			"Protein (G)": 3, "Carbohydrate, by difference (G)": 4, "Total lipid (fat) (G)": 2,
			"Sodium, Na (MG)": 40, "Potassium, K (MG)": 150, "Calcium, Ca (MG)": 120,
			"Iron, Fe (MG)": 0.1, "Vitamin C, total ascorbic acid (MG)": 2,
			"Fiber, total dietary (G)": 0.5, "Sugars, total including NLEA (G)": 4,
		},
		Portions: []ExternalFoodPortion{
			{Amount: 1, Unit: "fl oz", GramWeight: millilitersPerFluidOunce * 0.9},
			{Amount: 1, Unit: "tsp", GramWeight: millilitersPerTeaspoon * 0.8},
			{Amount: 1, Unit: "tbsp", GramWeight: millilitersPerTablespoon * 0.7},
			{Amount: 1, Unit: "cup", GramWeight: millilitersPerCup * 0.6},
			{Amount: 10, Unit: "mL", GramWeight: 11},
		},
	}
	candidate, err := NormalizeExternalRecord(record, activeVocabulary())
	if err != nil {
		t.Fatalf("NormalizeExternalRecord() error = %v", err)
	}
	if candidate.PhysicalState != repository.PhysicalStateLiquid || candidate.DensityGramsPerMilliliter != 1.1 || candidate.DensitySourceKind != DensitySourceImported || candidate.DensitySourceProvider != "usda" || candidate.DensitySourceFoodID != "100" {
		t.Fatalf("density candidate = %#v", candidate)
	}
	if candidate.ServingUnit != "fl_oz" || candidate.AverageServingVolumeMilliliters != 29.5735 {
		t.Fatalf("serving projection = %#v", candidate)
	}
	wantMacros := repository.MacroValues{Protein: 3.3, Carbohydrates: 4.4, Fat: 2.2}
	if candidate.MacrosPer100 != wantMacros || candidate.Micros["Sodium"] != 44 || candidate.Micros["Calcium"] != 132 || candidate.Micros["Sugar"] != 4.4 {
		t.Fatalf("normalized nutrients = %#v, %#v", candidate.MacrosPer100, candidate.Micros)
	}
}

func TestTrustedUSDADensityAcceptsEveryDocumentedVolumeAlias(t *testing.T) {
	tests := []struct {
		unit string
		ml   float64
	}{{"millilitres", 1}, {"cups", millilitersPerCup}, {"tablespoons", millilitersPerTablespoon}, {"teaspoons", millilitersPerTeaspoon}, {"fluid ounces", millilitersPerFluidOunce}}
	for _, test := range tests {
		t.Run(test.unit, func(t *testing.T) {
			got := trustedUSDADensity([]ExternalFoodPortion{{Amount: 1, Unit: test.unit, GramWeight: test.ml * 1.25}})
			if got != 1.25 {
				t.Fatalf("trustedUSDADensity(%q) = %v", test.unit, got)
			}
		})
	}
}

func TestNormalizeOpenFoodFactsPer100MillilitersWarnsButDoesNotRejectSuspiciousTotals(t *testing.T) {
	record := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "off-1", Name: "Concentrate", Nutrients: map[string]float64{
		"proteins_100ml": 40, "carbohydrates_100ml": 50, "fat_100ml": 20,
		"sodium_100ml": 0.04, "vitamin-c_100ml": 0.002, "vitamin-d_100ml": 0.000003,
	}}
	candidate, err := NormalizeExternalRecord(record, activeVocabulary())
	if err != nil {
		t.Fatalf("suspicious liquid rejected: %v", err)
	}
	if candidate.PhysicalState != repository.PhysicalStateLiquid || candidate.MacrosPer100 != (repository.MacroValues{Protein: 40, Carbohydrates: 50, Fat: 20}) {
		t.Fatalf("candidate = %#v", candidate)
	}
	if candidate.Micros["Sodium"] != 40 || candidate.Micros["VitaminC"] != 2 || candidate.Micros["VitaminD"] != 3 {
		t.Fatalf("micros = %#v", candidate.Micros)
	}
	if !hasWarning(candidate.Warnings, WarningSuspiciousLiquidMacroSum) || !hasWarning(candidate.Warnings, WarningMissingLiquidDensity) {
		t.Fatalf("warnings = %#v", candidate.Warnings)
	}
}

func TestNormalizeServingAndPackageValuesUsesCanonicalUnitAliases(t *testing.T) {
	serving := ExternalFoodRecord{
		Provider: "openfoodfacts", ExternalID: "serving", Name: "Snack", ServingSize: float64Pointer(2), ServingUnit: "ounces",
		Nutrients: map[string]float64{"proteins_serving": 5, "carbohydrates_serving": 10, "fat_serving": 2, "fiber_serving": 1},
	}
	got, err := NormalizeExternalRecord(serving, activeVocabulary())
	if err != nil {
		t.Fatalf("serving normalization error = %v", err)
	}
	if got.ServingUnit != "oz" || got.AverageUnitWeightGrams != 56.699 || got.MacrosPer100 != (repository.MacroValues{Protein: 8.8185, Carbohydrates: 17.637, Fat: 3.5274}) || got.Micros["Fiber"] != 1.7637 {
		t.Fatalf("serving candidate = %#v", got)
	}

	packaged := ExternalFoodRecord{
		Provider: "openfoodfacts", ExternalID: "package", Name: "Family pack", PackageSize: float64Pointer(500), PackageUnit: "grams",
		Nutrients: map[string]float64{"proteins_package": 50, "carbohydrates_package": 100, "fat_package": 25},
	}
	got, err = NormalizeExternalRecord(packaged, activeVocabulary())
	if err != nil {
		t.Fatalf("package normalization error = %v", err)
	}
	if got.PackageUnit != "g" || got.PackageSize != 500 || got.MacrosPer100 != (repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}) {
		t.Fatalf("package candidate = %#v", got)
	}
}

func TestNormalizeLiquidMassServingUsesTrustedDensity(t *testing.T) {
	for _, test := range []struct {
		name       string
		size       float64
		unit       string
		wantVolume float64
	}{
		{name: "grams", size: 10, unit: "g", wantVolume: 8.3333},
		{name: "ounces", size: 2, unit: "oz", wantVolume: 47.2492},
	} {
		t.Run(test.name, func(t *testing.T) {
			record := ExternalFoodRecord{
				Provider: "usda", ExternalID: "liquid-mass-serving", Name: "Measured liquid",
				ServingSize: float64Pointer(test.size), ServingUnit: test.unit,
				Nutrients: solidUSDAMacros(),
				Portions:  []ExternalFoodPortion{{Amount: 10, Unit: "ml", GramWeight: 12}},
			}
			candidate, err := NormalizeExternalRecord(record, activeVocabulary())
			if err != nil {
				t.Fatalf("NormalizeExternalRecord() error = %v", err)
			}
			if candidate.AverageUnitWeightGrams != 0 || candidate.AverageServingVolumeMilliliters != test.wantVolume {
				t.Fatalf("liquid serving metadata = %#v", candidate)
			}
		})
	}
}

// TestNormalizeLiquidMassServingWithoutDensityStaysIncomplete verifies
// IT-ARCH-012-002, ARCH-012, DESIGN-012 DataNormalizer, and SW-REQ-055/SW-REQ-090.
func TestNormalizeLiquidMassServingWithoutDensityStaysIncomplete(t *testing.T) {
	for _, unit := range []string{"g", "oz"} {
		t.Run(unit, func(t *testing.T) {
			record := ExternalFoodRecord{
				Provider: "openfoodfacts", ExternalID: "liquid-mass-no-density", Name: "Unmeasured liquid",
				ServingSize: float64Pointer(2), ServingUnit: unit,
				Nutrients: map[string]float64{
					"proteins_100ml": 1, "carbohydrates_100ml": 2, "fat_100ml": 3,
				},
			}
			candidate, err := NormalizeExternalRecord(record, activeVocabulary())
			if err != nil {
				t.Fatalf("incomplete liquid rejected: %v", err)
			}
			if candidate.AverageUnitWeightGrams != 0 || candidate.AverageServingVolumeMilliliters != 0 || !hasWarning(candidate.Warnings, WarningMissingLiquidDensity) {
				t.Fatalf("liquid mass serving guessed without density = %#v", candidate)
			}
		})
	}
}

func TestNormalizeLiquidMassPackageUsesDensityOrWarns(t *testing.T) {
	for _, test := range []struct {
		name string
		size float64
		unit string
	}{
		{name: "grams", size: 120, unit: "g"},
		{name: "ounces", size: 120 / gramsPerOunce, unit: "oz"},
	} {
		t.Run(test.name+" with density", func(t *testing.T) {
			record := ExternalFoodRecord{
				Provider: "openfoodfacts", ExternalID: "liquid-mass-package", Name: "Measured liquid",
				PackageSize: float64Pointer(test.size), PackageUnit: test.unit,
				Nutrients: map[string]float64{
					"proteins_package": 12, "carbohydrates_package": 24, "fat_package": 36,
				},
			}
			candidate, err := NormalizeExternalRecordWithOptions(record, activeVocabulary(), NormalizationOptions{
				PhysicalState: repository.PhysicalStateLiquid, DensityGramsPerMilliliter: 1.2, DensitySourceKind: DensitySourceImported,
			})
			if err != nil {
				t.Fatalf("NormalizeExternalRecord() error = %v", err)
			}
			if candidate.MacrosPer100 != (repository.MacroValues{Protein: 12, Carbohydrates: 24, Fat: 36}) {
				t.Fatalf("liquid package macros = %#v", candidate)
			}
		})

		t.Run(test.name+" without density", func(t *testing.T) {
			record := ExternalFoodRecord{
				Provider: "openfoodfacts", ExternalID: "liquid-mass-package-no-density", Name: "Unmeasured liquid",
				PackageSize: float64Pointer(test.size), PackageUnit: test.unit,
				Nutrients: map[string]float64{
					"proteins_package": 12, "carbohydrates_package": 24, "fat_package": 36,
					"sodium_100ml": 0.001,
				},
			}
			candidate, err := NormalizeExternalRecordWithOptions(record, activeVocabulary(), NormalizationOptions{PhysicalState: repository.PhysicalStateLiquid})
			if err != nil {
				t.Fatalf("incomplete liquid rejected: %v", err)
			}
			if candidate.MacrosPer100 != (repository.MacroValues{}) || !hasWarning(candidate.Warnings, WarningMissingLiquidDensity) || !hasWarning(candidate.Warnings, WarningUncertainUnitConversion) {
				t.Fatalf("liquid mass package guessed without density = %#v", candidate)
			}
		})
	}
}

func TestNormalizeRejectsServingMetadataOverflow(t *testing.T) {
	for _, test := range []struct {
		name      string
		unit      string
		nutrients map[string]float64
	}{
		{name: "solid ounces", unit: "oz", nutrients: solidUSDAMacros()},
		{name: "liquid fluid ounces", unit: "fl_oz", nutrients: map[string]float64{
			"proteins_100ml": 1, "carbohydrates_100ml": 2, "fat_100ml": 3,
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			record := ExternalFoodRecord{
				Provider: "openfoodfacts", ExternalID: "overflow", Name: "Extreme serving",
				ServingSize: float64Pointer(math.MaxFloat64), ServingUnit: test.unit, Nutrients: test.nutrients,
			}
			candidate, err := NormalizeExternalRecord(record, activeVocabulary())
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Code != ProviderErrorInvalidPayload {
				t.Fatalf("overflow result = %#v, %v", candidate, err)
			}
			if math.IsInf(candidate.AverageUnitWeightGrams, 0) || math.IsInf(candidate.AverageServingVolumeMilliliters, 0) {
				t.Fatalf("overflow candidate metadata = %#v", candidate)
			}
		})
	}
}

// TestNormalizeNeverAssumesOneMilliliterEqualsOneGram verifies IT-ARCH-012-002,
// ARCH-012, DESIGN-012 DataNormalizer, and SW-REQ-055/SW-REQ-090.
func TestNormalizeNeverAssumesOneMilliliterEqualsOneGram(t *testing.T) {
	record := ExternalFoodRecord{
		Provider: "openfoodfacts", ExternalID: "liquid-no-density", Name: "Unknown liquid",
		ServingSize: float64Pointer(250), ServingUnit: "millilitres",
		Nutrients: map[string]float64{"proteins_100g": 10, "carbohydrates_100g": 20, "fat_100g": 5},
	}
	candidate, err := NormalizeExternalRecord(record, activeVocabulary())
	if err != nil {
		t.Fatalf("incomplete candidate rejected: %v", err)
	}
	if candidate.MacrosPer100 != (repository.MacroValues{}) || !hasWarning(candidate.Warnings, WarningMissingLiquidDensity) || !hasWarning(candidate.Warnings, WarningUncertainUnitConversion) || !hasWarning(candidate.Warnings, WarningMissingMacros) {
		t.Fatalf("candidate silently converted = %#v", candidate)
	}
}

func TestNormalizeDensityProvenanceOptions(t *testing.T) {
	record := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "density", Name: "Drink", Nutrients: map[string]float64{"proteins_100ml": 1, "carbohydrates_100ml": 2, "fat_100ml": 3}}
	for _, kind := range []DensitySourceKind{DensitySourceManual, DensitySourceEstimated, DensitySourceImported} {
		t.Run(string(kind), func(t *testing.T) {
			candidate, err := NormalizeExternalRecordWithOptions(record, activeVocabulary(), NormalizationOptions{PhysicalState: repository.PhysicalStateLiquid, DensityGramsPerMilliliter: 1.05, DensitySourceKind: kind})
			if err != nil {
				t.Fatalf("NormalizeExternalRecordWithOptions() error = %v", err)
			}
			if candidate.DensitySourceKind != kind || candidate.DensityGramsPerMilliliter != 1.05 {
				t.Fatalf("provenance = %#v", candidate)
			}
			if kind == DensitySourceImported && (candidate.DensitySourceProvider != record.Provider || candidate.DensitySourceFoodID != record.ExternalID) {
				t.Fatalf("imported provenance = %#v", candidate)
			}
			if kind != DensitySourceImported && (candidate.DensitySourceProvider != "" || candidate.DensitySourceFoodID != "") {
				t.Fatalf("curator provenance leaked provider identity = %#v", candidate)
			}
		})
	}
}

// TestNormalizeEmitsMissingWarningsAndRejectsUnknownCanonicalMicronutrients
// verifies IT-ARCH-012-002, ARCH-012, DESIGN-012 DataNormalizer, and SW-REQ-055/SW-REQ-090.
func TestNormalizeEmitsMissingWarningsAndRejectsUnknownCanonicalMicronutrients(t *testing.T) {
	record := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "missing", Name: "Incomplete", Nutrients: map[string]float64{}}
	candidate, err := NormalizeExternalRecord(record, activeVocabulary())
	if err != nil {
		t.Fatalf("incomplete candidate rejected: %v", err)
	}
	for _, warning := range []string{WarningMissingImage, WarningMissingMacros, WarningMissingMicronutrients} {
		if !hasWarning(candidate.Warnings, warning) {
			t.Fatalf("warning %q absent from %#v", warning, candidate.Warnings)
		}
	}

	record.Nutrients["sodium_100g"] = 0.1
	if _, err := NormalizeExternalRecord(record, []repository.MicronutrientVocabularyEntry{{Key: "Fiber", Active: true}}); !repository.IsKind(err, repository.ErrorKindInvalidMicronutrientKey) {
		t.Fatalf("unknown canonical micronutrient error = %v", err)
	}
}

// TestDataNormalizerLoadsOneVocabularySnapshotPerWorkflow verifies
// IT-ARCH-012-001, ARCH-012, DESIGN-012 DataNormalizer, and SW-REQ-055/SW-REQ-090.
func TestDataNormalizerLoadsOneVocabularySnapshotPerWorkflow(t *testing.T) {
	vocabulary := &countingVocabulary{entries: activeVocabulary()}
	normalizer := NewDataNormalizer(vocabulary)
	records := []ExternalFoodRecord{
		{Provider: "usda", ExternalID: "1", Name: "One", Nutrients: solidUSDAMacros()},
		{Provider: "openfoodfacts", ExternalID: "2", Name: "Two", Nutrients: map[string]float64{"proteins_100g": 1, "carbohydrates_100g": 2, "fat_100g": 3}},
		{Provider: "usda", ExternalID: "3", Name: "Three", Nutrients: solidUSDAMacros()},
	}
	candidates, err := normalizer.NormalizeRecords(context.Background(), records)
	if err != nil || len(candidates) != 3 {
		t.Fatalf("NormalizeRecords() = %#v, %v", candidates, err)
	}
	if vocabulary.listCalls != 1 || vocabulary.allowedCalls != 0 {
		t.Fatalf("vocabulary queries: ListActive=%d IsAllowed=%d", vocabulary.listCalls, vocabulary.allowedCalls)
	}

	vocabulary.err = errors.New("query failed")
	if _, err := normalizer.NormalizeRecords(context.Background(), records); !errors.Is(err, vocabulary.err) {
		t.Fatalf("query error = %v", err)
	}
	if _, err := normalizer.NormalizeRecords(nil, records); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := NewDataNormalizer(nil).NormalizeRecords(context.Background(), records); err == nil {
		t.Fatal("nil vocabulary accepted")
	}
}

func TestNormalizeRejectsMalformedRecordsAndDensityOptions(t *testing.T) {
	valid := ExternalFoodRecord{Provider: "usda", ExternalID: "1", Name: "Food", Nutrients: solidUSDAMacros()}
	tests := []struct {
		name    string
		record  ExternalFoodRecord
		options NormalizationOptions
	}{
		{"provider", ExternalFoodRecord{Provider: "other", ExternalID: "1", Name: "Food", Nutrients: map[string]float64{}}, NormalizationOptions{}},
		{"too many nutrients", ExternalFoodRecord{Provider: "usda", ExternalID: "1", Name: "Food", Nutrients: oversizedNutrientMap()}, NormalizationOptions{}},
		{"nonfinite nutrient", ExternalFoodRecord{Provider: "usda", ExternalID: "1", Name: "Food", Nutrients: map[string]float64{"Protein (G)": math.Inf(1)}}, NormalizationOptions{}},
		{"density without kind", valid, NormalizationOptions{DensityGramsPerMilliliter: 1}},
		{"invalid density", valid, NormalizationOptions{DensityGramsPerMilliliter: -1, DensitySourceKind: DensitySourceManual}},
		{"solid density", valid, NormalizationOptions{PhysicalState: repository.PhysicalStateSolid, DensityGramsPerMilliliter: 1, DensitySourceKind: DensitySourceEstimated}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NormalizeExternalRecordWithOptions(test.record, activeVocabulary(), test.options); err == nil {
				t.Fatal("malformed input accepted")
			}
		})
	}
}

func TestNormalizerCoversDefensiveConversionBranches(t *testing.T) {
	valid := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "1", Name: "Food", Nutrients: map[string]float64{
		"protein_100g": 1, "proteins_100g": 2, "carbohydrates_100g": 3, "fat_100g": 4,
		"sodium_100g": 0.001, "sodium-na_100g": 0.002, "energy-kcal_100g": 50,
	}}
	if candidate, err := NormalizeExternalRecord(valid, activeVocabulary()); err != nil || candidate.MacrosPer100.Protein != 1 || candidate.Micros["Sodium"] != 2 {
		t.Fatalf("deterministic alias priority = %#v, %v", candidate, err)
	}
	invalidState := NormalizationOptions{PhysicalState: repository.PhysicalState("gas")}
	if _, err := NormalizeExternalRecordWithOptions(valid, activeVocabulary(), invalidState); err == nil {
		t.Fatal("invalid physical state accepted")
	}
	highMacros := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "2", Name: "Impossible solid", Nutrients: map[string]float64{"proteins_100g": 50, "carbohydrates_100g": 50, "fat_100g": 50}}
	if _, err := NormalizeExternalRecord(highMacros, activeVocabulary()); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("solid macro invariant error = %v", err)
	}
	overflow := ExternalFoodRecord{Provider: "openfoodfacts", ExternalID: "3", Name: "Overflow", PackageSize: float64Pointer(0.001), PackageUnit: "g", Nutrients: map[string]float64{"sodium_package": math.MaxFloat64}}
	if _, err := NormalizeExternalRecord(overflow, activeVocabulary()); err == nil {
		t.Fatal("normalized overflow accepted")
	}
	if _, err := NewDataNormalizer(&countingVocabulary{entries: activeVocabulary()}).NormalizeRecords(context.Background(), []ExternalFoodRecord{{Provider: "other"}}); err == nil {
		t.Fatal("per-item workflow error omitted")
	}

	if size, unit := canonicalQuantity(float64Pointer(1), "cup"); size != 0 || unit != "" {
		t.Fatalf("unsupported canonical quantity = %v %q", size, unit)
	}
	if size, unit := canonicalQuantity(nil, "g"); size != 0 || unit != "" {
		t.Fatalf("nil canonical quantity = %v %q", size, unit)
	}
	if density := trustedUSDADensity([]ExternalFoodPortion{{Amount: 0, Unit: "ml", GramWeight: 1}, {Amount: 1, Unit: "unknown", GramWeight: 1}, {Amount: 1, Unit: "ml", GramWeight: 1}, {Amount: 1, Unit: "cup", GramWeight: 1}}); density != 1 {
		t.Fatalf("invalid portions produced density %v", density)
	}
	if density := trustedUSDADensity([]ExternalFoodPortion{{Amount: 1, Unit: "ml", GramWeight: math.SmallestNonzeroFloat64}}); density != 0 {
		t.Fatalf("rounded-zero density accepted: %v", density)
	}
	if ml, rank := volumeMilliliters(0, "ml"); ml != 0 || rank != 99 {
		t.Fatalf("invalid volume = %v, %d", ml, rank)
	}
	if ml, rank := volumeMilliliters(1, "unknown"); ml != 0 || rank != 99 || normalizeVolumeAlias("pint") != "" {
		t.Fatalf("unknown volume = %v, %d", ml, rank)
	}

	for _, test := range []struct {
		state      repository.PhysicalState
		unit       string
		wantWeight float64
		wantVolume float64
	}{{repository.PhysicalStateSolid, "g", 2, 0}, {repository.PhysicalStateSolid, "oz", 56.699, 0}, {repository.PhysicalStateLiquid, "ml", 0, 2}, {repository.PhysicalStateLiquid, "fl_oz", 0, 59.1471}, {repository.PhysicalStateSolid, "serving", 0, 0}, {repository.PhysicalStateLiquid, "serving", 0, 0}} {
		candidate := NormalizedFoodCandidate{PhysicalState: test.state, ServingSize: 2, ServingUnit: test.unit}
		if err := setServingMeasures(&candidate); err != nil {
			t.Fatalf("setServingMeasures(%q) error = %v", test.unit, err)
		}
		if candidate.AverageUnitWeightGrams != test.wantWeight || candidate.AverageServingVolumeMilliliters != test.wantVolume {
			t.Fatalf("setServingMeasures(%q) = %#v", test.unit, candidate)
		}
	}
	emptyServing := NormalizedFoodCandidate{}
	if err := setServingMeasures(&emptyServing); err != nil {
		t.Fatalf("empty serving error = %v", err)
	}

	if _, _, ok := classifyUSDANutrient("malformed"); ok {
		t.Fatal("malformed USDA key classified")
	}
	if _, _, ok := classifyUSDANutrient("Energy (KCAL)"); ok {
		t.Fatal("unknown USDA micronutrient classified")
	}
	if _, _, ok := classifyUSDANutrient("Sodium (KCAL)"); ok {
		t.Fatal("unsupported USDA nutrient unit classified")
	}
	if _, _, ok := classifyOpenFoodFactsNutrient("energy_100g"); ok {
		t.Fatal("unknown OpenFoodFacts micronutrient classified")
	}
	if _, _, ok := classifyOpenFoodFactsNutrient("proteins"); ok {
		t.Fatal("basis-free OpenFoodFacts key classified")
	}
	if macroAlias("energy") != "" || microAlias("energy") != "" {
		t.Fatal("unknown nutrient alias accepted")
	}

	unitCases := []struct {
		micro string
		unit  string
		want  float64
		ok    bool
	}{{"Sodium", "g", 1000, true}, {"Fiber", "mg", 0.001, true}, {"VitaminD", "mg", 1000, true}, {"Sodium", "mcg", 0.001, true}, {"Sodium", "kcal", 0, false}}
	for _, test := range unitCases {
		got, ok := microUnitFactor(test.micro, test.unit)
		if got != test.want || ok != test.ok {
			t.Fatalf("microUnitFactor(%q, %q) = %v, %t", test.micro, test.unit, got, ok)
		}
	}

	quantityCases := []struct {
		state   repository.PhysicalState
		density float64
		size    float64
		unit    string
		ok      bool
	}{{repository.PhysicalStateLiquid, 1.2, 10, "g", true}, {repository.PhysicalStateLiquid, 1, 2, "fl_oz", true}, {repository.PhysicalStateLiquid, 0, 2, "ml", true}, {repository.PhysicalStateSolid, 0, 1, "ml", false}, {repository.PhysicalStateSolid, 0, 0, "g", false}, {repository.PhysicalStateSolid, 0, 1, "serving", false}}
	for _, test := range quantityCases {
		_, ok := quantityPer100Factor(test.state, test.density, test.size, test.unit)
		if ok != test.ok {
			t.Fatalf("quantityPer100Factor(%q, %v, %v, %q) ok = %t", test.state, test.density, test.size, test.unit, ok)
		}
	}
	if _, _, ok := per100Factor(nutrientBasis(99), repository.PhysicalStateSolid, 0, 0, "", 0, ""); ok {
		t.Fatal("unknown nutrient basis converted")
	}
}

type countingVocabulary struct {
	entries      []repository.MicronutrientVocabularyEntry
	err          error
	listCalls    int
	allowedCalls int
}

func (v *countingVocabulary) ListActive(context.Context) ([]repository.MicronutrientVocabularyEntry, error) {
	v.listCalls++
	return v.entries, v.err
}

func (v *countingVocabulary) IsAllowed(context.Context, string) (bool, error) {
	v.allowedCalls++
	return false, nil
}

func (v *countingVocabulary) Upsert(context.Context, repository.MicronutrientVocabularyEntry) error {
	return nil
}

func activeVocabulary() []repository.MicronutrientVocabularyEntry {
	keys := []string{"Sodium", "Potassium", "Calcium", "Iron", "VitaminC", "VitaminD", "Fiber", "Sugar"}
	entries := make([]repository.MicronutrientVocabularyEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, repository.MicronutrientVocabularyEntry{Key: key, Active: true})
	}
	return entries
}

func solidUSDAMacros() map[string]float64 {
	return map[string]float64{"Protein (G)": 1, "Carbohydrate, by difference (G)": 2, "Total lipid (fat) (G)": 3}
}

func oversizedNutrientMap() map[string]float64 {
	values := make(map[string]float64, maxExternalNutrientFields+1)
	for i := 0; i <= maxExternalNutrientFields; i++ {
		values[string(rune(i+1))] = 0
	}
	return values
}

func float64Pointer(value float64) *float64 { return &value }

func hasWarning(warnings []string, want string) bool {
	for _, warning := range warnings {
		if warning == want {
			return true
		}
	}
	return false
}
