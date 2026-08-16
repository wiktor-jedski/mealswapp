# Review Evidence: Task 242 — Curation Input Normalization

```yaml
task_id: 242
component: "Phase 08 Curation Input Normalization"
static_aspect: "DESIGN-013: InputNormalizer"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T07:40:31Z"
review_agent: "Codex independent repair re-review"
evidence_file: "docs/implementation/reviews/task-242-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69 plus preparation repair manifest"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "reference/go.md and reference/security-review-guide.md"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 08: extend typed normalization and request validation for admin-authored item and classification names, external queries, provider identifiers, image URLs, serving-unit aliases, and provider text used by curation flows without logging rejected raw values.

**Depends On:** 115 (`PASSED`)

**Testing Coverage Exceptions:** None.

**Verification Criteria:** Table-driven unit and HTTP validation tests cover whitespace and Unicode normalization, length and character boundaries, supported providers, safe image URLs, canonical serving-unit aliases, malformed numeric/macro/micronutrient fields, rejected control characters, structured 400 responses before provider or repository dispatch, and metadata-only logs for changed or rejected values.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation repair report claims completion of all five prior important findings.
- [x] A task-specific baseline, prior review, and current repair manifest are available and trustworthy.
- [x] `code-review-skill` was invoked exactly once and its relevant Go and security guides were read.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list-status changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "None. The prior five important findings were reverified as repaired."
```

## 3. Review Baseline and Change Surface

Baseline/reference method: The preparation report fixes `81ca40ce00cb667ea29243ed2d34068e11229a69`, identifies the original seven Task 242 implementation/test paths, records the prior rejected review, and records the repaired before/after hashes. The current task row is independently `PREPARED`, dependency 115 is `PASSED`, and the current seven implementation/test hashes match the repaired preparation manifest. The task-list status and unrelated Tasks 238–241 worktree changes were excluded.

Commands used to reconstruct the diff:

```bash
git rev-parse --verify HEAD
rg -n '^\| 242 \||^\| 115 \|' docs/implementation/02_TASK_LIST.md
git status --short --untracked-files=all
git diff -- backend/go.mod backend/internal/security/normalizer.go
git diff --no-index -- /dev/null backend/internal/security/curation_normalizer_test.go
git diff --no-index -- /dev/null backend/internal/curation/validation.go
git diff --no-index -- /dev/null backend/internal/curation/validation_test.go
git diff --no-index -- /dev/null backend/internal/httpapi/curation_validation.go
git diff --no-index -- /dev/null backend/internal/httpapi/curation_validation_test.go
rg -n '^(func|type|var|const) ' <seven Task 242 Go files>
rg -n '<repaired symbol names>' backend --glob '*.go'
sha256sum <seven Task 242 implementation/test files>
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains unrelated tracked and untracked changes from Tasks 238–241, including API, application, repository, deletion-worker, design, frontend, migration, and task-list files. They were preserved. Task 242 owns only `backend/go.mod`, the curation additions, the curation HTTP additions, the curation security tests, and the existing curation normalizer implementation surface. The preparation report and review evidence are evidence files, not production symbols. No production code or task-list status was edited during this re-review.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/go.mod` | Original Task 242 dependency diff | HIGH | Direct `golang.org/x/text` dependency configuration |
| `backend/internal/security/normalizer.go` | Original Task 242 diff; unchanged during repair | HIGH | Curation field dispatch, text/provider/ID/URL/unit helpers |
| `backend/internal/security/curation_normalizer_test.go` | Repair expansion of original Task 242 tests | HIGH | Three table-driven security tests |
| `backend/internal/curation/validation.go` | Original Task 242 addition plus repair bounds/logging changes | HIGH | Typed contracts, normalizer methods, numeric and metadata helpers |
| `backend/internal/curation/validation_test.go` | Repair expansion of original Task 242 tests | HIGH | Seven curation tests |
| `backend/internal/httpapi/curation_validation.go` | Original Task 242 addition plus repaired raw middleware/handoff | HIGH | Validator, accessors, strict decoder, recursive duplicate scanner |
| `backend/internal/httpapi/curation_validation_test.go` | Repair expansion of original Task 242 HTTP test | HIGH | HTTP dispatch, handoff, malformed-input, duplicate, and log tests |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Whitespace and Unicode normalization | Security and typed-request tables plus handler-visible values | PASS | NFC composition and whitespace collapse are verified for names, queries, provider text, providers, units, and the HTTP handoff. |
| 2 | Length and character boundaries | Exact maximum and over-maximum tests, UTF-8 and character-category inspection | PASS | Rune bounds, ASCII provider identifiers, name character allowlists, URL byte bounds, malformed UTF-8, symbols, and controls are covered. |
| 3 | Supported providers | Provider allowlist tests and external-versus-persisted `all` behavior | PASS | `usda`, `openfoodfacts`, and external-only `all` normalize to fixed tokens; unsupported and persisted `all` are rejected. |
| 4 | Safe image URLs | HTTPS, host, credential, fragment, private-target, escape, port, and length tests | PASS | Lexical public-looking HTTPS validation rejects local/private literals, unsafe schemes, credentials, fragments, controls, invalid ports, and malformed hosts. No URL is fetched in this task. |
| 5 | Canonical serving-unit aliases | Alias table tests and repository vocabulary inspection | PASS | Accepted aliases produce only `g`, `ml`, `oz`, `fl_oz`, or `serving`; unsupported `cup` is rejected. |
| 6 | Malformed numeric, macro, and micronutrient fields are rejected | Typed bounds tests and HTTP null, missing, type, finite-extreme, key, and dispatch tests | PASS | Missing/null/non-object macros and null required members reject before dispatch; all macro components, serving quantity, and micronutrients enforce finite nonnegative domain/storage ceilings. |
| 7 | Rejected control characters | Text, URL, malformed UTF-8, bidi, and HTTP control-input tests | PASS | Visible text and decoded URL controls are rejected; canonical provider/unit outputs cannot carry input controls. |
| 8 | Structured 400 before provider/repository dispatch | Fiber route counters for all repaired malformed cases | PASS | Null/missing macros, duplicate JSON at top level and nested macro objects, duplicate query keys, malformed bodies, numeric extremes, and common invalid values return the generic validation envelope with zero rejected dispatches. |
| 9 | Metadata-only logs for changed/rejected values | Memory-sink sentinel tests and runtime allowlist inspection | PASS | Logs contain only fixed service/message, allowlisted field/outcome categories, booleans, counts, and timestamps; raw values, errors, arbitrary fields, and arbitrary outcomes are dropped. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `golang.org/x/text` direct dependency | configuration | `backend/go.mod:8-35` | modified | NFC normalizer build | backend tests and module resolution |
| 2 | Curation `InputField` constants | constants | `backend/internal/security/normalizer.go:51-68` | added | `NormalizeInput`; curation normalizer | security curation tables |
| 3 | Curation length constants | constants | `backend/internal/security/normalizer.go:79-84` | added | curation text, ID, URL, and provider helpers | boundary tables |
| 4 | `NormalizeInput` | function | `backend/internal/security/normalizer.go:106-155` | modified | all existing normalizer callers and curation normalizer | security and curation tests |
| 5 | `normalizeCurationName` | function | `backend/internal/security/normalizer.go:441-453` | added | `NormalizeInput` | text boundary table |
| 6 | `normalizeVisibleText` | function | `backend/internal/security/normalizer.go:457-477` | added | names, queries, provider text | text and HTTP tests |
| 7 | `normalizeCurationProvider` | function | `backend/internal/security/normalizer.go:482-502` | added | `NormalizeInput` and curation methods | provider table |
| 8 | `normalizeProviderIdentifier` | function | `backend/internal/security/normalizer.go:506-521` | added | `NormalizeInput` and item normalization | identifier table |
| 9 | `normalizeImageURL` | function | `backend/internal/security/normalizer.go:525-570` | added | `NormalizeInput` and item normalization | URL safety table |
| 10 | `normalizeServingUnit` | function | `backend/internal/security/normalizer.go:573-596` | added | `NormalizeInput` and item normalization | unit table |
| 11 | `containsControl` | function | `backend/internal/security/normalizer.go:598-606` | added | image URL decoder checks | URL safety table |
| 12 | `isDisallowedControl` | function | `backend/internal/security/normalizer.go:609-611` | added | visible text and URL checks | control tables |
| 13 | `TestCurationTextNormalizationBoundaries` | test | `backend/internal/security/curation_normalizer_test.go:10-49` | modified | curation security helpers | table assertions |
| 14 | `TestCurationProviderIdentifierAndUnitNormalization` | test | `backend/internal/security/curation_normalizer_test.go:51-86` | modified | provider, ID, unit helpers | table assertions |
| 15 | `TestCurationImageURLSafety` | test | `backend/internal/security/curation_normalizer_test.go:88-122` | modified | image URL helper | URL table assertions |
| 16 | `ExternalSearchRequest` | behavioral type | `backend/internal/curation/validation.go:21-25` | added | provider search boundary | typed and HTTP tests |
| 17 | `ItemRequest` | behavioral type | `backend/internal/curation/validation.go:29-40` | added | item curator/import boundary | typed and HTTP tests |
| 18 | `ClassificationRequest` | behavioral type | `backend/internal/curation/validation.go:44-46` | added | classification boundary | typed and HTTP tests |
| 19 | `RejectionField` | type | `backend/internal/curation/validation.go:50` | added | `RecordRejection` | metadata tests |
| 20 | Rejection field constants | constants | `backend/internal/curation/validation.go:54-58` | added | HTTP pre-decode paths | metadata and HTTP tests |
| 21 | `InputNormalizer` | behavioral type | `backend/internal/curation/validation.go:63-66` | added | curation flows | curation tests |
| 22 | `NewInputNormalizer` | function | `backend/internal/curation/validation.go:78-80` | added | HTTP constructor and typed callers | curation and HTTP tests |
| 23 | `RecordRejection` | method | `backend/internal/curation/validation.go:84-86` | modified | HTTP decoding failures | metadata and HTTP tests |
| 24 | `NormalizeExternalSearch` | method | `backend/internal/curation/validation.go:90-107` | modified | future provider proxy and HTTP middleware | typed and HTTP tests |
| 25 | `NormalizeItem` | method | `backend/internal/curation/validation.go:109-165` | modified | future importer/item curator and HTTP middleware | typed bounds and HTTP tests |
| 26 | `NormalizeClassification` | method | `backend/internal/curation/validation.go:167-175` | added | future tag manager and HTTP middleware | typed and HTTP tests |
| 27 | `normalize` | method | `backend/internal/curation/validation.go:177-189` | added | all typed normalizer methods | curation tests |
| 28 | `log` | method | `backend/internal/curation/validation.go:191-201` | modified | normalization/rejection paths | metadata tests |
| 29 | `allowedLogField` | function | `backend/internal/curation/validation.go:203-218` | added | `log` runtime allowlist | invalid metadata test |
| 30 | `validMacrosWithinBounds` | function | `backend/internal/curation/validation.go:220-224` | added | `NormalizeItem` | numeric bounds test |
| 31 | `validateMicronutrients` | function | `backend/internal/curation/validation.go:226-247` | modified | `NormalizeItem` | micro boundary and numeric tests |
| 32 | `TestInputNormalizerNormalizesTypedCurationRequests` | test | `backend/internal/curation/validation_test.go:16-50` | modified | typed request methods | canonical output/log assertions |
| 33 | `TestInputNormalizerRejectsMalformedCurationFieldsWithoutRawLogs` | test | `backend/internal/curation/validation_test.go:52-105` | modified | item validation paths | malformed/log assertions |
| 34 | `TestInputNormalizerOptionalAndMetadataBranches` | test | `backend/internal/curation/validation_test.go:107-124` | modified | optional map and rejection metadata | branch assertions |
| 35 | `TestRecordRejectionDropsUnknownMetadata` | test | `backend/internal/curation/validation_test.go:126-136` | added | log field/outcome boundary | raw-sentinel assertions |
| 36 | `TestInputNormalizerRejectsNumericValuesAboveDocumentedBounds` | test | `backend/internal/curation/validation_test.go:138-178` | added | macro, serving, micro bounds | max and downstream scaling assertions |
| 37 | `TestValidateMicronutrientsBoundaries` | test | `backend/internal/curation/validation_test.go:180-200` | modified | micronutrient validator | key/value/count assertions |
| 38 | `TestExternalSearchRejectsBeforeProviderUse` | test | `backend/internal/curation/validation_test.go:202-214` | added | external search normalizer | rejection assertions |
| 39 | normalized Fiber local constants | constants | `backend/internal/httpapi/curation_validation.go:17-21` | added | curation accessors and handlers | HTTP handoff test |
| 40 | `CurationRequestValidator` | behavioral type | `backend/internal/httpapi/curation_validation.go:25-27` | added | Fiber route validation | HTTP integration test |
| 41 | `NewCurationRequestValidator` | function | `backend/internal/httpapi/curation_validation.go:31-33` | added | future route composition | HTTP integration test |
| 42 | `ValidateExternalSearchQuery` | method | `backend/internal/httpapi/curation_validation.go:37-64` | modified | `RouteDefinition.Validate` | query dispatch and duplicate tests |
| 43 | `ValidateItemBody` | method | `backend/internal/httpapi/curation_validation.go:68-80` | modified | `RouteDefinition.Validate` | body dispatch and macro tests |
| 44 | `ValidateClassificationBody` | method | `backend/internal/httpapi/curation_validation.go:84-96` | modified | `RouteDefinition.Validate` | classification dispatch tests |
| 45 | `NormalizedExternalSearchRequest` | function | `backend/internal/httpapi/curation_validation.go:100-103` | added | provider handlers | normalized handoff test |
| 46 | `NormalizedCurationItemRequest` | function | `backend/internal/httpapi/curation_validation.go:107-110` | added | item repository handlers | normalized handoff test |
| 47 | `NormalizedCurationClassificationRequest` | function | `backend/internal/httpapi/curation_validation.go:114-117` | added | classification handlers | normalized handoff test |
| 48 | `inputNormalizer` | method | `backend/internal/httpapi/curation_validation.go:121-127` | modified | all validator methods | indirect HTTP coverage |
| 49 | `curationValidationError` | function | `backend/internal/httpapi/curation_validation.go:130-133` | added | validator error paths | structured 400 assertions |
| 50 | `decodeStrictBody` | function | `backend/internal/httpapi/curation_validation.go:136-155` | modified | item/classification validators | malformed body tests |
| 51 | `validateRequiredMacros` | function | `backend/internal/httpapi/curation_validation.go:157-180` | added | item validator | null/missing/non-object tests |
| 52 | `rejectDuplicateJSONKeys` | function | `backend/internal/httpapi/curation_validation.go:182-192` | added | strict body decoder | duplicate JSON tests |
| 53 | `scanJSONValue` | function | `backend/internal/httpapi/curation_validation.go:196-231` | added | recursive duplicate scanner | nested macro and array traversal |
| 54 | `TestCurationHTTPValidationStopsBeforeProviderOrRepositoryDispatch` | test | `backend/internal/httpapi/curation_validation_test.go:18-146` | modified | curation middleware and handlers | full HTTP regression matrix |
| 55 | `assertCurationStatus` | test helper | `backend/internal/httpapi/curation_validation_test.go:148-155` | modified | HTTP integration test | status assertions |
| 56 | `curationRequest` | test helper | `backend/internal/httpapi/curation_validation_test.go:157-166` | modified | HTTP integration test | request setup |
| 57 | `assertStructuredCuration400` | test helper | `backend/internal/httpapi/curation_validation_test.go:168-176` | modified | HTTP integration test | safe envelope assertions |

```yaml
inventory_source_count: 57
audited_symbol_count: 57
inventory_complete: true
generated_groupings:
  - "No generated executable units; constants and the direct dependency configuration are grouped only where they form one declaration block."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `golang.org/x/text` direct dependency | Supplies NFC normalization used by curation text. | Module resolution succeeds and no unrelated import was added. | N/A — immutable dependency metadata. | Trusted pinned module is used only for Unicode normalization. | No runtime I/O owned by this unit. | Necessary direct dependency. | Full build, tests, and vulnerability scan pass. | PASS |
| Curation `InputField` constants | Give each curation input one explicit normalizer rule. | All curation switch cases are present; unknown fields fail closed. | N/A — immutable constants. | Fixed field categories do not contain user values. | O(1) dispatch. | Clear typed vocabulary. | Security table exercises each curation category. | PASS |
| Curation length constants | Bound normalized names, queries, IDs, URLs, and provider text. | Exact and over-limit cases are tested; rune versus byte policy is intentional. | N/A — immutable constants. | Prevent oversized accepted fields. | Constant-time lookup. | Minimal shared policy. | Boundary tables and HTTP cases pass. | PASS |
| `NormalizeInput` | Dispatches each field to its typed curation policy. | Every added field has a case; default returns a generic unsupported-field error. | Stateless and synchronous. | No logging or external side effects. | O(input) plus bounded helper work. | Preserves existing API shape. | Security and typed tests cover all added cases. | PASS |
| `normalizeCurationName` | Produces required NFC/whitespace-normalized names with allowlisted Unicode categories. | Empty, overlong, symbol, emoji, mark, punctuation, and control paths inspected. | Stateless; N/A — no waits or resources. | Rejects controls and unsupported symbols before persistence. | Full input normalization precedes rune bound; HTTP body is framework-bounded. | Small idiomatic helper. | Text table and HTTP handoff pass. | PASS |
| `normalizeVisibleText` | Validates UTF-8, rejects controls and format characters, collapses whitespace, and applies rune bounds. | Invalid UTF-8, empty required, optional empty, exact bound, and over-bound paths are handled. | Stateless. | Prevents bidi/control values from logs or trusted fields. | O(input); no external I/O. | Uses standard NFC and `strings.Fields`. | Security tests include malformed UTF-8 and controls. | PASS |
| `normalizeCurationProvider` | Maps only supported provider tokens to canonical identifiers. | Case, trim, alias, unsupported, and persistable `all` cases are handled. | Stateless. | Provider selection is an allowlist. | Tiny bounded allocation. | Explicit switch is easy to audit. | Provider table and HTTP search handoff pass. | PASS |
| `normalizeProviderIdentifier` | Accepts bounded ASCII opaque provider IDs with a narrow token alphabet. | Empty, Unicode, spaces, overlong, and allowed punctuation paths are handled. | Stateless. | Blocks control and injection-shaped characters at provider boundary. | O(input), bounded after trim. | Explicit character loop. | Identifier table and item tests pass. | PASS |
| `normalizeImageURL` | Accepts optional absolute HTTPS URLs with public-looking hosts, no credentials/fragments, and no decoded controls. | Invalid scheme, host, credentials, local/private literal, port, escape, fragment, and length paths are handled. | No DNS, fetch, goroutine, or resource lifecycle; future fetch must revalidate DNS and redirects. | Lexical SSRF defenses are fail-closed for this no-fetch task. | Bounded URL parse and unescape. | Conservative URL policy is documented. | URL table includes public/invalid ports, Unicode host, private literals, and encoded NUL. | PASS |
| `normalizeServingUnit` | Maps accepted aliases to canonical repository unit tokens. | Alias, unsupported, empty, and whitespace paths are deterministic. | Stateless. | Output is a closed unit vocabulary. | Small allocation. | Clear alias switch. | Unit table and HTTP canonical handoff pass. | PASS |
| `containsControl` | Detects controls in decoded URL path/query data. | Iterates all decoded runes and includes format controls through the shared predicate. | Stateless. | Protects protocol and log boundaries. | O(input). | Minimal helper. | URL tests cover decoded NUL and control policy. | PASS |
| `isDisallowedControl` | Rejects Unicode control and format-control categories. | ASCII controls and bidi/format controls are rejected. | Stateless. | Blocks log/protocol control injection. | O(1) per rune. | Idiomatic Unicode predicates. | Text and URL tests pass. | PASS |
| `TestCurationTextNormalizationBoundaries` | Proves curation text normalization and boundaries. | Table covers NFC, whitespace, lengths, symbols, controls, bidi, and malformed UTF-8. | Test-local state only. | Sentinel values are not logged. | Small deterministic table. | Idiomatic table-driven test. | Direct coverage passes. | PASS |
| `TestCurationProviderIdentifierAndUnitNormalization` | Proves provider, ID, and unit allowlists. | Supported aliases, unsupported providers, IDs, and canonical units are asserted. | Test-local only. | Unsafe IDs and unsupported provider choices reject. | Small deterministic table. | Clear table structure. | Direct coverage passes. | PASS |
| `TestCurationImageURLSafety` | Proves lexical image URL safety. | HTTPS, credentials, private/local literals, ports, Unicode host, fragments, escapes, and length are covered. | No network or DNS. | SSRF-focused negative cases are deterministic. | Small table. | No flaky external dependency. | Direct coverage passes. | PASS |
| `ExternalSearchRequest` | Carries normalized provider query, provider, and page. | HTTP creates it only after strict query collection and typed normalization. | Plain value type; no owned resources. | Caller boundary is validated before provider use. | No I/O. | Appropriate typed contract. | Typed and HTTP tests inspect canonical values. | PASS |
| `ItemRequest` | Carries typed curation item fields and validated numeric/maps. | Required macro presence is enforced at raw HTTP boundary; typed normalizer enforces domain and bounds. | Plain value type. | Only normalized values are handed to handlers. | Numeric and map count/value bounds are enforced. | Appropriate future-flow contract. | Typed and HTTP malformed tests pass. | PASS |
| `ClassificationRequest` | Carries a normalized required classification name. | Empty, controls, unknown fields, duplicates, and malformed bodies reject. | Plain value type. | Handler receives only normalized name. | Trivial value copy. | Minimal typed contract. | Typed and HTTP tests pass. | PASS |
| `RejectionField` | Represents pre-decode metadata categories. | Arbitrary casts are runtime-rejected by the logging allowlist. | Immutable string value. | Unknown/raw values cannot enter logs. | O(1) allowlist switch. | Public type remains safe at runtime. | Raw sentinel test passes. | PASS |
| Rejection field constants | Enumerate fixed HTTP rejection categories. | All pre-decode callers use constants; pagination is allowlisted. | Immutable. | Non-PII categories only. | O(1). | Clear vocabulary. | Metadata and HTTP tests pass. | PASS |
| `InputNormalizer` | Owns typed curation validation and metadata sink. | Nil sink works; methods return safe generic errors after metadata logging. | Shared sink pointer; no goroutines, locks, files, or network. | Sink events contain no raw input. | Synchronous bounded log event. | Cohesive small type. | Full curation package coverage. | PASS |
| `NewInputNormalizer` | Constructs a normalizer with optional sink. | Nil and nonnil sinks are supported. | No resource acquisition. | Caller-provided sink receives only fixed events. | O(1). | Idiomatic constructor. | Indirect and direct tests pass. | PASS |
| `RecordRejection` | Records only allowlisted pre-normalization categories. | Unknown fields are silently dropped; known fields emit rejected metadata. | Synchronous sink call; sink error is intentionally non-fatal. | Runtime allowlist prevents raw metadata injection. | O(1) event creation. | Safe public boundary. | Invalid field/outcome sentinel test passes. | PASS |
| `NormalizeExternalSearch` | Returns canonical query/provider/page before outbound use. | Invalid query/provider/page returns before result; page parsing is safe after the page normalizer. | No provider I/O; caller context reaches log sink. | Provider and query are allowlisted/controlled. | Bounded text and page work. | Typed return contract is preserved. | Typed rejection and HTTP handoff tests pass. | PASS |
| `NormalizeItem` | Returns one canonical item only after all field, macro, serving, provenance, and micro checks. | Null/missing macro presence is enforced before this method at HTTP; finite, negative, sum, pair, and upper-bound errors return safely. | No external resources; synchronous bounded map loop. | Image, provider ID, text, macro, and micro trust boundaries are checked. | Micro map capped at 200 and values bounded. | Reuses repository macro invariants without exposing errors. | Typed malformed and max/downstream tests pass. | PASS |
| `NormalizeClassification` | Returns canonical classification name before repository use. | Required/control/Unicode errors return safely. | Stateless except optional metadata sink. | No raw error or value logging. | O(name). | Minimal. | Typed and HTTP tests pass. | PASS |
| `normalize` | Centralizes field normalization and metadata logging. | Rejected fields log category only; changed fields log boolean/count only. | Uses caller context; no owned resources. | Result/error values never enter logs. | One helper call plus bounded event. | Good reuse. | Curation tests inspect event shape. | PASS |
| `log` | Emits fixed categorical metadata and rejects unknown field/outcome strings. | Nil receiver/sink, unknown fields, and unknown outcomes are no-ops. | Synchronous sink; sink errors are intentionally ignored for validation. | No raw values or error strings are accepted. | One fixed-size map allocation. | Small defensive boundary. | Unknown metadata test passes. | PASS |
| `allowedLogField` | Closes event field vocabulary over non-PII categories. | Every production field and rejection constant is included; arbitrary values reject. | Immutable switch; N/A — no state. | Prevents field-name injection and PII metadata. | O(1). | Explicitly auditable. | Sentinel test and caller audit pass. | PASS |
| `validMacrosWithinBounds` | Enforces persisted `numeric(12,4)` upper bound for every macro component. | Max and next-representable values are tested; lower/finite checks remain in repository validator. | Stateless. | Prevents storage overflow and later arithmetic extremes. | O(1). | Small focused helper. | All three components and scaling pass. | PASS |
| `validateMicronutrients` | Enforces bounded map size, canonical-shaped keys, finite nonnegative values, and numeric maximum. | Empty, long, invalid key, count, NaN, Inf, negative, and over-maximum paths are handled. | No external resources; bounded loop. | Values and keys cannot inject controls or unbounded numeric data; active vocabulary membership remains later scoped work. | At most 200 entries and bounded key length. | Clear validation loop. | Boundary and max tests pass. | PASS |
| `TestInputNormalizerNormalizesTypedCurationRequests` | Proves canonical typed outputs and metadata-only logs. | Search, item, classification, aliases, NFC, and provider text are asserted. | Test-local memory sink. | Raw sentinels are checked absent. | Small deterministic test. | Good integration-style unit test. | Direct coverage passes. | PASS |
| `TestInputNormalizerRejectsMalformedCurationFieldsWithoutRawLogs` | Proves representative typed malformed inputs reject. | URL, unit, provider, ID, state, macro, serving, micro, and control cases are covered. | Test-local sink; no leaks or resources. | Raw sentinels are asserted absent. | Small table. | Clear negative test. | Direct coverage passes. | PASS |
| `TestInputNormalizerOptionalAndMetadataBranches` | Proves optional micro map and valid rejection metadata behavior. | Nil map becomes an empty map; invalid classification rejects. | Memory sink only. | Known category is emitted safely. | Small deterministic test. | Focused branch test. | Direct coverage passes. | PASS |
| `TestRecordRejectionDropsUnknownMetadata` | Proves arbitrary field and outcome text cannot become events. | Unknown field and outcome paths are both exercised. | Test-local sink. | Raw sentinel is absent. | O(1). | Direct security regression. | Passes. | PASS |
| `TestInputNormalizerRejectsNumericValuesAboveDocumentedBounds` | Proves macro, serving, and micro maxima. | Every macro component, next representable bound, exact maxima, and downstream scaling are asserted. | Test-local values only. | Storage-compatible numeric domain is enforced. | Small deterministic map. | Strong table plus boundary assertions. | Passes. | PASS |
| `TestValidateMicronutrientsBoundaries` | Proves map/key/value rejection classes. | Count, empty/long/marked keys, NaN, and Inf cases are covered. | Test-local map. | Key syntax is bounded and non-control. | Count capped at 201 in test. | Table-driven. | Passes. | PASS |
| `TestExternalSearchRejectsBeforeProviderUse` | Proves core search rejection before a provider caller could use the request. | Empty/control/unsupported/zero-page cases reject. | No provider or network. | Fixed provider allowlist. | Small table. | Clear pre-dispatch intent. | Passes. | PASS |
| normalized Fiber local constants | Name private per-request storage for approved typed values. | One local per request kind; accessors type-check values. | Fiber request-local state, no shared mutable state. | Handlers cannot receive raw values through these accessors. | O(1) local lookup. | Minimal handoff mechanism. | HTTP handler assertions pass. | PASS |
| `CurationRequestValidator` | Adapts typed curation rules to Fiber middleware. | Holds the normalizer and uses request context. | Per-validator sink pointer is safe for concurrent requests when sink is thread-safe by contract. | Raw body/query is validated before `ctx.Next`. | No I/O beyond optional synchronous log sink. | Appropriate route middleware shape. | HTTP integration test mounts it directly. | PASS |
| `NewCurationRequestValidator` | Builds reusable metadata-safe HTTP validation. | Nil sink is accepted. | No resource acquisition. | Sink receives fixed metadata only. | O(1). | Idiomatic constructor. | HTTP test passes. | PASS |
| `ValidateExternalSearchQuery` | Rejects ambiguous query args and stores one canonical typed request before handler dispatch. | Duplicate decoded names, wrong count, bad page, invalid fields, and extra params reject. | Uses `ctx.UserContext`; no goroutines/resources. | Raw query values do not reach handler or logs. | One bounded map and typed normalization. | Consistent with route middleware contract. | Duplicate and canonical handoff tests pass. | PASS |
| `ValidateItemBody` | Strictly decodes, requires macro object/members, normalizes, and stores item before dispatch. | Malformed UTF-8, null/missing/non-object macros, null required members, duplicates, unknowns, type errors, trailing data, and domain errors reject. | Uses request context and no external resources. | Raw JSON is scanned before typed conversion and handler dispatch. | Bounded by Fiber body limit and capped nested map semantics. | Generic error mapping hides parser detail. | Full HTTP malformed matrix passes. | PASS |
| `ValidateClassificationBody` | Strictly decodes, normalizes, and stores classification before dispatch. | Non-object, malformed, duplicate, unknown, trailing, empty, and control inputs reject. | Request context; no resources. | Raw data is not handed to handler. | Bounded raw decode. | Consistent middleware API. | HTTP matrix passes. | PASS |
| `NormalizedExternalSearchRequest` | Returns only the typed search value installed by validation. | Missing or wrong local type returns false rather than a zero-approved value. | Request-local lookup; no shared state. | Handler contract is explicit and typed. | O(1). | Small accessor. | Handler canonical-value assertions pass. | PASS |
| `NormalizedCurationItemRequest` | Returns only the typed item value installed by validation. | Missing or wrong local returns false. | Request-local lookup. | Prevents raw reparse requirement. | O(1). | Small accessor. | Handler canonical-value assertions pass. | PASS |
| `NormalizedCurationClassificationRequest` | Returns only the typed classification value installed by validation. | Missing or wrong local returns false. | Request-local lookup. | Prevents raw value handoff. | O(1). | Small accessor. | Handler canonical-value assertions pass. | PASS |
| `inputNormalizer` | Supplies a safe no-log normalizer for nil validator composition. | Nil receiver and nil normalizer are handled. | Constructs no external resource. | Nil composition cannot create raw logs. | O(1). | Defensive helper. | Covered indirectly; remaining branch is harmless optional coverage. | PASS |
| `curationValidationError` | Maps all typed/parser rejection paths to one generic structured 400. | Fixed status/category/code/message; no parser details exposed. | No state/resources. | Prevents raw input and internal diagnostics in responses. | Constant-sized value. | Simple error factory. | Envelope assertions pass. | PASS |
| `decodeStrictBody` | Accepts one UTF-8 JSON object with no duplicates, unknowns, type mismatch, or trailing data. | Empty, non-object, malformed, duplicate, unknown, malformed-type, and trailing paths reject; valid object reaches typed decode. | Local decoder only; no external I/O. | Raw body is preserved until duplicate and strict checks complete. | Scanner and decoder are bounded by framework body limit; no unbounded map acceptance. | Correct abstraction for strict body boundary. | HTTP malformed tests exercise relevant paths. | PASS |
| `validateRequiredMacros` | Requires `macrosPer100` object and non-null protein, carbohydrates, and fat members. | Missing, null, non-object, malformed, and valid required-member paths are handled. | Local raw-message maps only. | Prevents zero-value struct bypass at trust boundary. | Fixed required-key loop. | Explicit presence check. | Null/missing HTTP regressions pass. | PASS |
| `rejectDuplicateJSONKeys` | Rejects duplicate object names throughout a JSON document. | Top-level, nested object, array traversal, malformed and trailing paths are handled. | Local decoder; no goroutines/resources. | Prevents last-value-wins ambiguity before typed conversion. | Linear scan with bounded body and 200-key domain map after decode. | Small scanner wrapper. | Top-level and nested duplicate tests pass. | PASS |
| `scanJSONValue` | Recursively consumes one JSON value while tracking object keys. | Objects, arrays, primitives, malformed delimiters, duplicate names, and premature EOF are handled by decoder errors. | Stack depth is bounded by the standard JSON decoder and request body limit; no owned resources. | Escaped duplicate names are compared after JSON decoding. | Linear token scan; per-object seen set. | Idiomatic decoder token walk. | Nested macro and array-containing body paths are exercised. | PASS |
| `TestCurationHTTPValidationStopsBeforeProviderOrRepositoryDispatch` | Proves valid typed handoff and rejected-before-dispatch behavior. | Covers query duplicates, body duplicates, nested macro duplicates, null/missing/type/extreme/malformed data, controls, URLs, unknowns, trailing bytes, and structured envelopes. | Fiber counters and memory sink are test-local; response bodies are closed. | Raw sentinels are checked absent from logs and handlers inspect normalized values. | Deterministic in-process HTTP test. | Strong regression matrix. | Passes under normal and race tests. | PASS |
| `assertCurationStatus` | Asserts expected HTTP status and closes response. | Success/error statuses are checked. | Response body is closed on all helper paths. | N/A — test helper. | Minimal. | Idiomatic helper. | Used by valid dispatch tests. | PASS |
| `curationRequest` | Builds bounded test JSON requests and returns responses. | Request errors fail test; content type is set. | Response ownership is passed to caller and callers close it. | Test-only. | Minimal allocation. | Idiomatic helper. | Used throughout HTTP matrix. | PASS |
| `assertStructuredCuration400` | Verifies generic validation envelope. | Status, error presence, category, and code are checked; body closes. | No leaks in helper. | Confirms no parser detail reaches client envelope. | Minimal. | Focused helper. | Used for every rejected boundary case. | PASS |

Mandatory audit conclusion: Every inventory entry was inspected with its callers, callees, design sources, error paths, malformed inputs, cancellation, concurrency, security boundaries, allocation/I/O behavior, API necessity, and adversarial tests. Task 242 creates no goroutines, files, subprocesses, SQL, transactions, locks, or outbound network calls. HTTP validation now propagates the request context. The image URL policy remains lexical by scope; future dereferencing must DNS-resolve and revalidate every result and redirect.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| [optional] | `scripts/verify-local-stack.py:106-243` | aggregate local-stack verifier | The repository aggregate command could not reach its later stages because the pre-existing local `mealswapp_test` PostgreSQL state is incomplete and migration reruns fail at unrelated migration dependencies. | `python3 scripts/check.py` exited 1 at local-stack migration verification; a direct rerun failed at `000010_admin_import_audit` because `curated_imports` does not exist. No Task 242 source or migration is involved, and all task-focused/full backend gates passed. | No Task 242 repair. Recreate/reset the local test database through the project-owner environment workflow before relying on the aggregate gate. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

No blocking or important finding remains. The optional local-stack issue is external verification state, not a Task 242 implementation defect.

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/security ./internal/curation ./internal/httpapi` | `backend` | 0 | PASS | Focused repaired packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/security ./internal/curation ./internal/httpapi` | `backend` | 0 | PASS | Focused race suite passes. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=task-242-cover.out ./internal/security ./internal/curation ./internal/httpapi` | `backend` | 0 | PASS | Security 99.7%, curation 100.0%, HTTP 87.8%, focused aggregate 89.9%; changed curation functions and normalizer helpers are covered. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=task-242-cover.out` | `backend` | 0 | PASS | Repaired symbol coverage inspected; defensive decoder/nil-composition branches are reached through the HTTP matrix or are non-security optional gaps. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/security ./internal/curation ./internal/httpapi` | `backend` | 0 | PASS | Focused vet. |
| `gofmt -d <Task 242 Go files>; git diff --check` | repository root | 0 | PASS | No formatting or whitespace errors. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | `backend` | 0 | PASS | All backend packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | `backend` | 0 | PASS | All backend packages pass under race detection. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend` | 0 | PASS | Full backend vet. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | Zero vulnerabilities in called code; 18 required-module vulnerabilities are not called. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 sequential tasks; task 242 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability validation passes. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with existing warning | One explicitly ignored OAuth callback 302-only warning. |
| `python3 scripts/check.py` | repository root | 1 | ENVIRONMENT BLOCKED | All stages before local-stack migration passed; existing local PostgreSQL migration state stopped the aggregate before frontend stages. No Task 242 source was changed. |
| `python3 scripts/verify-local-stack.py --keep-services` | repository root | 1 | ENVIRONMENT BLOCKED | Direct rerun failed on unrelated incomplete local migration state; no services or migration files were changed. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend` | 0 | PASS | Frontend production build passed. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend` | 0 | PASS | 459 tests passed. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | `frontend` | 0 | PASS | 459 tests passed; 95.13% line coverage with existing documented deviations. |
| `python3 scripts/verify-frontend.py` | repository root | 0 | PASS | Browser verification passed; desktop/mobile screenshots written under `/tmp/mealswapp-frontend-verifier/`. |
| `sha256sum <seven Task 242 implementation/test files>` | repository root | 0 | PASS | Current hashes are recorded below and match the repaired preparation manifest. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-242-review.md` | repository root | 0 | PASS | Full review template structure and 57 inventory/audit row counts validate. |

## 9. Files Inspected and Staleness Fingerprints

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `backend/go.mod` | Direct NFC-normalization dependency | none | SHA-256 | `f5862e14e1ed5853faeac3570bca1dee331f475f5dd901b8eeeba7634f79e997` |
| `backend/internal/security/normalizer.go` | Curation field dispatch and text/provider/ID/URL/unit safety | none | SHA-256 | `f87732321090d144229227b4573cf5ff1155d80f95c4e68da44a513c55802607` |
| `backend/internal/security/curation_normalizer_test.go` | Normalization and URL adversarial tests | none | SHA-256 | `0d47f4607931e798ab9f86ba3a11a07fbeb76b56ae6f375d834b6bcdeb9d6303` |
| `backend/internal/curation/validation.go` | Typed curation contracts, bounds, allowlisted logging, and normalization | none | SHA-256 | `8b66ed5241864693c7634b0d4dd41aa30535625daa57976455871ba19a3274f6` |
| `backend/internal/curation/validation_test.go` | Typed malformed, metadata, and numeric-bound tests | none | SHA-256 | `62bd666b294d0609a5f007cd016fc8c5c6c20e418e4ae1ef10ad4c23b12ae5fd` |
| `backend/internal/httpapi/curation_validation.go` | Strict raw HTTP validation, duplicate scanner, context, and typed handoff | none | SHA-256 | `14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1` |
| `backend/internal/httpapi/curation_validation_test.go` | HTTP dispatch, handoff, malformed-input, duplicate, and log regressions | none | SHA-256 | `f541715892e9d4ecbfabce62e602e029d3a8977e552c9d38bdd35a7780e32292` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior rejected review hashes are stale for repaired curation and HTTP files; all affected symbols were re-reviewed and current hashes match the repaired preparation manifest."
```

## 10. Coverage and Exceptions

- [x] Required focused coverage command ran.
- [x] Report path and observed package/function coverage are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row: no Task 242 exception was introduced; the aggregate environment failure is unrelated local database state.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "backend/task-242-cover.out"
observed_line_coverage: "security 99.7%; curation 100.0%; HTTP 87.8%; focused aggregate 89.9%; frontend 95.13% existing documented deviations"
coverage_passed: true
```

Coverage finding: Task-owned security and curation validation behavior is covered and the repaired HTTP boundary is exercised through normal and adversarial integration paths. The HTTP package total includes unrelated pre-existing controllers; remaining defensive decoder/nil-composition branch gaps do not leave an acceptance or security path untested. No Task 242 coverage exception was added.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No source-of-truth design or requirement was contradicted within Task 242 scope.
- [x] No generated/cache/build/temporary artifact was unintentionally added to the reviewed surface.
- [x] Public API additions are necessary and used by the typed handler handoff contract.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: The five prior important findings are closed. Null and missing macro objects reject before `ctx.Next`; explicit macro, serving, and micronutrient maxima reject extreme finite values; log fields/outcomes are runtime-allowlisted; normalized typed locals reach handlers; and duplicate query/JSON names reject recursively before typed conversion. No SQL, provider network, persistence, authorization, secret, PII, URL dereference, goroutine, lock, or resource-lifecycle regression was introduced. The only failed command is the unrelated local-stack migration environment check recorded above.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

Before accepting the decision, run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-242-review.md
```

```yaml
decision: "PASSED"
reason: "All original criteria and all five prior important findings pass independent symbol, caller, adversarial, race, vet, security, and hash verification; only unrelated local PostgreSQL state blocked the aggregate script after its earlier gates."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for Task 242. Keep the task PREPARED until the orchestrator applies the requested status transition; future handlers must use the Normalized*Request accessors and future image fetching must revalidate DNS and redirects."
```

## 13. Repair Context

Not applicable for a PASSED re-review. The previous five important findings were repaired and independently verified; no further production repair instructions are required.
