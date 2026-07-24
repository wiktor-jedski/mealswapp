package externaldata

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// DensitySourceKind identifies how a liquid density was obtained.
// Implements DESIGN-012 DataNormalizer and DESIGN-005 FoodItemEntity provenance.
type DensitySourceKind string

// Implements DESIGN-012 DataNormalizer density provenance.
const (
	DensitySourceImported  DensitySourceKind = "imported"
	DensitySourceManual    DensitySourceKind = "manual"
	DensitySourceEstimated DensitySourceKind = "estimated"
)

// Implements DESIGN-012 DataNormalizer stable warning vocabulary.
const (
	WarningMissingImage             = "missing_image"
	WarningMissingMacros            = "missing_macros"
	WarningMissingMicronutrients    = "missing_micronutrients"
	WarningMissingLiquidDensity     = "missing_liquid_density"
	WarningUncertainUnitConversion  = "uncertain_unit_conversion"
	WarningSuspiciousLiquidMacroSum = "suspicious_liquid_macros"
)

// Implements DESIGN-012 DataNormalizer bounded conversion constants.
const (
	gramsPerOunce             = 28.349523125
	millilitersPerFluidOunce  = 29.5735295625
	millilitersPerCup         = 236.5882365
	millilitersPerTablespoon  = 14.78676478125
	millilitersPerTeaspoon    = 4.92892159375
	maxExternalNutrientFields = 512
)

// NormalizationOptions supplies curator-owned state or density evidence.
// Imported density is normally derived from a trusted USDA portion; manual and
// estimated values remain explicit and are never synthesized by the normalizer.
// Implements DESIGN-012 DataNormalizer density provenance.
type NormalizationOptions struct {
	PhysicalState             repository.PhysicalState
	DensityGramsPerMilliliter float64
	DensitySourceKind         DensitySourceKind
}

// DataNormalizer reuses one active micronutrient snapshot for a normalization workflow.
// Implements DESIGN-012 DataNormalizer vocabulary-query boundary.
type DataNormalizer struct {
	vocabulary repository.MicronutrientVocabularyRepository
	telemetry  *observability.AdminExternalTelemetry
}

// WithTelemetry adds privacy-safe normalization warning observations.
// Implements DESIGN-014 MetricsCollector.
func (n *DataNormalizer) WithTelemetry(telemetry *observability.AdminExternalTelemetry) *DataNormalizer {
	if n != nil {
		n.telemetry = telemetry
	}
	return n
}

// NewDataNormalizer creates a workflow normalizer backed by the canonical vocabulary.
// Implements DESIGN-012 DataNormalizer.
func NewDataNormalizer(vocabulary repository.MicronutrientVocabularyRepository) *DataNormalizer {
	return &DataNormalizer{vocabulary: vocabulary}
}

// NormalizeRecords loads active vocabulary once and normalizes every provider record.
// Implements DESIGN-012 DataNormalizer without per-item full-vocabulary queries.
func (n *DataNormalizer) NormalizeRecords(ctx context.Context, records []ExternalFoodRecord) ([]NormalizedFoodCandidate, error) {
	if ctx == nil {
		return nil, errors.New("normalization context is required")
	}
	if n == nil || n.vocabulary == nil {
		return nil, errors.New("micronutrient vocabulary is required")
	}
	vocabulary, err := n.vocabulary.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]NormalizedFoodCandidate, 0, len(records))
	for _, record := range records {
		candidate, err := NormalizeExternalRecord(record, vocabulary)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// NormalizeRecordsWithWarnings drops malformed provider records while reusing one vocabulary snapshot.
// Implements DESIGN-009 ExternalSearchProxy bounded partial-result shaping and DESIGN-012 DataNormalizer.
func (n *DataNormalizer) NormalizeRecordsWithWarnings(ctx context.Context, records []ExternalFoodRecord) ([]NormalizedFoodCandidate, []ExternalDataWarning, error) {
	if ctx == nil {
		return nil, nil, errors.New("normalization context is required")
	}
	if n == nil || n.vocabulary == nil {
		return nil, nil, errors.New("micronutrient vocabulary is required")
	}
	vocabulary, err := n.vocabulary.ListActive(ctx)
	if err != nil {
		return nil, nil, err
	}
	candidates := make([]NormalizedFoodCandidate, 0, len(records))
	warnings := make([]ExternalDataWarning, 0)
	for _, record := range records {
		candidate, err := NormalizeExternalRecord(record, vocabulary)
		if err != nil {
			provider := boundedProvider(record.Provider)
			warnings = append(warnings, ExternalDataWarning{Provider: provider, Code: string(ProviderErrorInvalidPayload), Message: string(ProviderErrorInvalidPayload)})
			n.telemetry.NormalizationWarning(ctx, provider, string(ProviderErrorInvalidPayload))
			continue
		}
		for _, warning := range candidate.Warnings {
			n.telemetry.NormalizationWarning(ctx, boundedProvider(candidate.Provider), warning)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, warnings, nil
}

// boundedProvider keeps malformed record diagnostics in the closed provider vocabulary.
// Implements DESIGN-009 ExternalSearchProxy bounded warnings.
func boundedProvider(provider string) string {
	if provider == "usda" || provider == "openfoodfacts" {
		return provider
	}
	return "external"
}

// NormalizeExternalRecord converts one provider projection using a supplied vocabulary snapshot.
// Implements DESIGN-012 DataNormalizer NormalizeExternalRecord.
func NormalizeExternalRecord(record ExternalFoodRecord, vocabulary []repository.MicronutrientVocabularyEntry) (NormalizedFoodCandidate, error) {
	return NormalizeExternalRecordWithOptions(record, vocabulary, NormalizationOptions{})
}

// NormalizeExternalRecordWithOptions applies explicit curator state or density evidence.
// Implements DESIGN-012 DataNormalizer imported, manual, and estimated provenance.
func NormalizeExternalRecordWithOptions(record ExternalFoodRecord, vocabulary []repository.MicronutrientVocabularyEntry, options NormalizationOptions) (NormalizedFoodCandidate, error) {
	var err error
	record, err = validateExternalRecord(record)
	if err != nil {
		return NormalizedFoodCandidate{}, err
	}
	servingSize, servingUnit := canonicalQuantity(record.ServingSize, record.ServingUnit)
	packageSize, packageUnit := canonicalQuantity(record.PackageSize, record.PackageUnit)
	density, densityKind, densityProvider, densityFoodID, err := resolveDensity(record, options)
	if err != nil {
		return NormalizedFoodCandidate{}, err
	}
	state := options.PhysicalState
	if state == "" {
		state = inferPhysicalState(record, servingUnit, packageUnit, density)
	}
	if err := repository.ValidatePhysicalState(state); err != nil {
		return NormalizedFoodCandidate{}, err
	}
	if state == repository.PhysicalStateSolid && density > 0 {
		return NormalizedFoodCandidate{}, invalidNormalization()
	}

	candidate := NormalizedFoodCandidate{
		Provider: record.Provider, ExternalID: record.ExternalID, Name: record.Name,
		PhysicalState: state, ServingSize: servingSize, ServingUnit: servingUnit,
		PackageSize: packageSize, PackageUnit: packageUnit, DensityGramsPerMilliliter: density,
		DensitySourceProvider: densityProvider, DensitySourceFoodID: densityFoodID,
		DensitySourceKind: densityKind, Micros: repository.MicroValues{}, ImageURL: record.ImageURL,
	}
	if err := setServingMeasures(&candidate); err != nil {
		return NormalizedFoodCandidate{}, err
	}
	if record.ImageURL == "" {
		candidate.Warnings = append(candidate.Warnings, WarningMissingImage)
	}
	if state == repository.PhysicalStateLiquid && density == 0 {
		candidate.Warnings = append(candidate.Warnings, WarningMissingLiquidDensity)
	}

	macroFound := map[string]bool{}
	microRank := map[string]int{}
	macroRank := map[string]int{}
	keys := make([]string, 0, len(record.Nutrients))
	for key := range record.Nutrients {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		target, basis, ok := classifyNutrient(record.Provider, key)
		if !ok {
			continue
		}
		factor, rank, ok := per100Factor(basis, state, density, servingSize, servingUnit, packageSize, packageUnit)
		if !ok {
			candidate.Warnings = appendWarning(candidate.Warnings, WarningUncertainUnitConversion)
			continue
		}
		value := round4(record.Nutrients[key] * target.unitFactor * factor)
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return NormalizedFoodCandidate{}, invalidNormalization()
		}
		if target.macro != "" {
			if old, exists := macroRank[target.macro]; exists && old <= rank {
				continue
			}
			setMacro(&candidate.MacrosPer100, target.macro, value)
			macroFound[target.macro], macroRank[target.macro] = true, rank
			continue
		}
		if old, exists := microRank[target.micro]; exists && old <= rank {
			continue
		}
		candidate.Micros[target.micro], microRank[target.micro] = value, rank
	}
	if len(macroFound) != 3 {
		candidate.Warnings = append(candidate.Warnings, WarningMissingMacros)
	}
	if len(candidate.Micros) == 0 {
		candidate.Warnings = append(candidate.Warnings, WarningMissingMicronutrients)
	}
	if err := repository.ValidateMicronutrientKeys(candidate.Micros, vocabulary); err != nil {
		return NormalizedFoodCandidate{}, err
	}
	if err := repository.ValidateMacrosPer100(candidate.MacrosPer100, state); err != nil {
		return NormalizedFoodCandidate{}, err
	}
	if state == repository.PhysicalStateLiquid && candidate.MacrosPer100.Protein+candidate.MacrosPer100.Carbohydrates+candidate.MacrosPer100.Fat > 100 {
		candidate.Warnings = append(candidate.Warnings, WarningSuspiciousLiquidMacroSum)
	}
	return candidate, nil
}

// nutrientBasis identifies the provider quantity basis for one nutrient value.
// Implements DESIGN-012 DataNormalizer per-100, serving, and package conversion.
type nutrientBasis int

// Implements DESIGN-012 DataNormalizer nutrient quantity bases.
const (
	basisMass100 nutrientBasis = iota
	basisVolume100
	basisServing
	basisPackage
)

// nutrientTarget identifies one internal macro or canonical micronutrient and its unit scale.
// Implements DESIGN-012 DataNormalizer provider nutrient aliases.
type nutrientTarget struct {
	macro      string
	micro      string
	unitFactor float64
}

// validateExternalRecord revalidates provider projections at the normalization trust boundary.
// Implements DESIGN-012 DataNormalizer invalid external payload handling.
func validateExternalRecord(record ExternalFoodRecord) (ExternalFoodRecord, error) {
	provider, providerErr := security.NormalizeInput(security.InputFieldCurationProvider, record.Provider)
	externalID, idErr := security.NormalizeInput(security.InputFieldProviderIdentifier, record.ExternalID)
	name, nameErr := security.NormalizeInput(security.InputFieldProviderText, record.Name)
	image, imageErr := security.NormalizeInput(security.InputFieldImageURL, record.ImageURL)
	if providerErr != nil || idErr != nil || nameErr != nil || imageErr != nil || name.Value == "" || record.Nutrients == nil || len(record.Nutrients) > maxExternalNutrientFields {
		return ExternalFoodRecord{}, invalidNormalization()
	}
	for key, value := range record.Nutrients {
		if key == "" || utf8.RuneCountInString(key) > 128 || containsUnsafeProviderText(key) || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return ExternalFoodRecord{}, invalidNormalization()
		}
	}
	record.Provider, record.ExternalID, record.Name, record.ImageURL = provider.Value, externalID.Value, name.Value, image.Value
	return record, nil
}

// invalidNormalization returns the closed provider-safe payload error.
// Implements DESIGN-012 DataNormalizer invalid external payload handling.
func invalidNormalization() error {
	return &ProviderError{Code: ProviderErrorInvalidPayload}
}

// canonicalQuantity maps provider quantity aliases to the repository vocabulary.
// Implements DESIGN-012 DataNormalizer canonical unit aliases.
func canonicalQuantity(size *float64, unit string) (float64, string) {
	if size == nil || !finitePositive(*size) || unit == "" {
		return 0, ""
	}
	normalized, err := security.NormalizeInput(security.InputFieldServingUnit, unit)
	if err != nil {
		return 0, ""
	}
	return *size, normalized.Value
}

// inferPhysicalState selects the storage basis only from explicit provider evidence.
// Implements DESIGN-012 DataNormalizer physical-state inference.
func inferPhysicalState(record ExternalFoodRecord, servingUnit string, packageUnit string, density float64) repository.PhysicalState {
	if density > 0 || isVolumeUnit(servingUnit) || isVolumeUnit(packageUnit) {
		return repository.PhysicalStateLiquid
	}
	for key := range record.Nutrients {
		if strings.HasSuffix(strings.ToLower(key), "_100ml") {
			return repository.PhysicalStateLiquid
		}
	}
	return repository.PhysicalStateSolid
}

// resolveDensity validates curator evidence or derives imported USDA density.
// Implements DESIGN-012 DataNormalizer density provenance.
func resolveDensity(record ExternalFoodRecord, options NormalizationOptions) (float64, DensitySourceKind, string, string, error) {
	if options.DensityGramsPerMilliliter != 0 || options.DensitySourceKind != "" {
		if !finitePositive(options.DensityGramsPerMilliliter) {
			return 0, "", "", "", invalidNormalization()
		}
		switch options.DensitySourceKind {
		case DensitySourceManual, DensitySourceEstimated:
			return options.DensityGramsPerMilliliter, options.DensitySourceKind, "", "", nil
		case DensitySourceImported:
			return options.DensityGramsPerMilliliter, options.DensitySourceKind, record.Provider, record.ExternalID, nil
		default:
			return 0, "", "", "", invalidNormalization()
		}
	}
	if record.Provider != "usda" {
		return 0, "", "", "", nil
	}
	if density := trustedUSDADensity(record.Portions); density > 0 {
		return density, DensitySourceImported, record.Provider, record.ExternalID, nil
	}
	return 0, "", "", "", nil
}

// trustedUSDADensity selects the highest-priority measured USDA volume portion.
// Implements DESIGN-012 DataNormalizer USDA liquid density priority.
func trustedUSDADensity(portions []ExternalFoodPortion) float64 {
	bestRank, bestDensity := 99, 0.0
	for _, portion := range portions {
		milliliters, rank := volumeMilliliters(portion.Amount, portion.Unit)
		if rank >= bestRank || milliliters <= 0 || !finitePositive(portion.GramWeight) {
			continue
		}
		density := round4(portion.GramWeight / milliliters)
		if !finitePositive(density) {
			continue
		}
		bestRank, bestDensity = rank, density
	}
	return bestDensity
}

// volumeMilliliters converts a trusted USDA volume and returns its priority rank.
// Implements DESIGN-012 DataNormalizer USDA liquid density priority.
func volumeMilliliters(amount float64, unit string) (float64, int) {
	if !finitePositive(amount) {
		return 0, 99
	}
	switch normalizeVolumeAlias(unit) {
	case "ml":
		return amount, 0
	case "cup":
		return amount * millilitersPerCup, 1
	case "tbsp":
		return amount * millilitersPerTablespoon, 2
	case "tsp":
		return amount * millilitersPerTeaspoon, 3
	case "fl_oz":
		return amount * millilitersPerFluidOunce, 4
	default:
		return 0, 99
	}
}

// normalizeVolumeAlias canonicalizes only documented USDA density portion units.
// Implements DESIGN-012 DataNormalizer USDA liquid density aliases.
func normalizeVolumeAlias(unit string) string {
	unit = strings.ToLower(strings.TrimSpace(unit))
	unit = strings.NewReplacer(".", "", "_", " ", "-", " ").Replace(unit)
	switch unit {
	case "ml", "milliliter", "milliliters", "millilitre", "millilitres":
		return "ml"
	case "cup", "cups":
		return "cup"
	case "tbsp", "tablespoon", "tablespoons":
		return "tbsp"
	case "tsp", "teaspoon", "teaspoons":
		return "tsp"
	case "fl oz", "floz", "fluid ounce", "fluid ounces":
		return "fl_oz"
	default:
		return ""
	}
}

// setServingMeasures projects a canonical serving into repository serving metadata.
// Implements DESIGN-012 DataNormalizer serving conversion.
func setServingMeasures(candidate *NormalizedFoodCandidate) error {
	if candidate.ServingSize <= 0 {
		return nil
	}
	if candidate.PhysicalState == repository.PhysicalStateSolid {
		grams := candidate.ServingSize
		if candidate.ServingUnit == "oz" {
			grams *= gramsPerOunce
		} else if candidate.ServingUnit != "g" {
			return nil
		}
		value, ok := finiteRoundedMeasure(grams)
		if !ok {
			return invalidNormalization()
		}
		candidate.AverageUnitWeightGrams = value
		return nil
	}

	milliliters := candidate.ServingSize
	switch candidate.ServingUnit {
	case "fl_oz":
		milliliters *= millilitersPerFluidOunce
	case "g":
		if candidate.DensityGramsPerMilliliter == 0 {
			return nil
		}
		milliliters /= candidate.DensityGramsPerMilliliter
	case "oz":
		if candidate.DensityGramsPerMilliliter == 0 {
			return nil
		}
		milliliters = milliliters * gramsPerOunce / candidate.DensityGramsPerMilliliter
	case "ml":
	default:
		return nil
	}
	value, ok := finiteRoundedMeasure(milliliters)
	if !ok {
		return invalidNormalization()
	}
	candidate.AverageServingVolumeMilliliters = value
	return nil
}

// finiteRoundedMeasure prevents provider quantities from producing non-finite metadata.
// Implements DESIGN-012 DataNormalizer bounded serving conversion.
func finiteRoundedMeasure(value float64) (float64, bool) {
	if !finitePositive(value) {
		return 0, false
	}
	value = round4(value)
	return value, finitePositive(value)
}

// classifyNutrient dispatches provider-specific nutrient alias parsing.
// Implements DESIGN-012 DataNormalizer provider nutrient aliases.
func classifyNutrient(provider string, key string) (nutrientTarget, nutrientBasis, bool) {
	if provider == "usda" {
		return classifyUSDANutrient(key)
	}
	return classifyOpenFoodFactsNutrient(key)
}

// classifyUSDANutrient maps unit-qualified USDA names to internal nutrients.
// Implements DESIGN-012 DataNormalizer USDA nutrient aliases.
func classifyUSDANutrient(key string) (nutrientTarget, nutrientBasis, bool) {
	open := strings.LastIndex(key, " (")
	if open < 1 || !strings.HasSuffix(key, ")") {
		return nutrientTarget{}, 0, false
	}
	name := strings.ToLower(strings.TrimSpace(key[:open]))
	unit := strings.ToLower(strings.TrimSpace(key[open+2 : len(key)-1]))
	if macro := macroAlias(name); macro != "" && (unit == "g" || unit == "gram" || unit == "grams") {
		return nutrientTarget{macro: macro, unitFactor: 1}, basisMass100, true
	}
	micro := microAlias(name)
	if micro == "" {
		return nutrientTarget{}, 0, false
	}
	factor, ok := microUnitFactor(micro, unit)
	return nutrientTarget{micro: micro, unitFactor: factor}, basisMass100, ok
}

// classifyOpenFoodFactsNutrient maps normalized OpenFoodFacts fields and bases.
// Implements DESIGN-012 DataNormalizer OpenFoodFacts nutrient aliases.
func classifyOpenFoodFactsNutrient(key string) (nutrientTarget, nutrientBasis, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	basis, suffix, ok := openFoodFactsBasis(key)
	if !ok {
		return nutrientTarget{}, 0, false
	}
	name := strings.TrimSuffix(key, suffix)
	if macro := macroAlias(name); macro != "" {
		return nutrientTarget{macro: macro, unitFactor: 1}, basis, true
	}
	micro := microAlias(name)
	if micro == "" {
		return nutrientTarget{}, 0, false
	}
	factor := 1.0
	switch micro {
	case "Sodium", "Potassium", "Calcium", "Iron", "VitaminC":
		factor = 1000
	case "VitaminD":
		factor = 1000000
	}
	return nutrientTarget{micro: micro, unitFactor: factor}, basis, true
}

// openFoodFactsBasis parses supported per-100, serving, and package suffixes.
// Implements DESIGN-012 DataNormalizer OpenFoodFacts quantity bases.
func openFoodFactsBasis(key string) (nutrientBasis, string, bool) {
	for _, item := range []struct {
		suffix string
		basis  nutrientBasis
	}{{"_100g", basisMass100}, {"_100ml", basisVolume100}, {"_serving", basisServing}, {"_package", basisPackage}} {
		if strings.HasSuffix(key, item.suffix) {
			return item.basis, item.suffix, true
		}
	}
	return 0, "", false
}

// macroAlias maps provider macro names to internal fields.
// Implements DESIGN-012 DataNormalizer provider nutrient aliases.
func macroAlias(name string) string {
	name = normalizedNutrientName(name)
	switch name {
	case "protein", "proteins":
		return "protein"
	case "carbohydrate", "carbohydrates", "carbohydrate by difference":
		return "carbohydrates"
	case "fat", "fats", "total fat", "total lipid fat":
		return "fat"
	default:
		return ""
	}
}

// microAlias maps provider micronutrient names to canonical vocabulary keys.
// Implements DESIGN-012 DataNormalizer provider nutrient aliases.
func microAlias(name string) string {
	name = normalizedNutrientName(name)
	switch name {
	case "sodium", "sodium na":
		return "Sodium"
	case "potassium", "potassium k":
		return "Potassium"
	case "calcium", "calcium ca":
		return "Calcium"
	case "iron", "iron fe":
		return "Iron"
	case "vitamin c", "vitamin c total ascorbic acid":
		return "VitaminC"
	case "vitamin d", "vitamin d d2 + d3":
		return "VitaminD"
	case "fiber", "fibre", "fiber total dietary", "dietary fiber":
		return "Fiber"
	case "sugar", "sugars", "sugars total including nlea", "sugars total":
		return "Sugar"
	default:
		return ""
	}
}

// normalizedNutrientName produces a comparison-only provider alias token.
// Implements DESIGN-012 DataNormalizer provider nutrient aliases.
func normalizedNutrientName(name string) string {
	name = strings.NewReplacer("_", " ", "-", " ", ",", " ", "(", " ", ")", " ").Replace(strings.ToLower(name))
	return strings.Join(strings.Fields(name), " ")
}

// microUnitFactor converts USDA nutrient units to canonical vocabulary units.
// Implements DESIGN-012 DataNormalizer micronutrient unit conversion.
func microUnitFactor(micro string, unit string) (float64, bool) {
	unit = strings.NewReplacer("µ", "u", "μ", "u").Replace(strings.ToLower(unit))
	want := "mg"
	if micro == "VitaminD" {
		want = "ug"
	}
	if micro == "Fiber" || micro == "Sugar" {
		want = "g"
	}
	switch unit + ">" + want {
	case "g>g", "mg>mg", "ug>ug", "mcg>ug":
		return 1, true
	case "g>mg":
		return 1000, true
	case "mg>g":
		return 0.001, true
	case "mg>ug":
		return 1000, true
	case "ug>mg", "mcg>mg":
		return 0.001, true
	default:
		return 0, false
	}
}

// per100Factor selects the conversion to the physical state's per-100 basis.
// Implements DESIGN-012 DataNormalizer per-100 conversion.
func per100Factor(basis nutrientBasis, state repository.PhysicalState, density float64, servingSize float64, servingUnit string, packageSize float64, packageUnit string) (float64, int, bool) {
	switch basis {
	case basisMass100:
		if state == repository.PhysicalStateSolid {
			return 1, 0, true
		}
		return density, 0, density > 0
	case basisVolume100:
		return 1, 0, state == repository.PhysicalStateLiquid
	case basisServing:
		factor, ok := quantityPer100Factor(state, density, servingSize, servingUnit)
		return factor, 1, ok
	case basisPackage:
		factor, ok := quantityPer100Factor(state, density, packageSize, packageUnit)
		return factor, 2, ok
	default:
		return 0, 0, false
	}
}

// quantityPer100Factor converts one canonical serving or package quantity.
// Implements DESIGN-012 DataNormalizer serving and package conversion.
func quantityPer100Factor(state repository.PhysicalState, density float64, size float64, unit string) (float64, bool) {
	if !finitePositive(size) {
		return 0, false
	}
	switch unit {
	case "g":
		if state == repository.PhysicalStateSolid {
			return 100 / size, true
		}
		return 100 * density / size, density > 0
	case "oz":
		return quantityPer100Factor(state, density, size*gramsPerOunce, "g")
	case "ml":
		if state == repository.PhysicalStateLiquid {
			return 100 / size, true
		}
	case "fl_oz":
		return quantityPer100Factor(state, density, size*millilitersPerFluidOunce, "ml")
	}
	return 0, false
}

// isVolumeUnit reports whether a canonical unit is volume-based.
// Implements DESIGN-012 DataNormalizer physical-state inference.
func isVolumeUnit(unit string) bool { return unit == "ml" || unit == "fl_oz" }

// setMacro assigns one classified provider value to its internal macro field.
// Implements DESIGN-012 DataNormalizer macro mapping.
func setMacro(values *repository.MacroValues, key string, value float64) {
	switch key {
	case "protein":
		values.Protein = value
	case "carbohydrates":
		values.Carbohydrates = value
	case "fat":
		values.Fat = value
	}
}

// appendWarning adds one stable warning without duplicates.
// Implements DESIGN-012 DataNormalizer normalization warnings.
func appendWarning(warnings []string, warning string) []string {
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}

// round4 matches repository nutrition precision.
// Implements DESIGN-012 DataNormalizer nutrient conversion precision.
func round4(value float64) float64 { return math.Round(value*10000) / 10000 }
