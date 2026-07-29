# Review Evidence: Task 284 — DESIGN-008 AccountDeleter

task_id: 284
component: "DESIGN-008 AccountDeleter"
static_aspect: "Private isolation, portability, erasure, and managed real-stack acceptance"
input_status: PREPARED
review_decision: PASSED
reviewed_at_utc: 2026-07-28T18:50:58Z
review_agent: "Codex reviewer"
evidence_file: "docs/implementation/evidence/task-284-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677"
baseline_confidence: HIGH
latest_preparation_run: "b7dd8224067e397809b99c4e"
code_review_skill_invoked: true
relevant_language_guide: "Python, TypeScript, and Go security guidance from code-review-skill"
repair_context_required: false
pre_review_gates_passed: true
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
inventory_source_count: 33
audited_symbol_count: 33
blocking_findings: 0
important_findings: 0
optional_findings: 0

## 1. Task Source

Task 284 is the PREPARED row in `docs/implementation/02_TASK_LIST.md` for DESIGN-008 AccountDeleter and SW-REQ-043, SW-REQ-072, and SW-REQ-073. Dependencies 239, 240, 267, 279, and 280 are PREPARED or PASSED. The task-list status was not edited; its fresh SHA-256 is recorded below.

The repair package is `docs/implementation/evidence/task-284-preparation.md`. The earlier review was `docs/implementation/evidence/task-284-review.md` with five important findings and one nit. This re-review independently inspected the current source, generated contracts, backend integration tests, managed real-stack artifacts, Task 280 reporting/synchronization, Task 279 capability and cleanup boundaries, the finding ledger/history, and the current task row. The preparation report's older run references were treated as stale and superseded by the fresh managed run `b7dd8224067e397809b99c4e`.

The task row requires production-stack isolation, owner/account isolation and erasure, parsed JSON/CSV privacy, persistence/search/history/diet/cache/worker checks, retry/idempotency and failure injection, truthful reports, safe read-only parameterized proof, and synchronized findings for every non-pass product result.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report and repair evidence are available.
- [x] A task-specific baseline and current changed surface are available.
- [x] `code-review-skill` was invoked exactly once and its relevant language guides were read.
- [x] Review was independent of the repair and used current source rather than stale logs.
- [x] No production code or task-list status was edited by this review.

pre_review_gates_passed: true
blocking_issue: "None."

## 3. Review Baseline and Change Surface

The task-specific baseline is commit `f9a646ffede5c8a63ad787a07eaba7efc0433677`. The worktree contains unrelated prepared Tasks 276–283 changes; the review surface was restricted to Task 284, its saved-diet export contract, the real deletion integration coverage, Task 280 result synchronization, and the inherited Task 279 managed-stack capability and cleanup boundary.

The fresh command `python3 scripts/run-task284-acceptance.py --timeout-seconds 240` returned exit 1 because the real product still exposes the preserved export-owner projection defect. Its managed result matrix is 15 PASS, 2 FAIL, and 0 BLOCKED. The only failed criteria are `P08-SWR043-ACCEPT-01` and `P08-SWR072-STEP-04`; both use `ROOT-T284-EXPORT-OWNER-PROJECTION`, both requirement reports return exit 1, and `P08-SWR073` returns exit 0. The stack state is `cleaned`.

The two product failures do not fail this test-delivery task: the task requires non-pass results to remain visible, synchronized, and truthful. `P08-FIND-284-001` remains OPEN DEFECT in `docs/implementation/04_OPEN.md` and registered in `docs/testing/phase08/finding-history.json`, with the same root cause and the required parsed JSON/CSV retest condition.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Function-level evidence |
|---:|---|---|---|---|
| 1 | P08-SWR043-STEP-01 private A item creation and isolation | Production UI and generated client | PASS | `task284-private-erasure.spec.ts:256-387`; fresh result PASS |
| 2 | P08-SWR043-STEP-02 cross-owner and guessed-ID denial | Equal denial behavior from B context | PASS | owner-isolation scenario; fresh result PASS |
| 3 | P08-SWR043-STEP-03 A JSON export excludes B/global data | Authenticated A and B exports | PASS | owner-isolation export assertions; fresh result PASS |
| 4 | P08-SWR043-STEP-04 selected private-item deletion | Generated DELETE and refreshed export | PASS | owner-isolation scenario selected-item assertion; fresh result PASS |
| 5 | P08-SWR043-ACCEPT-01 no nested owner identity | Parsed JSON and CSV projection scan | PASS as evidence contract | `parseCSV`, `assertExportCSV`, and projection scenario correctly produce the synchronized product FAIL and `P08-FIND-284-001`; no false PASS |
| 6 | P08-SWR072-STEP-01 owner-scoped JSON | Authenticated A and B exports | PASS | owner-isolation JSON checks; fresh result PASS |
| 7 | P08-SWR072-STEP-02 saved data, history, and diet included | Exact export fixtures | PASS | `assertExportCSV` and JSON savedDiets checks; fresh result PASS |
| 8 | P08-SWR072-STEP-03 B/global data excluded | Cross-owner export checks | PASS | owner-isolation scenario; fresh result PASS |
| 9 | P08-SWR072-STEP-04 JSON/CSV projection privacy | Parsed JSON and parsed CSV checks | PASS as evidence contract | projection scenario parses both formats, detects the same product defect, and synchronizes the same root; fresh result FAIL is truthful |
| 10 | P08-SWR072-STEP-05 owner-scoped CSV | Header, structure, contents, exclusions | PASS | RFC 4180 parser plus exact section/field/item/diet assertions; fresh result PASS |
| 11 | P08-SWR072-STEP-06 deleted item absent from export | Generated DELETE and refreshed export | PASS | selected deletion and export assertions; fresh result PASS |
| 12 | P08-SWR073-STEP-01 pending deletion request | Generated account DELETE | PASS | erasure scenario and real service integration test; fresh result PASS |
| 13 | P08-SWR073-STEP-02 pending write lockout | Stale-session and repository write denial | PASS | browser and real PostgreSQL integration assertions; fresh result PASS |
| 14 | P08-SWR073-STEP-03 real worker completion | Production worker and independent proof | PASS | `workerProcess: production`, worker start token, and completion snapshot; fresh result PASS |
| 15 | P08-SWR073-STEP-04 A data/session/cache erasure | Complete read-only before/after proof | PASS | 268 observed parameterized reads, all account surfaces zero, A cache zero, assertionFailures empty |
| 16 | P08-SWR073-STEP-05 pseudonymous receipt and login denial | Receipt and stale-login checks | PASS | receipt pseudonymity, user_id removal, stale access and login denial; fresh result PASS |
| 17 | P08-SWR073-STEP-06 B/global unchanged | Counts and logical row hashes | PASS | B custom-row hash/count, global hash/count, and B cache unchanged; fresh result PASS |

The 15 PASS and 2 product FAIL matrix is therefore a passing result for the task's evidence/reporting contract. The preserved defect is not represented as a reviewer finding or converted into infrastructure failure.

## 5. Changed-Symbol Inventory

| # | Symbol or unit | Kind | Current file and lines | Consumer or boundary | Verification |
|---:|---|---|---|---|---|
| 1 | `load_module` | Python loader | `scripts/run-task284-acceptance.py:20-28` | real-stack and Task 283 helpers | managed run and focused tests |
| 2 | `validate_snapshot_request` | capability validator | `scripts/run-task284-acceptance.py:59-87` | proof observer | malformed, extra-key, nonce/user, and UUID tests |
| 3 | `Task284Harness.application_environment` | runtime configuration | `scripts/run-task284-acceptance.py:98-102` | inherited managed harness | managed real-stack run |
| 4 | `Task284Harness.execute` | lifecycle orchestration | `scripts/run-task284-acceptance.py:104-279` | API, UI, worker, observer | fresh managed run and cleanup state |
| 5 | `observe_snapshot_requests` | proof observer | `scripts/run-task284-acceptance.py:281-308` | browser handshake | fresh before/after proof |
| 6 | `collect_snapshot` and nested `read`, `count`, `row_hash`, `custom_count` | database/cache proof | `scripts/run-task284-acceptance.py:310-457` | owned PostgreSQL and Redis | focused adversarial tests and fresh proof |
| 7 | `cache_leak_count` | Redis leakage proof | `scripts/run-task284-acceptance.py:458-476` | run-owned Redis | fresh A/B cache proof |
| 8 | `write_backend_proof` | proof assertion writer | `scripts/run-task284-acceptance.py:477-524` | Task 280 evidence | fresh proof with zero assertion failures |
| 9 | `write_synthetic_browser` | failure artifact writer | `scripts/run-task284-acceptance.py:525-531` | failure finalization | focused finalizer tests |
| 10 | `combine_results` | result normalizer | `scripts/run-task284-acceptance.py:533-587` | Phase 08 reporter | browser/backend precedence tests and fresh reports |
| 11 | `finalize_reports`, `finalize_failure_reports`, `main` | CLI finalization | `scripts/run-task284-acceptance.py:589-654` | command exit/report contract | fresh exit 1 and clean teardown |
| 12 | `statePath`, `saveState`, `loadState`, `privateItem` | Playwright fixture helpers | `frontend/tests/task284-private-erasure.spec.ts:50-70` | three ordered scenarios | managed real-stack run |
| 13 | `signInAs`, `csrf`, `createCustom`, `createGlobal` | production request helpers | `frontend/tests/task284-private-erasure.spec.ts:72-126` | fixture setup and access checks | fresh browser matrix |
| 14 | `parseCSV` | structural CSV parser | `frontend/tests/task284-private-erasure.spec.ts:128-165` | CSV privacy assertions | exact-column and malformed-row unit behavior |
| 15 | `assertExportCSV` | CSV semantic assertion | `frontend/tests/task284-private-erasure.spec.ts:167-199` | owner-isolation scenario | fresh owner/diet/custom assertions |
| 16 | `collectKeys` | recursive projection scanner | `frontend/tests/task284-private-erasure.spec.ts:201-207` | JSON and parsed CSV cells | fresh projection failure |
| 17 | `snapshot` | proof request adapter | `frontend/tests/task284-private-erasure.spec.ts:209-238` | capability observer | v2 run-owned handshake |
| 18 | `record` | Phase 08 evidence adapter | `frontend/tests/task284-private-erasure.spec.ts:240-254` | requirement reporter | fresh reports and finding synchronization |
| 19 | owner-isolation scenario | Playwright acceptance test | `frontend/tests/task284-private-erasure.spec.ts:256-387` | SW-REQ-043 and SW-REQ-072 | 11 mapped criteria pass |
| 20 | projection scenario | Playwright acceptance test | `frontend/tests/task284-private-erasure.spec.ts:389-412` | two projection criteria | both defects detected and synchronized |
| 21 | deletion and erasure scenario | Playwright acceptance test | `frontend/tests/task284-private-erasure.spec.ts:414-510` | SW-REQ-073 | six criteria pass |
| 22 | `ExportBundle`, `ExportSavedDiet`, `ExportSavedDietEntry`, `NewExportService`, `WithCustomItems` | backend contract/service surface | `backend/internal/userdata/export.go:18-117` | account export route | backend tests and OpenAPI parity |
| 23 | `BuildExport`, `buildBundle`, `decryptField` | backend export assembly | `backend/internal/userdata/export.go:119-238` | repositories and safe DTOs | JSON savedDiets and no owner field tests |
| 24 | `encodeCSV` | backend CSV serializer | `backend/internal/userdata/export.go:240-283` | account export download | exact structural CSV test |
| 25 | `task240FailingCachePurger`, `task240FailureRecordRepository`, `task240ExpiredAttemptPurger` | failure-injection doubles | `backend/internal/app/task240_custom_item_erasure_integration_test.go:30-60` | deletion service tests | real retry/failure and lease tests |
| 26 | `runTask240CustomItemErasureIntegration` | PostgreSQL/Redis integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:63-266` | production app and repositories | real failure, partial completion, retry, receipt, and isolation |
| 27 | `TestTask240ProcessingLeaseRecoversFailureRecordOutage` and `TestTask240ExpiredAttemptCannotFinalizeReclaimedWork` | retry/race integration tests | `backend/internal/app/task240_custom_item_erasure_integration_test.go:444-563` | worker lease/finalization | normal and race runs |
| 28 | generated `ExportBundle` and `ExportSavedDiet` plus generator emitters | generated contract surface | `frontend/src/lib/api/generated.ts:750-769`; `scripts/generate-api-types.py:1388-1407` | runtime client and tests | generator drift check and type tests |
| 29 | `buildAccountExportUrl`, `buildAccountExportRequestInit`, `loadAccountExport`, `assertSavedDiet`, `deletePrivateCustomItem`, `readBoundedText` | runtime client and validation | `frontend/src/lib/api/account-data-client.ts:10-77` | private data UI/API | runtime tests for savedDiets, owner keys, size, CSRF, and 204 |
| 30 | generated/client export fixtures and adversarial tests | TypeScript tests | `frontend/src/lib/api/generated.test.ts:56-99`; `frontend/src/lib/api/account-data-client.test.ts:1-48` | generated/runtime contract | frontend check |
| 31 | backend export fixtures and CSV/JSON tests | Go tests | `backend/internal/userdata/export_test.go:31-285` | export DTO and serializer | focused Go tests and race suite |
| 32 | focused Task 284 acceptance tests | Python tests | `scripts/test_run_task284_acceptance.py:14-149` | runner capability/report contracts | 9 focused tests |
| 33 | Phase 08 reporter/synchronizer and managed capability boundary | shared orchestration surface | `frontend/tests/phase08-acceptance-reporter.ts`, `scripts/phase08_acceptance.py`, `scripts/run-real-stack-e2e.py`, `frontend/playwright.real-stack.config.ts` | result roots, process identity, DB/Redis isolation, cleanup | quick gate, harness tests, fresh cleaned run |

inventory_source_count: 33
audited_symbol_count: 33
inventory_complete: true
generated_groupings:
  - "Row 28 groups generated TypeScript output with its generator emitter because the generator is the source of truth and both output and source were inspected, drift-checked, and hashed."
  - "Rows 30-33 group test fixtures or shared orchestration only where the listed files and named boundaries are one verification surface; each named function or test boundary has its own audit row or is explicitly listed in the row."

## 6. Function-Level Audit

| Symbol or unit | Contract and invariants | Normal, edge, and error paths | State, resources, cancellation, and concurrency | Security boundaries | Performance, allocations, and I/O | Simplicity, API, and idioms | Tests and adversarial coverage | Result |
|---|---|---|---|---|---|---|---|---|
| `load_module` | Loads only the two named repository helper scripts. | Missing loader/module errors propagate. | No persistent state. | Does not add a package or secret surface. | One bounded module load per helper. | Small standard-library loader. | Used by managed runner and focused tests. | PASS |
| `validate_snapshot_request` | Enforces exact v2 shape, filename/run ID, nonce, users, phase, and UUID syntax. | Rejects extras, malformed IDs, wrong nonce/users, and invalid deletion request. | Pure validation before any query; no shared mutation. | Capability identity is required before proof access. | Constant-size validation. | Clear allowlisted contract. | Focused tests cover malformed, extra-key, mismatch, and valid requests. | PASS |
| `Task284Harness.application_environment` | Adds only explicit managed-run markers to inherited environment. | Inherited environment construction errors propagate. | No state beyond returned map. | Does not weaken loopback or test-database boundary. | Constant-size map copy. | Minimal override. | Managed run proves production wiring. | PASS |
| `Task284Harness.execute` | Owns disposable DB, Redis, API, frontend, worker, capability, observer, evidence, and cleanup. | Startup, browser product failure, observer timeout, and subprocess errors are surfaced; product non-pass remains reportable. | Observer joins in both paths; port reservations release; worker identity is captured. | No `FLUSHALL`, `TRUNCATE`, or broad cleanup; private capability is mode 0600 and request directory mode 0700. | Bounded process timeouts and local evidence writes. | Lifecycle is linear and guarded by `finally`. | Fresh run returns expected product exit 1 and state `cleaned`. | PASS |
| `observe_snapshot_requests` | Answers only validated browser proof handshakes and records snapshots. | Query errors become explicit snapshot-error evidence rather than success. | Deduplicates request IDs, stops on event, joins, and removes request files. | Response files are mode 0600; validation precedes collection. | 50 ms bounded polling. | Simple observer thread. | Fresh before/after snapshots and observer shutdown pass. | PASS |
| `collect_snapshot` with nested `read`, `count`, `row_hash`, `custom_count` | Uses an allowlist of all account surfaces, exact fixture rows, and fixed parameterized reads; binds immutable target IDs on first before snapshot and rejects later rebinding. | Rejects non-parameterized, wrong-placeholder, mutating, unknown-table, wrong-row, missing-row, and mismatched deletion queries. | Read-only observations are recorded; snapshots are ordered before/after. | Uses run-owned DB target and no user-controlled SQL identifiers outside allowlists; no PII is emitted in proof. | Multiple explicit counts and hashes are appropriate for acceptance evidence; no unbounded result materialization. | Nested helpers keep query policy local and auditable. | Adversarial unit test rejects arbitrary targets and rebind; fresh proof observes 268 reads. | PASS |
| `cache_leak_count` | Scans every key in the run-owned Redis instance and string values for A identity and fixture leakage. | Handles key types without treating non-string values as readable strings. | Uses SCAN and does not delete or flush keys. | Run-owned Redis and identity allowlist prevent cross-run destructive inspection. | Bounded SCAN cursor loop and value reads. | Narrow evidence helper. | Fresh proof reports A cache 1 to 0 and B cache unchanged. | PASS |
| `write_backend_proof` | Compares exact A erasure, retry, receipt, B/global hashes, and cache invariants; emits observed read-only/parameterized/run-owned metadata. | Missing snapshots, changed invariants, non-completed deletion, retry drift, and assertion failures are explicit. | Uses final after snapshot and captured worker identity. | Receipt must be pseudonymous and no forbidden owner data is accepted. | Small JSON artifact. | Assertions are centralized and deterministic. | Fresh proof has `assertionFailures: []`, retryCount preserved, 268 observations, and production worker. | PASS |
| `write_synthetic_browser` | Produces a non-pass browser artifact when setup/infrastructure prevents the real browser matrix. | Failure status is explicit and safe. | No external state. | Does not manufacture a product PASS. | Small JSON write. | Appropriate fallback. | Failure finalizer tests cover all mapped rows. | PASS |
| `combine_results` | Preserves every mapped criterion and lets proof failures override browser success. | Browser FAIL, BLOCKED, missing rows, proof mismatch, and synchronized root errors remain non-pass. | Deterministic ordering and safe evidence allowlists. | Redacts or limits evidence paths and request IDs. | In-memory criteria list. | Correct status normalization. | Focused result tests plus fresh 15/2/0 reports. | PASS |
| `finalize_reports`, `finalize_failure_reports`, `main` | Produces requirement reports and truthful process exit status. | Expected product failure returns 1; infrastructure failure returns BLOCKED/nonzero; signals and exceptions still clean resources. | Harness cleanup and report subprocess handling are bounded. | No false success on command or cleanup error. | Three report writes and bounded process control. | Conventional CLI finalization. | Fresh managed exit 1 matches two synchronized product failures. | PASS |
| Playwright fixture helpers | Preserve run state and exact fixture IDs across ordered scenarios. | Missing state or malformed fixture data fails closed. | State file links separate tests without reusing user credentials in output. | Run-owned artifact path and redaction boundary. | Small JSON state. | Minimal test fixture coupling. | Fresh managed matrix runs all 17 criteria. | PASS |
| Production request helpers | Use production API/generated request paths for login, CSRF, private creation, and global creation. | HTTP status, UUID, and denial errors fail assertions. | Context/session ownership is explicit. | Cross-owner requests use separate authenticated contexts. | Small request bodies. | Test helpers match application boundaries. | Owner isolation and erasure scenarios pass. | PASS |
| `parseCSV` | Parses quoted RFC 4180 fields including commas, escaped quotes, and newlines and returns rows only. | Rejects unclosed quotes and rows not exactly three fields. | Pure parser, no external resources. | Structural parsing prevents substring-only privacy claims. | Linear scan with row/field allocation proportional to CSV. | State machine is direct and auditable. | CSV rows are parsed before semantic assertions. | PASS |
| `assertExportCSV` | Requires exact header, allowed sections, user fields, custom JSON payloads, diet rows, and entry rows. | Rejects unexpected fields/sections, missing/extraneous IDs, malformed JSON, owner keys, and wrong values. | Pure assertions on downloaded data. | Recursively checks owner and ownerId projection leakage. | Bounded by export response and fixture sizes. | Exact assertions make failures actionable. | Fresh owner CSV and projection scenario exercise it. | PASS |
| `collectKeys` | Recursively enumerates object keys in JSON and parsed custom CSV cells. | Handles arrays, objects, and primitive values. | Pure recursion over one export projection. | Detects owner, ownerId, and UserID leakage. | Linear in parsed projection size. | Small reusable scanner. | Fresh product defect is found, not hidden. | PASS |
| `snapshot` | Writes a v2 request containing nonce, run-owned users, exact targets, phase, and deletion request, then waits for a response. | Timeout, error schema, wrong phase, and malformed response fail closed. | File handshake is bounded and request IDs are unique. | Capability nonce and target IDs are supplied by managed capability. | Small files and bounded polling. | Appropriate independent-proof adapter. | Before/after proof succeeds in fresh run. | PASS |
| `record` | Attaches criterion status, sanitized evidence, request IDs, and finding root to Task 280 reporter. | Non-pass requires one synchronized root; report write failures remain visible. | No hidden retries or status rewriting. | Evidence path/request identity is sanitized. | Small report records. | Clear reporting boundary. | Fresh failed criteria have identical root and ledger entry. | PASS |
| owner-isolation scenario | Proves A/B ownership denial, export contents, saved diets, custom item deletion, and accessibility path via production routes/UI. | Denial bodies are compared and missing/extraneous export data fails. | Separate contexts and fixture state are retained for later erasure. | No existence disclosure and no cross-owner mutation. | Fixture-sized API/browser operations. | Scenario follows requirement order. | 11 criteria PASS in fresh matrix. | PASS |
| projection scenario | Proves parsed JSON and parsed CSV projections contain no nested owner identity. | Any owner/UserID key creates a product FAIL with the canonical root. | Independent context and report record; no weakening after failure. | Projection boundary is checked after serialization, not only database state. | Recursive scan over bounded export. | Direct detector. | Fresh run detects both known projection failures. | PASS |
| deletion and erasure scenario | Proves request, lockout, worker completion, complete A erasure, receipt, denial, and B/global survival. | Repeated DELETE, stale writes/access/export, login denial, and missing counts fail. | Polls completion with timeout; verifies idempotent repeat and cache state. | Full account and cache leakage boundary. | Bounded 50-second completion poll. | Requirement-aligned sequence. | Six SW-REQ-073 criteria PASS in fresh matrix. | PASS |
| Backend export DTOs and service constructors | ExportBundle requires savedDiets and safe diet entry DTOs carry no owner field. | Optional repository dependency remains supported; repository errors propagate. | Context is passed to all repositories. | DTO is a projection boundary, not a repository entity leak. | Capacity-sized slices. | Existing service API preserved. | Backend JSON/CSV and runtime tests. | PASS |
| `BuildExport`, `buildBundle`, `decryptField` | Reads user-owned repositories and maps saved diets/entries to owner-free export values. | Missing user, decryption, profile, diet, or item errors return without partial success. | Context and repository ownership are preserved. | User ID is only the top-level established export field; diet DTO omits ownership. | Bounded repository list calls. | Explicit mapping is easy to inspect. | SavedDiets JSON checks and generator/runtime parity. | PASS |
| `encodeCSV` | Emits a three-column structured CSV with user, saved item, saved diet, entry, history, consent, and custom sections. | JSON marshal and writer errors propagate; empty custom section is explicit. | Pure serialization. | No repository identity fields beyond the declared export user field. | Buffer scales with export size; no duplicate query. | Standard `encoding/csv`. | Exact CSV test and browser structural parser. | PASS |
| Failure-injection doubles | Inject cache purge and failure-record outages while preserving real repository behavior; expiry purger honors cancellation. | Errors are typed connection failures and context cancellation. | Channels coordinate lease expiry without data races. | Test-only collaborators cannot be used by managed production harness. | No unbounded goroutines in test cleanup. | Small explicit doubles. | Real app integration and race suite. | PASS |
| `runTask240CustomItemErasureIntegration` | Exercises real PostgreSQL, Redis, production wiring, transactional cleanup, failed cache attempt, retry, receipt, and survivor invariants. | First attempt fails after DB cleanup; retry succeeds once; completed work is not reclaimed. | Real timestamps, leases, Redis keys, and transaction state are checked. | Tests owner isolation and pseudonymous receipt leakage. | Integration cost is appropriate and isolated by test DB/Redis fixtures. | Direct assertions at service and HTTP boundaries. | Normal and `-race` runs pass. | PASS |
| Lease/outage/concurrency integration tests | Preserve failure-record outage recovery, expired-attempt finalization rejection, production worker, and concurrent-write serialization. | Outage and stale lease paths are asserted rather than skipped. | Covers cross-goroutine cancellation and reclaim races. | Worker cannot finalize a reclaimed request. | Real DB synchronization is explicit. | Existing neighboring coverage retained. | Full app race suite passes. | PASS |
| Generated export contract and generator emitters | OpenAPI, generated TS, and generator all require closed savedDiets with exact fields and entries. | Generator drift or missing required field fails checks. | Pure generated contract. | Closed schemas reject undeclared owner projection fields at contract boundary. | Static generation. | Single source of truth retained. | `generate-api-types.py --check`, OpenAPI lint, type tests. | PASS |
| Runtime client and validators | Requires all export arrays, validates saved diets and owner-key absence, bounds response size, and requires exact 204 empty deletion response. | Missing field, invalid UUID/name/entry, oversized body, bad CSRF/status, or nonempty delete response rejects. | Abort signal is propagated; response body is bounded. | Runtime decoder is a second projection boundary. | 1 MiB cap and bounded text reads. | Explicit error type and small API surface. | Account client tests and full frontend check. | PASS |
| Generated/client export fixtures and adversarial tests | Fixtures exercise savedDiets parity and malformed/owner/size/status rejection. | Test globals restore and rejected promises are asserted. | No persistent external state. | Tests owner-field and denial boundaries. | Unit-sized fixtures. | Bun test idioms. | Included in 537 passing frontend tests. | PASS |
| Backend export fixtures and CSV/JSON tests | Verify exact CSV structure and safe JSON saved-diet projection plus dependency errors. | Empty/nonempty CSV, missing dependencies, and decrypt errors are covered. | In-memory repositories isolate unit behavior. | Assert no UserID/owner fields in diet projection. | Small test fixtures. | Standard Go tests. | Focused Go package tests and race. | PASS |
| Focused Task 284 acceptance tests | Protect exact manifest coverage, capability shape, target binding, fail-closed reporting, production worker, no destructive proof SQL, and generated capability wiring. | Tests reject arbitrary targets, rebinding, and missing contract pieces. | Pure tests; subprocess behavior is source-checked where appropriate. | No FLUSHALL/TRUNCATE and managed capability required. | Fast static contract checks. | Nine focused tests with direct failure messages. | All pass. | PASS |
| Reporter, synchronizer, managed harness, and Playwright capability boundary | Enforces process identity, loopback/test DB, run-labeled Redis, safe cleanup, criterion roots, and truthful report status. | Invalid capability or cleanup state blocks; product FAIL is not converted to PASS. | Process start tokens, isolated ports, observer teardown, and cleanup state are verified. | Prevents arbitrary DB/Redis access and stale run reuse. | Bounded health checks and cleanup. | Shared boundary is centralized. | quick gate, real-stack harness tests, and fresh `cleaned` state. | PASS |

## 7. Findings

The prior five important findings and one nit are fixed and re-audited above. No new blocking, important, or optional code-review finding remains.

| Severity | File and line | Symbol | Problem | Evidence or trigger | Required repair or disposition |
|---|---|---|---|---|---|
| None | None | None | No unresolved review finding. | All prior findings have current source, focused tests, race/integration evidence, and fresh managed artifacts. | None. Keep the independent product finding open until its required retest passes. |

blocking_findings: 0
important_findings: 0
optional_findings: 0

### Prior finding closure audit

| Prior finding | Current evidence | Closure |
|---|---|---|
| Proof IDs were syntactically valid but not semantically bound to the exact run-owned snapshot capability. | `validate_snapshot_request:59-87`, `collect_snapshot:310-457`, `test_snapshot_collection_rejects_arbitrary_database_targets_and_rebinding`, fresh capability nonce/users/fixture targets, and 268 observed queries. | FIXED |
| CSV checks used raw strings instead of structural assertions. | `parseCSV:128-165`, `assertExportCSV:167-199`, projection parsing of CSV cells, exact rows/fields/sections, fresh SW-REQ-072-STEP-05 PASS. | FIXED |
| `savedDiets` was missing from OpenAPI/generated/runtime/backend parity. | OpenAPI `ExportBundle` and `ExportSavedDiet`, generator, generated TS, runtime validator, backend `buildBundle` and `encodeCSV`, generated/client/backend tests. | FIXED |
| No real PostgreSQL/Redis failure injection, retry, partial-completion, idempotency, or race proof. | `Task284RealRetryFailureInjectionAndPartialCompletion`, real `openDailyDietAPIIntegrationDB` and `openTask206Redis`, real production app wiring, cache failure, retry count/lease assertions, retry non-reclaim, and app/repository/userdata/deletionworker/cache race suites. | FIXED |
| Erasure/leakage proof covered only a subset of account surfaces and one prefix. | `collect_snapshot` allowlist covers users, OAuth, sessions, reset, profile, saved items/diets/entries, history, consent, entitlements, usage, idempotency, custom items/classifications, receipt, B/global hashes, and full run-owned Redis key/value scan. | FIXED |
| Proof metadata was reported from literal constants rather than observed facts. | Query observations derive read-only and parameterized flags and count; worker PID/start token and run-owned cache label are observed; fresh proof records true, true, 268, production, and run-owned. | FIXED |
| Preserved `P08-FIND-284-001` could be lost or hidden by repaired evidence. | `04_OPEN.md` and finding history retain the finding; fresh run fails exactly the two projection criteria with `ROOT-T284-EXPORT-OWNER-PROJECTION`, reports exit 1, and state is cleaned. | TRUTHFUL AND PRESERVED |

## 8. Commands Run

| Command | Working directory | Exit | Evidence or result |
|---|---|---:|---|
| `python3 -m unittest -v scripts/test_run_task284_acceptance.py scripts/test_phase08_acceptance.py scripts/test_generate_api_types.py` | repository root | 0 | 58 tests passed |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | repository root then frontend | 0 | 537 tests and 2818 expectations passed; type/build/API drift checks passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/userdata ./internal/deletionworker ./internal/cache` | backend | 0 | Focused backend packages passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/app` | backend | 0 | App integration, retry, worker, outage, lease, and concurrency coverage passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/repository` | backend | 0 | Repository race suite passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/userdata ./internal/deletionworker ./internal/cache` | backend | 0 | Export/deletion/cache race suites passed |
| `cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | backend | 0 | No reachable vulnerabilities reported |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | Generated API types match OpenAPI |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | OpenAPI valid; existing OAuth callback warning only |
| `python3 scripts/validate-traceability.py` | repository root | 0 | Traceability passed |
| `python3 scripts/phase08_acceptance.py validate` | repository root | 0 | 12 scenarios and 91 criteria validated |
| `python3 scripts/validate-task-list.py` | repository root | 0 | 286 sequential tasks validated; Task 284 remains PREPARED |
| `python3 scripts/test_run_real_stack_e2e.py` | repository root | 0 | 28 managed-stack harness tests passed |
| `python3 scripts/check.py --quick` | repository root | 0 | Static, changed backend, frontend, and browser lanes passed |
| `python3 scripts/run-task284-acceptance.py --timeout-seconds 240` | repository root | 1 | Expected truthful product non-pass: 15 PASS, 2 FAIL, 0 BLOCKED; only the preserved projection root fails; cleanup `cleaned` |
| `git diff --check` | repository root | 0 | No whitespace errors |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-284-review.md` | repository root | 0 | This evidence is structurally valid |

## 9. Files Inspected and Staleness Fingerprints

Every current reviewed implementation file and current authoritative artifact is hashed with SHA-256. The prior review was stale because its implementation/report hashes and managed run were from the pre-repair evidence. The preparation report is also stale as a run record because it names an earlier managed run; its source repair claims were rechecked against current code and the fresh `b7dd8224067e397809b99c4e` artifacts.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `scripts/run-task284-acceptance.py` | managed runner, capability, proof, report finalization | prior proof/coverage findings fixed | SHA-256 | `8bb78f52424c0a311caa70f0b8572459a5c0ae40f0e279981e022cadafba14d9` |
| `scripts/test_run_task284_acceptance.py` | adversarial runner and report tests | prior proof-binding gaps covered | SHA-256 | `24e6446337b14107935c9b394df258acc20f031c7652f3f2015476da44c544d2` |
| `frontend/tests/task284-private-erasure.spec.ts` | real-stack owner, projection, CSV, and erasure scenarios | prior CSV/erasure gaps fixed | SHA-256 | `d5fcf6ca706a2dce8e9bd89254ec43593020eb266f3e8f83cddbc5742a70f0e7` |
| `frontend/playwright.real-stack.config.ts` | managed capability/process boundary | required capability enforced | SHA-256 | `cb7f6585eadce12097f62bb7b4d23ea20032fdb195c1e3c876f79555b64d2f11` |
| `frontend/src/lib/api/generated.ts` | generated savedDiets contract | parity fixed | SHA-256 | `a9d34a70bc39d217eba95923fb3cf5da993d35c38beefe7ba8f8db44148d400c` |
| `scripts/generate-api-types.py` | generated contract source | parity fixed | SHA-256 | `fc87002ba7330b3ab050885df4058568a4acb313d91b01dbf335a1d236d08701` |
| `frontend/src/lib/api/account-data-client.ts` | runtime export/delete decoder | savedDiets and safety checks fixed | SHA-256 | `eda6183fe3e1a8541efdf5a33f8218976c42289e6d2923454af976731d55f61c` |
| `frontend/src/lib/api/account-data-client.test.ts` | runtime adversarial tests | savedDiets/owner/size/status coverage | SHA-256 | `0e7f2063e73dc4d5d2f75cc71996c413af39539749f85dc2a029c992fdf74015` |
| `frontend/src/lib/api/generated.test.ts` | generated type contract test | savedDiets parity coverage | SHA-256 | `71e6bbcb44d2ca5b8ae264e5fc24a02d5ae2d283edb5e87d41a6b26d70f703bf` |
| `backend/internal/userdata/export.go` | backend export DTO, diet mapping, CSV | savedDiets parity fixed | SHA-256 | `ae55cb9602227cb9d84915fa38a80b34fe93474b2a4edb2e0a2f4454e1f3c8c4` |
| `backend/internal/userdata/export_test.go` | backend export JSON/CSV tests | structural and owner-free assertions | SHA-256 | `769780aa91ad12fcdde2e8e0da6f4b19851bda55727d5cf8b4db8aa035cd1cd8` |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | real DB/Redis failure, retry, receipt, lease, race coverage | prior injection gap fixed | SHA-256 | `75192f9b4a6336a9042a817145b37cfb6c28a46dc3adf6c183a9a6d8089d9203` |
| `scripts/run-real-stack-e2e.py` | loopback DB/Redis isolation and cleanup | run-owned target verified | SHA-256 | `52647965aead4bf16f87e6c1ae4fc4a8c6a7a9c20af4b1ea45921089f8aa140d` |
| `scripts/phase08_acceptance.py` | result normalization and finding synchronization | truthful non-pass roots preserved | SHA-256 | `b36ea8188e3e6d7e05ae17dc992c84cf493c9ac78a8a85706d51e7e40ca34bdb` |
| `frontend/tests/phase08-acceptance-reporter.ts` | criterion evidence/report boundary | synchronized result contract | SHA-256 | `6dcc854de96195876777fb34dde079d034cc42f96f45e789db9cae135cc41af9` |
| `api/openapi.yaml` | export contract source | savedDiets required and closed | SHA-256 | `8d5816d02f02f6bb43c5a2a0d8c4e7ea35aa4051c7b6e90f27fea002a74b16b5` |
| `docs/implementation/04_OPEN.md` | current product finding ledger | P08-FIND-284-001 preserved OPEN DEFECT | SHA-256 | `10f982da0fa6077b81c6c17bc2d14633cfb5fe0521ce643a63d559c7f673639c` |
| `docs/testing/phase08/finding-history.json` | finding registration history | root remains registered | SHA-256 | `ffa91399e166ab3e943966ee7ba1f60ed33c13620e9f3803ddd1cdf00ac03aa2` |
| `docs/implementation/02_TASK_LIST.md` | task status control | unchanged, Task 284 PREPARED | SHA-256 | `01e858020bc72b7599c4fb0adac5dac4f4037ed33f42a5ab4b9c6672efb48cd9` |
| `logs/real-stack-e2e/b7dd8224067e397809b99c4e/acceptance/backend/task284-proof.json` | fresh independent proof | exact erasure and leakage evidence | SHA-256 | `dfc9d5f4bad4c02ba6682a9e5d6ee12f862518487a70412802a4520d1caedef6` |
| `logs/real-stack-e2e/b7dd8224067e397809b99c4e/acceptance/results.json` | fresh criterion matrix | 15 PASS, 2 FAIL, 0 BLOCKED | SHA-256 | `dcc2096142b9ac6d894917586f1ddf1389c6011d34a80cbdb9a71af84fe6eccf` |
| `logs/real-stack-e2e/b7dd8224067e397809b99c4e/state.json` | fresh cleanup proof | state `cleaned` | SHA-256 | `957ebdbc89691f2cd085ff3a243a5a8db277ecba26c1c3db9f4813223ffe37aa` |
| `logs/phase08-acceptance/task284-b7dd8224067e397809b99c4e-sw-req-043/report.json` | synchronized SW-REQ-043 report | projection root preserved | SHA-256 | `d37e41766f98e5ad6b2a3ea8915309b15abf5d217591523abcdb0b10ee70ed70` |
| `logs/phase08-acceptance/task284-b7dd8224067e397809b99c4e-sw-req-072/report.json` | synchronized SW-REQ-072 report | projection root preserved | SHA-256 | `7fe7525d970ea31bc2457202523f090512e466e12592733bede9efe8d0e233bd` |
| `logs/phase08-acceptance/task284-b7dd8224067e397809b99c4e-sw-req-073/report.json` | synchronized SW-REQ-073 report | erasure requirement passes | SHA-256 | `882ef6e3ac6dff598db720b201888c2be23f0d2df2029f11b73ab564e45f44cc` |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/evidence/task-284-review.md: pre-repair review hashes and run 68bcf9edf41e8867a4b53b31"
  - "docs/implementation/evidence/task-284-preparation.md: superseded managed run 8bcec0b4e5b1ee53576e5c94"

## 10. Coverage and Exceptions

coverage_required: true
coverage_exception_allowed: true
coverage_passed: true

The focused Task 284 acceptance tests, backend export/deletion integration suites, frontend runtime/generated tests, full app/repository race suites, and managed real-stack matrix cover the repaired behavior. `scripts/check.py --quick` passed. The phase aggregate coverage contract reports 4645/4986 statements, 93.2%, with the current documented phase exception in `docs/implementation/04_OPEN.md`; this is not an unreported Task 284-specific omission. No review decision relies on coverage in place of the function-level audit.

Coverage finding: The documented phase aggregate deviation remains visible and accepted by the phase gate; Task 284's required semantic and integration coverage passed.

## 11. Negative and Regression Checks

- [x] Snapshot requests reject extra keys, malformed UUIDs, wrong nonce/users, and target rebinding.
- [x] Proof SQL is allowlisted, parameterized, read-only, and does not use `FLUSHALL`, `TRUNCATE`, or broad destructive cleanup.
- [x] CSV privacy is based on parsed structure and exact semantic rows, not raw substring checks.
- [x] Saved-diet OpenAPI, generated, runtime, backend, JSON, CSV, and test surfaces agree.
- [x] Real PostgreSQL and Redis failure injection observes partial completion and retry state; completed requests are not reclaimed.
- [x] Full erasure proof covers account tables, custom classifications, receipt, cache keys/values, B/global hashes, and stale access denial.
- [x] Real production worker identity is observed and worker failure/lease/concurrency tests pass under race detection.
- [x] `P08-FIND-284-001` remains OPEN DEFECT and every current product non-pass points to its canonical root.
- [x] Managed disposable DB/Redis cleanup completes with state `cleaned`.
- [x] Task-list SHA-256 is unchanged and no task-list status cell was edited.

## 12. Decision

A task may be accepted when the evidence scenario contract, exhaustive changed-symbol audit, current artifacts, and review gates pass. The product projection defect is intentionally a synchronized product result rather than an undisclosed test-delivery failure.

decision: PASSED
reason: "All repaired proof, parsing, contract-parity, real failure-injection/retry/race, erasure/leakage, and truthful-finding requirements pass review; the only product failures remain correctly reported under P08-FIND-284-001."
failed_criteria:
  - "None for the Task 284 test-delivery evidence contract; P08-SWR043-ACCEPT-01 and P08-SWR072-STEP-04 are intentionally reported product FAIL results."
unaudited_symbols: []
next_action: "None for this review. Keep P08-FIND-284-001 open and rerun the managed projection retest before product/UAT acceptance."

## 13. Repair Context

This is a successful re-review after the prior repair cycle. The previous important findings and nit are closed by the closure audit in Section 7. No repair instructions are required. The remaining export-owner projection defect is preserved as a product finding and is not silently repaired, weakened, or converted into a reviewer finding.
