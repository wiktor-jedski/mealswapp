# Task 246 Preparation — External Food Data Normalization

## Outcome and task control

- Task: **246 — Phase 08 External Food Data Normalization**.
- Result: **review repair complete; every task-owned verification clause has direct passing test evidence. Repository-wide full/race, traceability, and aggregate gates remain blocked by out-of-scope task 240/247 changes recorded below**.
- Fixed repository reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Preparation date: 2026-07-21, Europe/Warsaw.
- Dependencies 24, 31, 32, 34, 43, 242, 243, and 244 were re-read from `docs/implementation/02_TASK_LIST.md` and remain `PASSED`.
- Task 246 was observed as `PREPARED` before and after this repair. No task-list content or status was edited.
- Repair baseline and final task-list SHA-256: `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8`.
- The phase-orchestrator skill requires delegation when a writable subagent is available. No writable subagent capability was available in this session, so the parent executed the one-task preparation contract directly.

## Baseline and scope ownership

- The initial dirty worktree contained prior Phase 08 work and unrelated tracked/untracked changes across API, application, cache, curation, custom-item, deletion, HTTP, repository, search, security, userdata, migrations, frontend, scripts, design, implementation, and review paths. None was cleaned, reverted, staged, or reformatted outside the external-data package.
- Initial candidate hashes:
  - `backend/internal/externaldata/usda.go`: `21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb`.
  - `backend/internal/externaldata/usda_test.go`: `b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25`.
  - `backend/internal/externaldata/openfoodfacts.go`: `433733998bf73d63dd7dce66152ff24bb6ffb3cd4b80817d595b78949758b53d`.
  - `backend/internal/externaldata/openfoodfacts_test.go`: `c13dbb6309a8041476a5d32b03db432ac8b2d8a6863c84897d07f9a977ea98b4`.
  - `normalizer.go` and `normalizer_test.go`: absent.
- Task 245's `rate_limit.go` appeared concurrently after the initial baseline. Its first observed SHA-256 was `2c900e15ced99b497fde9eb98c3ee3a6b3a2c34f876a8d440d15024951d3c609`. Task 246 changed only the shared `NormalizedFoodCandidate` declaration and required repository import there; concurrent Task 245 changes to rate-limit configuration, orchestration, and tests remain Task 245-owned.
- Scope intentionally excludes provider retry/rate-limit behavior, admin authorization/routes, import persistence, curation audit, UI, and later workflow composition.

## Review repair

- Addressed both important findings in `docs/implementation/reviews/task-246-review.md`.
- `setServingMeasures` is now physical-state-aware: solid `g`/`oz` servings become unit weight; liquid `ml`/`fl_oz` servings become volume; and liquid `g`/`oz` servings divide converted grams by trusted positive density before becoming volume.
- Liquid mass servings without density remain safely incomplete and emit `missing_liquid_density`; no gram-to-milliliter equivalence is guessed.
- Every serving metadata conversion is checked before and after four-decimal rounding. Non-finite or rounded-overflow results fail closed with the structured `invalid_external_payload` provider error, so no candidate can expose `NaN` or `Inf` serving metadata.
- Adversarial tests cover liquid `g` and `oz` servings with trusted density, the same inputs without density, liquid `g`/`oz` package scaling with and without density, and `math.MaxFloat64` `oz`/`fl_oz` serving overflow.

## Sources and existing boundaries inspected

- `docs/design/DESIGN-012.md`: `DataNormalizer`, provider records, candidates, nutrient conversion, warnings, and invalid-payload handling.
- `docs/design/DESIGN-005.md`: repository macro/micro types, physical-state storage basis, canonical units, active micronutrient keys, serving metadata, and density provenance.
- `docs/implementation/01_PLAN.md` and `docs/implementation/04_OPEN.md`: canonical `g`/`ml`/`oz`/`fl_oz`/`serving`, warning-only suspicious liquid totals, no gram/milliliter proxy, density priority `ml -> cup -> tbsp -> tsp -> fl_oz`, and one reused active-vocabulary snapshot.
- `backend/internal/externaldata/usda.go`, `openfoodfacts.go`, and their tests: provider projection formats, USDA unit-qualified nutrient keys and measured portions, OpenFoodFacts suffix-based nutriments, and provider trust bounds.
- `backend/internal/security/normalizer.go` and curation validation: provider identity/text/image validation and serving-unit alias canonicalization.
- `backend/internal/repository/macros.go`, `units.go`, `types.go`, and micronutrient repository: per-100 invariants, canonical vocabulary validation, unit constants, and `ListActive`/`IsAllowed` query behavior.
- Seed migration `database/migrations/000004_micronutrient_vocabulary.up.sql`: active canonical micronutrient keys and units.

## Security assessment

The installed `golang-security` skill was applied in coding mode because all record fields and nutrient maps originate outside the trust boundary.

- Provider, identifier, display text, image URL, map size, key length/controls, and numeric finiteness are revalidated before normalization; the normalizer does not rely solely on HTTP-client projection.
- Conversion loops are bounded to 512 projected nutrient fields. Nutrient and serving-metadata results are checked after multiplication/division and rounding so overflow cannot reach repository-shaped output.
- Only allow-listed provider aliases become internal nutrients. Recognized micronutrients must resolve to an active canonical snapshot key; unrecognized provider fields such as energy are ignored, while recognized-but-inactive/absent canonical keys fail closed.
- No raw provider payload is logged or persisted. Errors remain categorical and warnings use a closed stable vocabulary.
- Density is accepted only when finite and positive. Cross-basis conversion requires measured or explicit density; no `1 ml = 1 g` fallback exists.
- No SQL, command execution, filesystem I/O, secrets, authentication state, cookies, PII, goroutines, locks, or mutable package-global state were added.

## Exact changed paths and symbols

| Path | Task 246 surface |
| --- | --- |
| `backend/internal/externaldata/normalizer.go` | New `DataNormalizer`, repository-shaped normalization, aliases, unit/basis conversion, density priority/provenance, warnings, trust-boundary checks, and vocabulary snapshot reuse. |
| `backend/internal/externaldata/normalizer_test.go` | New unit and repository-interface-backed acceptance, defensive, race, query-count, and 100%-normalizer-coverage tests. |
| `backend/internal/externaldata/usda.go` | Extended `ExternalFoodRecord` with optional package quantity/unit inputs consumed by normalization. |
| `backend/internal/externaldata/openfoodfacts.go` | Requested/projected bounded package quantity fields and shared serving/package canonicalization helper. |
| `backend/internal/externaldata/openfoodfacts_test.go` | Extended deterministic provider projection fixture/assertion with package quantity and alias canonicalization. |
| `backend/internal/externaldata/rate_limit.go` | Extended the concurrently introduced shared `NormalizedFoodCandidate` with repository-shaped normalized fields; retained Task 245's raw `Nutrients` bridge field unchanged. |
| `docs/implementation/preparations/task-246.md` | Baseline, scope, source, security, symbol, acceptance, command, and hash evidence. |

Production declarations added in `normalizer.go`:

- Types: `DensitySourceKind`, `NormalizationOptions`, `DataNormalizer`, `nutrientBasis`, and `nutrientTarget`.
- Constants: `DensitySourceImported`, `DensitySourceManual`, `DensitySourceEstimated`; all six `Warning*` values; conversion constants and `maxExternalNutrientFields`; all four nutrient-basis constants.
- Exported behavior: `NewDataNormalizer`, `(*DataNormalizer).NormalizeRecords`, `NormalizeExternalRecord`, and `NormalizeExternalRecordWithOptions`.
- Private behavior: `validateExternalRecord`, `invalidNormalization`, `canonicalQuantity`, `inferPhysicalState`, `resolveDensity`, `trustedUSDADensity`, `volumeMilliliters`, `normalizeVolumeAlias`, `setServingMeasures`, `classifyNutrient`, `classifyUSDANutrient`, `classifyOpenFoodFactsNutrient`, `openFoodFactsBasis`, `macroAlias`, `microAlias`, `normalizedNutrientName`, `microUnitFactor`, `per100Factor`, `quantityPer100Factor`, `isVolumeUnit`, `setMacro`, `appendWarning`, and `round4`.
- Review-repair private behavior: `setServingMeasures` now returns a structured conversion error, and `finiteRoundedMeasure` enforces finite positive metadata before and after rounding.

Existing production declarations modified:

- `ExternalFoodRecord`: added `PackageSize` and `PackageUnit`.
- `openFoodFactsFields`: added `product_quantity` and `product_quantity_unit` to the bounded provider projection.
- `openFoodFactsProduct`: added typed raw package quantity/unit fields.
- `projectOpenFoodFactsProduct`: now canonicalizes and projects serving and package quantities through `projectOpenFoodFactsQuantity`.
- `projectOpenFoodFactsQuantity`: added as the shared optional quantity-pair projector.
- `NormalizedFoodCandidate`: added physical state, canonical serving/package data, repository serving measures, density/provenance, macros, and micros. The concurrent Task 245 raw `Nutrients` bridge remains present.

Test declarations added in `normalizer_test.go`:

- Tests: `TestNormalizeUSDAAliasesAndTrustedDensityPriority`, `TestTrustedUSDADensityAcceptsEveryDocumentedVolumeAlias`, `TestNormalizeOpenFoodFactsPer100MillilitersWarnsButDoesNotRejectSuspiciousTotals`, `TestNormalizeServingAndPackageValuesUsesCanonicalUnitAliases`, `TestNormalizeNeverAssumesOneMilliliterEqualsOneGram`, `TestNormalizeDensityProvenanceOptions`, `TestNormalizeEmitsMissingWarningsAndRejectsUnknownCanonicalMicronutrients`, `TestDataNormalizerLoadsOneVocabularySnapshotPerWorkflow`, `TestNormalizeRejectsMalformedRecordsAndDensityOptions`, and `TestNormalizerCoversDefensiveConversionBranches`.
- Review-repair tests: `TestNormalizeLiquidMassServingUsesTrustedDensity`, `TestNormalizeLiquidMassServingWithoutDensityStaysIncomplete`, `TestNormalizeLiquidMassPackageUsesDensityOrWarns`, and `TestNormalizeRejectsServingMetadataOverflow`.
- Repository test double: `countingVocabulary`, `(*countingVocabulary).ListActive`, `(*countingVocabulary).IsAllowed`, and `(*countingVocabulary).Upsert`.
- Helpers: `activeVocabulary`, `solidUSDAMacros`, `oversizedNutrientMap`, `float64Pointer`, and `hasWarning`.
- Existing test/fixture modified: `TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically` and `validOpenFoodFactsPayload`.

## Verification criteria

| Criterion | Direct evidence | Result |
| --- | --- | --- |
| Provider nutrient aliases | USDA macro/micro names with element suffixes and OpenFoodFacts plural/hyphen aliases map deterministically to repository macros and seeded canonical micros. | PASS |
| Per-100g/per-100ml | Solid per-100g remains mass-based; liquid per-100ml remains volume-based; USDA per-100g liquid nutrients use measured density. | PASS |
| Serving/package conversion | OpenFoodFacts `_serving` and `_package` values scale through actual canonical quantities; liquid mass quantities require trusted density; liquid `g`/`oz` serving metadata is stored as volume; no basis-free value is guessed. | PASS |
| Canonical unit aliases | Shared security normalization maps grams/millilitres/ounces/fluid ounces/portion to `g`/`ml`/`oz`/`fl_oz`/`serving`; provider package projection is covered. | PASS |
| Density priority | Mixed portions prove `ml` wins over `cup`, `tbsp`, `tsp`, and `fl_oz`; each documented alias family independently derives the expected density. | PASS |
| Provenance | Derived USDA density is `imported` with provider/food ID; explicit `manual` and `estimated` density omit provider identity; explicit imported evidence retains it. | PASS |
| No silent `1 ml = 1 g` | A liquid with only per-100g nutrients and no density returns zero unconvertible macros plus density/unit/missing warnings. | PASS |
| Suspicious liquid warning | A 110 g macro sum per 100 ml returns a candidate with `suspicious_liquid_macros`; repository liquid validation does not reject it. | PASS |
| Missing warnings | Missing image, macros, micros, liquid density, and uncertain conversion each have direct stable-warning assertions. | PASS |
| Unknown micronutrient rejection | Recognized provider Sodium is rejected with `ErrorKindInvalidMicronutrientKey` when Sodium is absent from the active snapshot. | PASS |
| No per-item full-vocabulary load | Three records normalize with exactly one `ListActive` call and zero `IsAllowed` calls; query failure and nil boundaries are covered. | PASS |
| Bounded serving metadata | `math.MaxFloat64` `oz` and `fl_oz` servings return structured `invalid_external_payload` errors and never expose non-finite candidate metadata. | PASS |

## Commands and results

| Command | Result |
| --- | --- |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=/tmp/task-246-repair.cover ./internal/externaldata` | PASS; external-data package race clean at 100.0% statements; every function in `normalizer.go` is 100.0%. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | FAIL outside task 246: `internal/app/TestTask240CustomItemErasureIntegration` reports that transactional cleanup left 2 owner custom items. `internal/externaldata` passes. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | FAIL at the same out-of-scope task-240 integration assertion; `internal/externaldata` passes and no race report is emitted. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: no vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; Task 246 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | FAIL outside task 246: concurrent task-247 declaration `adminAuditSnapshotRule` in `backend/internal/repository/compliance_repository.go:545` lacks its required doc and adjacent design traceability comments. Task-246 declarations remain traced. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | FAIL immediately at the same out-of-scope task-247 traceability defect; later aggregate stages do not run. |

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/externaldata/normalizer.go` | `08bc5afc680300e46d83c4e9f2d59d2aba33f470ff7390fedec503693db78ec4` |
| `backend/internal/externaldata/normalizer_test.go` | `a615b5dacb7635f769e279e979c618655b6e22df288f4d07abc5f331f6b2e8b9` |
| `backend/internal/externaldata/usda.go` | `78b198fba01bd7da8ff665b401e5b95b1fa64fdd86fc3df030e9c63864aac116` |
| `backend/internal/externaldata/openfoodfacts.go` | `e839c6594ab265a11c8ab1b7bb2d295911696a86adc08326bf789e9ba1e0ef57` |
| `backend/internal/externaldata/openfoodfacts_test.go` | `c41107a76b9651fccb98d4eead9d579cd6d4080923715a756cc8e7e8a6c06742` |
| `backend/internal/externaldata/rate_limit.go` | `d7649887ac6fb8c960a2df2734f33ca842fc680cce94722a150a52d31a08660b` |
| `docs/implementation/02_TASK_LIST.md` (unchanged control) | `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8` |

This evidence document's own final digest is reported in the handoff because embedding it here would change that digest.

## Risks and handoff

- Task 246 has no unresolved task-owned acceptance, security, race, static-analysis, or coverage blocker. Repository-wide acceptance remains blocked by the task-240 integration assertion and task-247 traceability defect listed in command evidence.
- `SearchExternalFoods` still carries Task 245's raw projected `Nutrients` bridge; composing provider orchestration with `DataNormalizer` requires a vocabulary dependency and belongs to the later curation/import workflow rather than this isolated normalization task.
- OpenFoodFacts package conversion depends on explicit `product_quantity` plus a supported unit. Missing or unsupported package evidence produces an incomplete candidate warning; it is never guessed.
- Provider-specific micronutrient aliases are intentionally limited to the active seeded internal surface. Expanding the vocabulary requires adding the canonical entry and an explicit alias/unit mapping with tests.
- Independent review should recompute every hash above and inspect all listed declarations. Task-list status must remain untouched by preparation.
