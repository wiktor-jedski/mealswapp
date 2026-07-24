# Review Evidence: Task 246 — DataNormalizer

```yaml
task_id: 246
component: DataNormalizer
static_aspect: DESIGN-012 provider nutrient, unit, serving, package, density, provenance, warning, and vocabulary normalization
input_status: PREPARED
review_decision: PASSED
decision: PASSED
reviewed_at_utc: 2026-07-21
review_agent: Codex GPT-5 independent re-reviewer
baseline_ref: 81ca40ce00cb667ea29243ed2d34068e11229a69 plus task 245 PASSED evidence and repaired task 246 preparation
baseline_confidence: MEDIUM
code_review_skill_invoked: true
relevant_language_guide: Go, data normalization contracts, security review, error handling, and bounded input guidance
pre_review_gates_passed: true
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
blocking_findings: 0
important_findings: 0
optional_findings: 2
```

## 1. Task Source

Task 246 is the exact `PREPARED` row in `docs/implementation/02_TASK_LIST.md`, titled Phase 08 External Food Data Normalization and covering the DESIGN-012 `DataNormalizer` static aspect. The full reviewer template, current task preparation, prior task-246 review, task-245 PASSED evidence, designs, architectures, stack, implementation plan, and open decisions were read.

This is an independent re-review of the repaired task. Production code and task-list content/status were not edited. The template requests merging the main branch, but no merge was performed because the user required a non-mutating review and the shared worktree already contains unrelated dirty Phase 08 work. That boundary is recorded rather than changing the review baseline.

The repaired task claims physical-state-aware liquid serving metadata, density-backed liquid mass conversion, safe missing-density behavior, finite serving-metadata overflow rejection, and direct passing evidence for all earlier normalization/provider criteria.

## 2. Pre-Review Gates

| Gate | Result | Evidence |
|---|---|---|
| Task sequence | PASS | Task 245 is `PASSED`; task 246 is exactly `PREPARED`; task 247 is `PREPARED`; task 248 is `OPEN`. |
| Task-list validation | PASS | `python3 scripts/validate-task-list.py` — 263 sequential tasks and ordered dependencies. |
| Traceability validation | PASS | `python3 scripts/validate-traceability.py`. |
| Review template | PASS | `docs/implementation/reviewer-prompt.md` read fully. |
| Review skill | PASS | `code-review-skill` was invoked exactly once for this re-review; its full Go and security references were read and applied. |
| Scope control | PASS | Only task-246 normalizer/provider/shared candidate behavior and its callers, tests, designs, and evidence were assessed. |

## 3. Review Baseline and Change Surface

The fixed repository reference is `81ca40ce00cb667ea29243ed2d34068e11229a69`; it has no `backend/internal/externaldata` tree. The task-owned diff was reconstructed from the repaired task preparation, task-245 PASSED evidence, current source, current tests, and current task ordering.

Task 246 owns the new `normalizer.go` and `normalizer_test.go` behavior. It extends the neutral `ExternalFoodRecord` with package quantity inputs, extends Open Food Facts projection with typed package fields, and extends the concurrently introduced `NormalizedFoodCandidate` with repository-shaped normalized fields. Task-245 provider retry/rate-limit behavior and its raw `Nutrients` bridge remain outside task-246 ownership. Task 248 remains the owner of composing `DataNormalizer` into external search results.

The repaired current hashes match the preparation for the new normalizer files. Shared provider/rate-limit files are still later-task-shared files, so current hashes—not stale preparation-era shared-file hashes—are the review authority.

## 4. Acceptance Criteria Checklist

| Criterion | Result | Evidence |
|---|---|---|
| Provider nutrient aliases | PASS | USDA unit-qualified aliases and Open Food Facts plural/hyphen aliases map deterministically to canonical macros and seeded micronutrients. |
| Per-100g conversion | PASS | Solid `_100g` values remain mass-based; liquid `_100g` values require density and convert to the liquid per-100ml basis. |
| Per-100ml conversion | PASS | Liquid `_100ml` values remain volume-based; direct test covers macros and micronutrient unit factors. |
| Serving conversion | PASS | Solid `g`/`oz` servings produce finite unit weight; liquid `ml`/`fl_oz` produce finite volume; liquid `g`/`oz` divide converted grams by trusted positive density. |
| Package conversion | PASS | `_package` values scale through canonical package quantities; liquid mass package values require density and otherwise remain incomplete with warnings. |
| Canonical unit aliases | PASS | Security normalization maps documented gram, milliliter, ounce, fluid-ounce, and serving aliases to `g`, `ml`, `oz`, `fl_oz`, and `serving`. |
| USDA density priority | PASS | Measured portion priority is `ml`, `cup`, `tbsp`, `tsp`, then `fl_oz`; all documented aliases are tested. |
| USDA density provenance | PASS | Trusted USDA density is `imported` with provider/food ID; manual and estimated options omit provider identity; explicit imported evidence retains it. |
| No `1 ml = 1 g` assumption | PASS | Missing density does not synthesize mass/volume equivalence; conversion remains incomplete and warns. |
| Warnings versus rejection | PASS | Suspicious liquid macro totals warn; malformed records, invalid options, unknown active micronutrient keys, solid macro overflow, and serving metadata overflow reject. |
| Missing data | PASS | Missing image, macros, micros, liquid density, and uncertain conversion have stable warning assertions. |
| Unknown micronutrients | PASS | A recognized provider micronutrient is rejected when its canonical key is absent/inactive in the supplied vocabulary snapshot; unknown provider fields are ignored. |
| Vocabulary snapshot/query count | PASS | `NormalizeRecords` calls `ListActive` once for three records and performs zero per-item `IsAllowed` calls. Query failure and nil boundaries are covered. |
| Bounded inputs | PASS | Provider pages/body reads and projected nutrient maps are bounded; keys, text, numeric values, and optional quantities are revalidated at the normalizer trust boundary. |
| Finite overflow behavior | PASS | `math.MaxFloat64` `oz` and `fl_oz` serving metadata return `invalid_external_payload`; no candidate metadata contains `NaN` or `Inf`. |
| No raw provider payload in normalization | PASS | Provider decoders discard raw payloads; normalizer construction copies canonical fields only and does not copy `RawPayload` or raw nutrient maps. |
| Caller/test/design alignment | PASS | USDA/Open Food Facts clients, rate-limit caller boundary, normalizer tests, repository vocabulary contract, DESIGN-012, DESIGN-005, ARCH-012, and ARCH-005 were audited. |

## 5. Changed-Symbol Inventory

The 27 rows below group related declarations only for readability. Every current normalizer, provider, shared candidate, caller, test, repository vocabulary boundary, and design/evidence source in the reviewed surface is named or explicitly enumerated.

| # | Unit and symbols | Surface |
|---:|---|---|
| 1 | `DensitySourceKind`, `DensitySourceImported`, `DensitySourceManual`, `DensitySourceEstimated`, all `Warning*` constants, conversion constants, `maxExternalNutrientFields` | Normalizer provenance, warning, and bounded-conversion vocabulary. |
| 2 | `NormalizationOptions`, `DataNormalizer`, `NewDataNormalizer`, `(*DataNormalizer).NormalizeRecords`, `NormalizeExternalRecord`, `NormalizeExternalRecordWithOptions` | Normalizer configuration, public entry points, and one-snapshot workflow boundary. |
| 3 | `nutrientBasis`, `basisMass100`, `basisVolume100`, `basisServing`, `basisPackage`, `nutrientTarget` | Provider basis and target classification. |
| 4 | `validateExternalRecord`, `invalidNormalization` | Trust-boundary validation and closed provider error construction. |
| 5 | `canonicalQuantity`, `inferPhysicalState` | Canonical serving/package aliases and solid/liquid inference. |
| 6 | `resolveDensity`, `trustedUSDADensity`, `volumeMilliliters`, `normalizeVolumeAlias` | USDA density priority, conversion, aliases, and provenance. |
| 7 | `setServingMeasures`, `finiteRoundedMeasure` | Repaired serving metadata conversion and finite overflow guard. |
| 8 | `classifyNutrient`, `classifyUSDANutrient` | Provider-neutral and USDA nutrient dispatch. |
| 9 | `classifyOpenFoodFactsNutrient`, `openFoodFactsBasis` | Open Food Facts nutrient dispatch and `_100g`/`_100ml`/`_serving`/`_package` bases. |
| 10 | `macroAlias`, `microAlias`, `normalizedNutrientName` | Canonical provider alias mapping. |
| 11 | `microUnitFactor`, `per100Factor`, `quantityPer100Factor`, `isVolumeUnit` | Micronutrient units and per-100/serving/package scaling. |
| 12 | `setMacro`, `appendWarning`, `round4` | Candidate assignment, deduplicated warnings, and precision. |
| 13 | `ExternalSearchQuery`, `ExternalFoodPortion`, `ExternalFoodRecord`, `ProviderErrorCode`, provider error constants, `ProviderError` | Neutral provider record and error contracts consumed by normalization. |
| 14 | `USDAConfig`, `USDAClient`, `LoadUSDAAPIKey`, `NewUSDAClient`, `(*USDAClient).Search`, `(*USDAClient).SearchResult`, `validateUSDAQuery` | USDA request/caller boundary. |
| 15 | `usdaSearchPayload`, `usdaFood`, `usdaNutrient`, `usdaMeasure`, `usdaMeasureUnit`, `decodeUSDASearch`, `finitePositive`, `mapUSDAStatus`, USDA transport/failure helpers | USDA bounded payload projection and safe diagnostics. |
| 16 | `OpenFoodFactsConfig`, `OpenFoodFactsClient`, `NewOpenFoodFactsClient`, `(*OpenFoodFactsClient).Search`, `(*OpenFoodFactsClient).SearchResult`, `validateOpenFoodFactsQuery` | Open Food Facts request/caller boundary. |
| 17 | `openFoodFactsSearchPayload`, `openFoodFactsProduct`, `decodeOpenFoodFactsSearch`, `projectOpenFoodFactsProduct`, `projectOpenFoodFactsQuantity`, `decodeJSONString`, `containsUnsafeProviderText`, `validCallerID`, Open Food Facts status/transport/failure helpers | Open Food Facts bounded projection and typed serving/package quantities. |
| 18 | `ProviderRateLimit`, `rateState`, `RateLimitHandler`, `ResultProvider`, `ProviderResult`, `projectRateLimitHeaders`, `ProviderSet`, `ExternalDataWarning` | Shared provider result and quota boundary from task 245. |
| 19 | `NormalizedFoodCandidate`, `SearchExternalFoods`, `validateExternalSearchQuery` | Shared candidate fields, adjacent raw bridge, and later normalizer-composition caller. |
| 20 | `TestNormalizeUSDAAliasesAndTrustedDensityPriority`, `TestTrustedUSDADensityAcceptsEveryDocumentedVolumeAlias`, `TestNormalizeOpenFoodFactsPer100MillilitersWarnsButDoesNotRejectSuspiciousTotals`, `TestNormalizeServingAndPackageValuesUsesCanonicalUnitAliases`, `TestNormalizeNeverAssumesOneMilliliterEqualsOneGram` | Core normalizer acceptance tests. |
| 21 | `TestNormalizeLiquidMassServingUsesTrustedDensity`, `TestNormalizeLiquidMassServingWithoutDensityStaysIncomplete`, `TestNormalizeLiquidMassPackageUsesDensityOrWarns`, `TestNormalizeRejectsServingMetadataOverflow` | Repaired liquid serving/package and finite-overflow regressions. |
| 22 | `TestNormalizeDensityProvenanceOptions`, `TestNormalizeEmitsMissingWarningsAndRejectsUnknownCanonicalMicronutrients`, `TestDataNormalizerLoadsOneVocabularySnapshotPerWorkflow`, `TestNormalizeRejectsMalformedRecordsAndDensityOptions`, `TestNormalizerCoversDefensiveConversionBranches` | Density, warnings, vocabulary, malformed-input, and defensive normalizer tests. |
| 23 | `countingVocabulary`, `ListActive`, `IsAllowed`, `Upsert`, `activeVocabulary`, `solidUSDAMacros`, `oversizedNutrientMap`, `float64Pointer`, `hasWarning` | Vocabulary spy and normalizer test helpers. |
| 24 | All USDA test declarations: `TestLoadUSDAAPIKey`, `TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically`, `TestUSDASearchRejectsInvalidInputBeforeOutboundRequest`, `TestUSDASearchHonorsDeadlineAndCallerCancellation`, `TestUSDASearchPreservesContextErrorsWhileReadingBody`, `TestUSDASearchDoesNotFollowCredentialBearingRedirects`, `TestUSDASearchBoundsResponseAndRejectsMalformedOrPartialPayloads`, `TestUSDASearchHandlesRequestTransportAndBodyReadFailures`, `TestDecodeUSDASearchAcceptsEmptyResultsAndOrdersPortionTies`, `TestUSDASearchMapsProviderStatusesAndLogsOnlyBoundedMetadata`, `TestUSDASearchResultProjectsBoundedHeadersOnSuccessAndFailure`, `TestNewUSDAClientRejectsUnsafeConfiguration` | USDA caller, payload, bounds, cancellation, redirect, and diagnostic tests. |
| 25 | All Open Food Facts test declarations: `TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically`, `TestOpenFoodFactsSearchRejectsInvalidInputBeforeOutboundRequest`, `TestOpenFoodFactsSearchHonorsDeadlineAndCallerCancellation`, `TestOpenFoodFactsSearchPreservesContextErrorsWhileReadingBody`, `TestOpenFoodFactsSearchPreservesTransportContextSentinels`, `TestOpenFoodFactsSearchBoundsBodiesAndHandlesMalformedOrPartialPayloads`, `TestOpenFoodFactsSearchEnforcesFiniteAllocationBound`, `TestProjectOpenFoodFactsProductHandlesOptionalAndMalformedFields`, `TestProjectOpenFoodFactsProductRejectsMalformedNumericNutriments`, `TestOpenFoodFactsSearchMapsStatusesAndLogsOnlyBoundedMetadata`, `TestOpenFoodFactsSearchResultProjectsBoundedHeadersOnSuccessAndFailure`, `TestOpenFoodFactsSearchHandlesRequestTransportAndBodyReadFailures`, `TestNewOpenFoodFactsClientRejectsUnsafeConfiguration` | Open Food Facts caller, quantity projection, bounds, payload, cancellation, redirect, and diagnostic tests. |
| 26 | All rate-limit/caller test declarations: `TestSearchExternalFoodsRecordsErrorHeadersAndSkipsUntilReset`, `TestSearchExternalFoodsPropagatesRealProviderHeadersOnErrorAndSuccess`, `TestSearchExternalFoodsPreservesInFlightCallerCancellation`, `TestSearchExternalFoodsPreservesInFlightCallerDeadline`, `TestSearchExternalFoodsRejectsInvalidInputWithoutProviderCalls`, `TestSearchExternalFoodsReportsMissingSelectedProviders`, `TestRateLimitHandlerAdversarialBranches`, `TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation`, `TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset`, `TestSearchExternalFoodsNonRetryableUnavailableIsWarningAndSingleCall`, `TestSearchExternalFoodsEmitsNoTelemetryAndDoesNotLeakPayloadOrSecrets`, `TestRateLimitHandlerDeterministicRetryDeadlineAndHeaderIsolation`, `TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings` | Shared caller, partial-success, retry, cancellation, warning, quota, and raw-bridge boundary tests. |
| 27 | `DESIGN-012.md`, `DESIGN-005.md`, `ARCH-012.md`, `ARCH-005.md`, `01_TECH_STACK.md`, `01_PLAN.md`, `04_OPEN.md`, repository `types.go`/`macros.go`/`units.go`/`vocabulary_repository.go`, vocabulary SQL, seed migration, security normalizer, reviewer template, task preparations, task-245 review, task list | Design, architecture, repository, security, vocabulary, preparation, and control evidence. |

inventory_source_count: 27

## 6. Function-Level Audit

| # | Audited unit | Result | Evidence |
|---:|---|---|---|
| 1 | Normalizer constants and provenance/warning vocabulary | PASS | Closed warning/source values and bounded conversion constants match the design. |
| 2 | Normalizer options, constructor, and public workflow functions | PASS | Nil context/vocabulary boundaries are safe; `ListActive` is loaded once and reused. |
| 3 | Nutrient basis and target classification | PASS | Mass, volume, serving, and package bases remain distinct and deterministic. |
| 4 | Record validation and provider-safe errors | PASS | Provider text, identity, image, key count/length/control characters, finite values, and nonnegative values are revalidated. |
| 5 | Quantity canonicalization and state inference | PASS | Supported aliases canonicalize; volume evidence, density, and `_100ml` evidence infer liquid without a mass proxy. |
| 6 | Density resolution, trusted USDA priority, and volume aliases | PASS | Explicit provenance wins; measured USDA portions use documented priority and finite positive density. |
| 7 | Repaired serving metadata conversion and finite guard | PASS | Liquid mass `g`/`oz` converts through density; no-density inputs stay incomplete; overflow returns `invalid_external_payload`. |
| 8 | Provider-neutral and USDA nutrient dispatch | PASS | USDA macro/micro names and units map to canonical targets. |
| 9 | Open Food Facts basis and dispatch | PASS | Supported suffixes and aliases map to the correct physical basis and unit factor. |
| 10 | Alias normalization | PASS | Canonical macro and seeded micro aliases are explicit; unknown fields do not become internal nutrients. |
| 11 | Unit and per-100 factors | PASS | Serving/package mass and volume conversion requires matching state and density where needed; nutrient results are finite-checked. |
| 12 | Candidate assignment, warnings, and rounding | PASS | Warning deduplication is stable; all emitted serving metadata is finite and positive when present. |
| 13 | Neutral provider records and errors | PASS | Optional package/portion evidence is typed; raw payload is not copied by normalizer construction. |
| 14 | USDA client/request boundary | PASS | Query, endpoint, body, redirect, deadline, cancellation, header, and secret boundaries pass existing tests. |
| 15 | USDA payload projection | PASS | Nutrients and portions are validated, deterministic, bounded, and suitable for density derivation. |
| 16 | Open Food Facts client/request boundary | PASS | Caller ID, query, endpoint, body, redirect, deadline, cancellation, and safe headers pass. |
| 17 | Open Food Facts projection | PASS | Typed serving/package quantities and nutrient maps are optional/validated and raw provider bytes are discarded. |
| 18 | Shared provider result/quota types | PASS | Task-245 result/header boundary remains bounded and independently checked. |
| 19 | Shared candidate and search caller | PASS | Task-245 raw bridge is identified as adjacent; task 248 is the documented composition owner. |
| 20 | Core normalizer acceptance tests | PASS | Aliases, per-100 conversion, canonical serving/package aliases, density priority, and no 1 ml = 1 g behavior pass. |
| 21 | Repaired regression tests | PASS | Trusted-density liquid mass serving, missing-density serving, package density/warnings, and `MaxFloat64` overflow tests pass. |
| 22 | Remaining normalizer tests | PASS | Provenance, missing data, unknown micronutrient rejection, snapshot query count, malformed inputs, and defensive branches pass. |
| 23 | Normalizer test doubles/helpers | PASS | Vocabulary spy demonstrates exactly one snapshot and zero `IsAllowed` calls; helpers are bounded and deterministic. |
| 24 | USDA test suite | PASS | All current USDA caller/projection tests pass under package race and coverage execution. |
| 25 | Open Food Facts test suite | PASS | All current Open Food Facts caller/projection tests pass under package race and coverage execution. |
| 26 | Rate-limit/search caller test suite | PASS | Existing task-245 retry/quota/cancellation/partial-success/raw-bridge tests pass under package race execution. |
| 27 | Design, repository, security, SQL, and evidence sources | PASS | Contracts require canonical units, physical-state basis, active vocabulary, no 1:1 proxy, and safe provider boundaries; current hashes are recorded. |

audited_symbol_count: 27

## 7. Findings

No blocking or important findings remain. The two prior important findings are repaired and directly regression-tested.

### Optional observations

1. `NormalizeRecords` caps projected nutrient fields per record but does not impose a separate maximum number of records supplied by a direct internal caller. USDA/Open Food Facts page and body bounds cap the normal provider path. A later workflow boundary may add a record-count cap if direct callers become externally reachable.
2. `SearchExternalFoods` still returns task-245's raw projected `Nutrients` bridge and does not yet call `DataNormalizer`; this is the documented handoff to OPEN task 248. Task 248 must compose the normalizer before exposing repository-shaped candidates, while task 246's normalizer itself does not copy raw fields.

blocking_findings: 0
important_findings: 0
optional_findings: 2

## 8. Commands Run

| Command | Result |
|---|---|
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=/tmp/task-246-rereview.cover ./internal/externaldata` | PASS; package race clean and 100.0% statement coverage. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=/tmp/task-246-rereview.cover` | PASS; every current external-data function and total report 100.0%. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -run '^(TestNormalize|TestTrusted|TestDataNormalizer|TestNormalizer)' ./internal/externaldata` | PASS; all focused normalizer and repaired regression tests. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | FAIL outside task 246: task-240 custom-item erasure assertion, deletionworker retry/metrics assertion, and repository migration bootstrap duplicate PostgreSQL type/key. `internal/externaldata` passes. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | FAIL outside task 246: task-240 custom-item erasure assertion; `internal/externaldata` passes and no race report appears. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no vulnerabilities found in called/imported code. |
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `gofmt -d backend/internal/externaldata/*.go` | PASS; no formatting diff. |
| `git diff --check` | PASS for the pre-document review worktree. |
| `python3 scripts/check.py` | FAIL outside task 246 during local-stack verification: existing local PostgreSQL migration-down state reports missing `saved_diet_meal_entries` (`SQLSTATE 42P01`). Earlier aggregate stages passed, including traceability, task list, Go doc, Redocly (one existing ignored 302-only warning), Python checks, vet, vulnerability scan, and selected Go tests. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-246-review.md` | PASS after this document was refreshed; structural evidence is valid. |

## 9. Files Inspected and Staleness Fingerprints

All reviewed normalizer/provider/shared candidate source, test, caller, design, architecture, repository, security, SQL, preparation, template, prior-review, and task-list files were hashed. Current hashes are authoritative for this dirty shared worktree.

| File | Current SHA-256 |
|---|---|
| `backend/internal/externaldata/normalizer.go` | `08bc5afc680300e46d83c4e9f2d59d2aba33f470ff7390fedec503693db78ec4` |
| `backend/internal/externaldata/normalizer_test.go` | `a615b5dacb7635f769e279e979c618655b6e22df288f4d07abc5f331f6b2e8b9` |
| `backend/internal/externaldata/usda.go` | `78b198fba01bd7da8ff665b401e5b95b1fa64fdd86fc3df030e9c63864aac116` |
| `backend/internal/externaldata/usda_test.go` | `d76cfc8a6ae122c8b7a182dcfb85a4c4567803ccf25d810db3addbe5d3364b58` |
| `backend/internal/externaldata/openfoodfacts.go` | `e839c6594ab265a11c8ab1b7bb2d295911696a86adc08326bf789e9ba1e0ef57` |
| `backend/internal/externaldata/openfoodfacts_test.go` | `c41107a76b9651fccb98d4eead9d579cd6d4080923715a756cc8e7e8a6c06742` |
| `backend/internal/externaldata/rate_limit.go` | `d7649887ac6fb8c960a2df2734f33ca842fc680cce94722a150a52d31a08660b` |
| `backend/internal/externaldata/rate_limit_test.go` | `f7ea7b2e01e041cbff3c40688e929214b2919669715009a96ff4f339536ef025` |
| `backend/internal/security/normalizer.go` | `f87732321090d144229227b4573cf5ff1155d80f95c4e68da44a513c55802607` |
| `backend/internal/repository/macros.go` | `fe08f2fe0a693b99b413153ca190ec1db0e40e2140bf8d79cdfd86c186381af2` |
| `backend/internal/repository/types.go` | `5534be37a865c95390f84687ed82007e0adbca63a94fbd8c7e849ccb8cc40ac6` |
| `backend/internal/repository/units.go` | `9d9a8296654cc4b57e13bfb0090f15dced673d85782abc39675d8e2967463127` |
| `backend/internal/repository/vocabulary_repository.go` | `c27715ce33cf4da3a3715f1d8019489dcdb5e81998321dfd2a9c1dd7d27153e6` |
| `backend/internal/repository/sql/vocabulary_list_active.sql` | `3c949bf8bd2cb92a504a4411da5dc6a272151eda0c876751ffec5bb5eda85950` |
| `backend/internal/repository/sql/vocabulary_is_allowed.sql` | `a8e61b7b6085958d5bd228d58076c4946a8940a935e5156c3e4129a9d7900fb7` |
| `backend/internal/repository/sql/vocabulary_upsert.sql` | `1e680a13728c84d3adbc2af1bab5ee1655041bd749175228650cfb34a3210eba` |
| `docs/design/DESIGN-012.md` | `53ac9bd6a34bd07216666d4beaae6533a0281c905fc2d5c474f48f614746eddf` |
| `docs/design/DESIGN-005.md` | `91e9f1e152554e5d6eb62093018d57464ac3d38ca2add217215281927f885d31` |
| `docs/design/01_TECH_STACK.md` | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/architecture/ARCH-012.md` | `8377243ff9409b27ac9c43de556f0ff094c163ca465abe562ee8cec5721ed435` |
| `docs/architecture/ARCH-005.md` | `fc1fc595ee41ef952bf6568d579110481ba282a1fad153d0a5b68bbb2174818e` |
| `docs/implementation/01_PLAN.md` | `59fef9bf6f8c1cf058533ab296e87d9264d091cbcf204b56a2ff6b8dbfa4ba1d` |
| `docs/implementation/04_OPEN.md` | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |
| `database/migrations/000004_micronutrient_vocabulary.up.sql` | `7564567ae0bee0f90302c02c61a57e4dd08276c65896e3113c8e6ebae74c1156` |
| `docs/implementation/reviewer-prompt.md` | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |
| `docs/implementation/preparations/task-246.md` | `b00be082177bc24f3add22dd18d3ceca14a12d915a4ecab127bcf18dc4cd3712` |
| `docs/implementation/preparations/task-245.md` | `b01bfc16819b4613b1bd65d78397493e280971bb832bf7b5bd5031e29a5f010c` |
| `docs/implementation/reviews/task-245-review.md` | `22424777517d81ff005ea9e2c6fc1b726ef2fa83b511970fcacae7b0524780d9` |
| `docs/implementation/02_TASK_LIST.md` | `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8` |

`all_reviewed_files_hashed: true`
`prior_evidence_checked_for_staleness: true`

## 10. Coverage and Exceptions

The focused external-data package run reports 100.0% statement coverage and 100.0% for every current normalizer/provider/rate-limit function. The repaired branches are behaviorally asserted, not merely reached: liquid mass serving conversion, missing density, liquid package scaling, and finite metadata overflow all have direct tests.

The full ordinary test run has unrelated failures in task-240 custom-item erasure, deletionworker metrics, and repository migration bootstrap state. The full race run has the unrelated task-240 integration assertion. The aggregate check reaches its local-stack gate and fails because the existing local migration-down database state lacks `saved_diet_meal_entries`. These are recorded exceptions; no out-of-scope code or local database was repaired.

`review_evidence_validator: PASS`
`coverage_exception_count: 0`

## 11. Negative and Regression Checks

- USDA macro/micro aliases, Open Food Facts aliases, per-100g, per-100ml, serving, and package bases were exercised.
- Canonical aliases for `g`, `ml`, `oz`, `fl_oz`, and `serving` were checked at the security/provider boundary and normalizer boundary.
- Trusted USDA density priority was tested with mixed portions and each documented volume alias.
- Repaired liquid `g` and `oz` serving values use density: 10 g at 1.2 g/ml becomes 8.3333 ml; 2 oz becomes 47.2492 ml.
- Liquid `g`/`oz` serving values without density remain zero-measure and emit `missing_liquid_density`; no 1 ml = 1 g fallback occurs.
- Liquid `g`/`oz` package nutrient scaling succeeds with density and emits uncertain/missing warnings without density.
- `math.MaxFloat64` `oz` and `fl_oz` serving values fail with the closed `invalid_external_payload` error before non-finite metadata can escape.
- Explicit manual, estimated, and imported density provenance was checked, including source identity rules.
- Suspicious liquid macro totals warn rather than reject; solid macro totals over 100 reject.
- Missing image/macros/micros/density and uncertain conversion warnings are stable and bounded.
- Recognized-but-inactive/absent canonical micronutrients reject; unknown provider fields such as energy do not become micros.
- Three records use exactly one active-vocabulary snapshot and zero `IsAllowed` calls.
- Provider page/body/key/nutrient-map bounds, malformed numeric values, invalid controls, redirects, cancellation, deadline, safe headers, and no raw provider payload logging were checked.
- Normalizer candidate construction does not copy raw payload or raw provider nutrient maps; the adjacent task-245 raw bridge is explicitly deferred to task 248 composition.

## 12. Decision

`PASSED` — no blocking or important findings remain for task 246.

The two prior important findings are repaired: liquid mass serving metadata is now converted through trusted positive density, and all serving metadata conversion/rounding paths fail closed on non-finite results. The repaired task-owned tests pass under race and full package coverage. The task-list row remains `PREPARED` because this re-review was explicitly forbidden from editing task-list status.

## 13. Repair Context

No further task-246 repair is required. The exact repairs verified by this re-review are:

1. `setServingMeasures` now maps solid mass to finite unit weight, liquid volume to finite serving volume, and liquid mass to volume only after conversion through trusted density.
2. Missing liquid density leaves mass-based liquid serving metadata incomplete and preserves the documented warning path.
3. `finiteRoundedMeasure` checks finite positive values before and after four-decimal rounding; overflow returns the structured invalid-payload error.
4. Tests cover both liquid mass units, density/no-density behavior, package scaling, and `math.MaxFloat64` overflow.

No production source or task-list edit was made during this re-review. The task-list SHA-256 remained `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8`.
