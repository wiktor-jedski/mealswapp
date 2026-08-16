# Task 242 Preparation — Curation Input Normalization

## Outcome and task control

- Task: **242 — Phase 08 Curation Input Normalization**.
- Repair result: **all five important findings in `docs/implementation/reviews/task-242-review.md` are repaired and regression-tested**.
- Fixed implementation reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Repair date: 2026-07-21, Europe/Warsaw.
- Dependency 115 remains `PASSED`.
- Task 242 remains `PREPARED`. This repair did not edit `docs/implementation/02_TASK_LIST.md` or change task status.
- Task-list unchanged-control SHA-256 during final repair verification: `7da6214eb5b778c8d5b63b5b2dd27808d2f38cebb81c872190afe46b3b7f1932`.
- Review evidence SHA-256 used as the repair source: `f66259757f8eef899ae844e6da81d415e0f05b4153a06d2d7a12b0e379442833`.
- Existing unrelated Tasks 238–241 and Phase 08 worktree changes were preserved.

## Design and security basis

- `docs/design/DESIGN-013.md`: typed normalization before persistence, `input_rejected` before dispatch, and metadata-only `normalized` logging.
- `docs/design/DESIGN-010.md`: strict request validation and structured validation failures.
- `docs/design/DESIGN-009.md`: typed external-search, item-curation, and classification dispatch boundaries.
- `docs/design/DESIGN-005.md`: macro invariants and persisted `numeric(12,4)` nutrition fields.
- `docs/design/DESIGN-004.md` and `api/openapi.yaml`: documented quantity ceiling of `1,000,000`.
- `backend/internal/security/normalizer.go`: existing page ceiling of `10,000`, text/provider/unit allowlists, and image URL policy.
- The `diagnose` workflow established failing Fiber and typed-normalizer regressions before repair. The `golang-security` guidance caused raw duplicate-preserving decoding, runtime log-metadata allowlists, generic client errors, request-context propagation, and adversarial malformed-input tests.

## Repair findings and implementation

| Review finding | Repair | Regression evidence |
| --- | --- | --- |
| Null macro object/scalars reached dispatch | Curation HTTP decoding now reads raw JSON, requires `macrosPer100` to be an object, and requires non-null `protein`, `carbohydrates`, and `fat` members before normalization. | HTTP tests reject null object, null member, missing/malformed macro data with structured 400 and unchanged dispatch counters. |
| Numeric fields lacked upper bounds | Added `MaxCurationNutritionValue = 99,999,999.9999` for every macro component and micronutrient value, retained page `10,000`, and added `MaxCurationServingQuantity = 1,000,000`. | Maximum, next-representable-above-maximum, extreme finite, and bounded downstream scaling tests. |
| `RejectionField` and log metadata were open | `allowedLogField` runtime-allowlists every non-PII field category; outcomes are restricted to `normalized` or `rejected`; unknown enum/field/outcome values are dropped. | Raw sentinel tests prove arbitrary rejection fields and outcomes produce no event. |
| HTTP discarded normalized typed values | Each curation validator is now Fiber middleware that normalizes once, stores a typed request in private Fiber locals, and exposes typed accessors for handlers. Request logging uses `ctx.UserContext()`. | Handlers assert canonical `apple`, `usda`, `Café au lait`, `ml`, `openfoodfacts`, `Cafe drink`, and `Fresh foods`. |
| Generic maps collapsed duplicate keys | Curation JSON is scanned recursively before typed decoding; duplicate object keys at any depth reject. Query args are visited before map validation and repeated decoded names reject. | Top-level JSON, nested macro JSON, classification JSON, and duplicate query-key requests all return structured 400 before dispatch. |

The safe image URL policy, provider/unit allowlists, Unicode normalization, control rejection, generic error envelopes, no-fetch scope, and optional micronutrient-map behavior are preserved. Shared `ValidateJSON`/`ValidateQuery`, routes, provider clients, persistence, migrations, authorization, OpenAPI, and task-list state were not changed for this repair.

## Changed paths and symbols

| Path | Task 242 surface after repair |
| --- | --- |
| `backend/go.mod` | Existing direct `golang.org/x/text v0.29.0` dependency for NFC normalization; unchanged by repair. |
| `backend/internal/security/normalizer.go` | Existing curation `InputField` dispatch and normalization helpers; unchanged by repair. |
| `backend/internal/security/curation_normalizer_test.go` | Extends adversarial coverage for malformed UTF-8, all canonical unit branches, Unicode hosts, and valid/invalid URL ports. |
| `backend/internal/curation/validation.go` | Adds `MaxCurationNutritionValue`, `MaxCurationServingQuantity`, `allowedLogField`, and `validMacrosWithinBounds`; updates `NormalizeItem`, `log`, and `validateMicronutrients`. |
| `backend/internal/curation/validation_test.go` | Adds `TestRecordRejectionDropsUnknownMetadata` and `TestInputNormalizerRejectsNumericValuesAboveDocumentedBounds`; covers all macro components and downstream scaling. |
| `backend/internal/httpapi/curation_validation.go` | Changes the three `Validate*` methods to Fiber middleware; adds three normalized typed accessors, raw strict decoding, required macro checks, recursive duplicate-key detection, generic error mapping, and request-context use. |
| `backend/internal/httpapi/curation_validation_test.go` | Mounts curation middleware directly and verifies normalized handler handoff, nulls, duplicates, malformed UTF-8/non-objects/trailing data, extreme finite values, and zero rejected dispatches. |
| `docs/implementation/preparations/task-242.md` | Refreshes repair scope, symbols, hashes, and final verification results. |

Key production symbols are:

- Curation: `MaxCurationNutritionValue`, `MaxCurationServingQuantity`, `RecordRejection`, `NormalizeExternalSearch`, `NormalizeItem`, `NormalizeClassification`, `allowedLogField`, `validMacrosWithinBounds`, `validateMicronutrients`.
- HTTP: `ValidateExternalSearchQuery`, `ValidateItemBody`, `ValidateClassificationBody`, `NormalizedExternalSearchRequest`, `NormalizedCurationItemRequest`, `NormalizedCurationClassificationRequest`, `decodeStrictBody`, `validateRequiredMacros`, `rejectDuplicateJSONKeys`, `scanJSONValue`.
- New regressions: `TestRecordRejectionDropsUnknownMetadata`, `TestInputNormalizerRejectsNumericValuesAboveDocumentedBounds`; expanded `TestCurationHTTPValidationStopsBeforeProviderOrRepositoryDispatch`, `TestCurationTextNormalizationBoundaries`, `TestCurationProviderIdentifierAndUnitNormalization`, and `TestCurationImageURLSafety`.

## Acceptance evidence

| Criterion | Final evidence | Result |
| --- | --- | --- |
| Unicode, whitespace, lengths, controls | NFC/whitespace, exact boundaries, malformed UTF-8, ASCII/Unicode control and bidi cases. | PASS |
| Providers, identifiers, image URLs, serving units | Fixed provider/unit vocabularies and adversarial HTTPS host/port/private-target tests. | PASS |
| Numeric/macro/micronutrient validation | Required non-null macro members; finite, nonnegative, domain/storage maxima on page, every macro, serving quantity, and every micronutrient. | PASS |
| Before-dispatch rejection | Structured `400 validation_failed`; null, duplicate, malformed, unknown, and out-of-range requests leave provider/repository counters unchanged. | PASS |
| Typed normalization handoff | Handlers retrieve only normalized `curation.*Request` values from Fiber locals. | PASS |
| Metadata-only logs | Fixed metadata keys, allowlisted categorical field/outcome values, no raw accepted/rejected values or error text. | PASS |

## Verification results

| Command | Result |
| --- | --- |
| `go test -count=1 -race -coverprofile=task-242-cover.out ./internal/security ./internal/curation ./internal/httpapi` | PASS; security 99.7%, curation 100.0%, HTTP 87.8%, focused aggregate 89.9%. |
| `go vet ./internal/security ./internal/curation ./internal/httpapi` | PASS. |
| `go test -count=1 ./...` | PASS for every backend package, including repository/search integration packages. |
| `go test -count=1 -race ./...` | PASS for every backend package. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: zero vulnerabilities in called code; 18 vulnerable required-module versions are not called. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS; existing OAuth callback 302-only warning remains explicitly ignored. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; no status edit. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | PASS, exit 0: all aggregate documentation, security, migration, local stack, UAT, backend test/race/coverage, frontend verification/build/unit/coverage, focused Playwright (72 + 30), full Playwright (237 passed/3 expected skipped), and 459 frontend unit tests completed. |

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `backend/go.mod` | `f5862e14e1ed5853faeac3570bca1dee331f475f5dd901b8eeeba7634f79e997` |
| `backend/internal/security/normalizer.go` | `f87732321090d144229227b4573cf5ff1155d80f95c4e68da44a513c55802607` |
| `backend/internal/security/curation_normalizer_test.go` | `0d47f4607931e798ab9f86ba3a11a07fbeb76b56ae6f375d834b6bcdeb9d6303` |
| `backend/internal/curation/validation.go` | `8b66ed5241864693c7634b0d4dd41aa30535625daa57976455871ba19a3274f6` |
| `backend/internal/curation/validation_test.go` | `62bd666b294d0609a5f007cd016fc8c5c6c20e418e4ae1ef10ad4c23b12ae5fd` |
| `backend/internal/httpapi/curation_validation.go` | `14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1` |
| `backend/internal/httpapi/curation_validation_test.go` | `f541715892e9d4ecbfabce62e602e029d3a8977e552c9d38bdd35a7780e32292` |
| `docs/implementation/02_TASK_LIST.md` (unchanged control) | `7da6214eb5b778c8d5b63b5b2dd27808d2f38cebb81c872190afe46b3b7f1932` |

## Handoff

- All exact important review findings are repaired; no Task 242 security or acceptance blocker remains.
- Future curation handlers must use the `Normalized*Request` accessors and must not reparse raw request data.
- The image URL validator remains intentionally lexical because Task 242 performs no fetch. Future dereferencing must resolve/revalidate every DNS result and redirect target.
- Active micronutrient vocabulary membership remains later curation/repository work; Task 242 enforces shape and numeric integrity only.
- Independent re-review should recompute the hashes above and inspect the repaired symbols and adversarial tests. Task-list status must remain untouched by the repair agent.
