# Review Evidence: Task 281 — DESIGN-009 AdminController

~~~yaml
task_id: 281
component: "Phase 08.02 Administrative Authorization Acceptance Scenarios"
static_aspect: "DESIGN-009: AdminController"
input_status: "PREPARED"
review_decision: "PASSED"
decision: "PASSED"
reviewed_at_utc: "2026-07-28T06:12:36Z"
review_agent: "independent-task-281-final-review"
reviewer_instructions: "/home/wiktor/.codex/agents/reviewer.toml"
evidence_file: "docs/implementation/evidence/task-281-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guides: "Python and TypeScript"
repair_context_required: false
pre_review_gates_passed: true
inventory_source_count: 53
audited_symbol_count: 53
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 1. Task Source

Description: Task 281 executes isolated real-stack SW-REQ-054 scenarios for
anonymous, ordinary-user, administrator, malformed, stale, and spoofed-identity
contexts. It must produce real API, PostgreSQL, Redis, generated-client, and
browser evidence and synchronize every non-pass result through Task 280. The
authoritative task row remains PREPARED; this review did not edit its status.

Depends On: 276, 279, 280. All three dependency rows are PASSED.

Testing Coverage Exceptions: None in the task row. Existing repository-wide
coverage exceptions were not used to waive a Task 281 acceptance or symbol
audit failure.

Verification Criteria: Independently named Playwright/API scenarios cover
every SW-REQ-054 manifest criterion with real API, PostgreSQL, Redis,
generated-client, and browser evidence without route interception. Assertions
cover anonymous 401, ordinary-user 403, verified-admin success, spoofed
header/body identity denial, malformed/stale-claim denial, undocumented-route
denial, no restricted payload/control leakage, sign-out/sign-in claim refresh,
request correlation, and accessible responsive navigation. Repeated and failed
runs leave no fixture state. The report accurately records PASS/FAIL/BLOCKED
and returns the specified exit code; every non-pass has one synchronized Phase
08 finding with owner, evidence, and retest condition. Focused tests, finding
consistency, and traceability pass, and failing scenarios may not be skipped or
weakened.

The source contracts reviewed were req_tests.md:3-16 and
docs/design/DESIGN-009.md:24-38, especially Logic 13, requiring existing
access and refresh sessions to retain signed claims until reauthentication.

## 2. Pre-Review Gates

| Gate | Evidence | Result |
|---|---|---|
| Input status | Current row 281 is PREPARED; task-list status was not changed. | PASS |
| Dependencies | Rows 276, 279, and 280 are PASSED. | PASS |
| Preparation | Updated task-281-preparation.md was read in full. | PASS |
| Baseline | Preparation baseline is f9a646ffede5c8a63ad787a07eaba7efc0433677; unrelated dirty work was excluded. | PASS |
| Reviewer instructions | reviewer.toml was read and applied. | PASS |
| Review skill | code-review-skill was invoked exactly once; Python and TypeScript guides were read. | PASS |
| Template | phase-orchestrator review_checklist.md was read completely before review. | PASS |
| Independence | Fresh re-review; no implementation or task-list status was edited. | PASS |

~~~yaml
blocking_issue: "None. F-281-005 is repaired and independently regression-tested."
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: compared the preparation baseline commit, updated
preparation report, prior rejected review, current source, current tests, and
current hashes. Untracked Task 281 files were inspected directly because a
normal git diff does not include them. The 52-row inventory below covers the
Task 280 report boundary, the Task 281 configuration/reporter/helper/spec
units, the isolated runner and its failure paths, and the registered gates;
configuration and reporter helper functions are audited within their owning
unit rows.

Commands used:

~~~text
git status --short
git rev-parse HEAD
rg -n '^| (276|279|280|281) |' docs/implementation/02_TASK_LIST.md
nl -ba frontend/playwright.real-stack.config.ts
nl -ba frontend/tests/phase08-acceptance-reporter.ts
nl -ba frontend/tests/task281-acceptance-helpers.ts
nl -ba frontend/tests/task281-prebootstrap.spec.ts
nl -ba frontend/tests/task281-postbootstrap.spec.ts
nl -ba scripts/run-task281-acceptance.py
nl -ba scripts/test_task281_acceptance.py
git diff f9a646ffede5c8a63ad787a07eaba7efc0433677 -- scripts/check.py scripts/phase08_acceptance.py
~~~

Pre-existing dirty-worktree changes: earlier Phase 08 implementation, tests,
documentation, and untracked operator work for Tasks 276-280 were preserved.
Only the preparation/prior review Task 281 surface and its shared Task 280
report path were attributed here. No production backend or application
frontend source was changed by this review.

| Changed file | Source | Confidence | Symbols/units |
|---|---|---|---|
| scripts/phase08_acceptance.py | Task 280 report consumed by 281 | HIGH | command_report, build_parser |
| scripts/test_phase08_acceptance.py | Task 280 report regression | HIGH | requirement report test |
| frontend/playwright.real-stack.config.ts | Task 281 config | HIGH | configuration |
| frontend/tests/phase08-acceptance-reporter.ts | Task 281 producer | HIGH | types, reporter, helpers |
| frontend/tests/task281-acceptance-helpers.ts | Task 281 helpers | HIGH | envelope, fixture, UI, evidence |
| frontend/tests/task281-prebootstrap.spec.ts | Task 281 pre phase | HIGH | two tests, type |
| frontend/tests/task281-postbootstrap.spec.ts | Task 281 post phase | HIGH | four tests, type |
| scripts/run-task281-acceptance.py | Task 281 runner | HIGH | lifecycle, evidence, report |
| scripts/test_task281_acceptance.py | Task 281 regressions | HIGH | test class and methods |
| scripts/check.py | Task 281 gate registration | HIGH | acceptance gate and traceability |
| docs/implementation/04_OPEN.md | finding ledger | HIGH | Task 281 findings |
| docs/testing/phase08/finding-history.json | finding registry | HIGH | finding IDs |
| docs/testing/phase08/acceptance-manifest.json-trace.md | report trace sidecar | HIGH | report trace |

Fresh managed evidence is logs/phase08-acceptance/task281-325cf05bb43ce884b9bb89d4/report.json.
Its run cleaned the owned database, Redis container, API, and frontend; stale
cleanup subsequently reported resources=0.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Every SW-REQ-054 criterion has an independently named real-stack scenario. | Nine IDs once, both projects. | PASS | Fresh report has 9 results; desktop and mobile pre/post phases ran. |
| 2 | Anonymous API calls return 401 with no restricted data. | API assertions and report. | PASS | ACCEPT-01 is PASS in fresh report. |
| 3 | Ordinary-user calls return 403 with no restricted data. | API assertions and report. | PASS | ACCEPT-02 is PASS in fresh report. |
| 4 | Verified-admin calls succeed and generated-client mutation is exercised. | API, browser, database, audit. | PASS | ACCEPT-03 is PASS; backend artifact validates 2 mutations and 2 audits. |
| 5 | Spoofed role, user, body, and request identity do not authorize. | Deliberate spoofing and server request IDs. | PASS | ACCEPT-05 is PASS; safeEnvelope rejects spoof sentinel. |
| 6 | Malformed and stale sessions fail closed. | Malformed and stale tests. | FAIL as product result, PASS as detector | Malformed passes. Stale refresh defect reproduces on both viewports and links P08-FIND-281-002. |
| 7 | Undocumented routes are denied. | Direct read and mutation to undocumented endpoints. | PASS | Post-bootstrap API test records 404 for both. |
| 8 | Restricted DOM/API data is absent and correlation IDs are server-derived. | Navigation, envelopes, screenshots. | PASS | Fresh report contains denied screenshots and server-derived IDs. |
| 9 | Existing sessions require sign-out and fresh sign-in after bootstrap. | Stale and fresh sign-in tests. | FAIL as product result, PASS as detector | Fresh sign-in passes; stale browser elevates before reauth. |
| 10 | Responsive desktop/mobile, keyboard, theme, and accessibility behavior is covered. | Both projects, keyboard, theme, axe, screenshots. | PASS | Fresh run passes both projects and fresh-admin UI assertions. |
| 11 | Repeated and failed runs leave no fixture state. | Harness state and stale cleanup. | PASS | State is cleaned; cleanup-stale returned resources=0. |
| 12 | Persistence, audit attribution, and Redis evidence are asserted. | Backend JSON and mismatch regressions. | PASS | validated=true, counts 2/2/2, expected 2, Redis reachable; four mismatch cases fail closed. |
| 13 | Every lifecycle or producer failure produces a complete synchronized report and exit code. | Failure tests, malformed-attachment reproduction, and report artifacts. | PASS | `parseAttachment` rejects malformed typed evidence, `_safe_evidence` independently rejects the injected `evidence[].type=[]`, and Task 280 publishes nine `BLOCKED` rows with the synchronized infrastructure root and exit code 2. |
| 14 | Managed task mode cannot target an unowned or shared system. | Capability validation and foreign-target guard test. | PASS | `validateTask281Capability` binds the exact nonce, run-scoped evidence root, loopback origin, frontend PID/start token, cwd, port, and `--strictPort`; the foreign-base regression exits nonzero. |
| 15 | Focused Playwright/API, finding-consistency, and traceability checks pass. | Unit tests and validators. | PASS | Fresh focused suite ran 35 tests; the malformed-evidence regression, manifest, task-list, traceability, typecheck, build, and changed-area Playwright/unit checks pass. |

The fresh report is intentionally FAIL with 7 PASS and 2 product FAIL results.
That is correct test-delivery evidence; the product defect is synchronized and
does not waive or create a review finding for this test-delivery task.

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Change | Consumer | Tests/evidence |
|---:|---|---|---|---|---|---|
| 1 | command_report | function | scripts/phase08_acceptance.py:754-774 | modified | final report | requirement test |
| 2 | build_parser | function | scripts/phase08_acceptance.py:802-828 | modified | CLI | report test |
| 3 | test_requirement_report_accepts_one_complete_manifest_scenario | test | scripts/test_phase08_acceptance.py:230-273 | added | Task 280 suite | unittest |
| 4 | Playwright configuration block and `validateTask281Capability` | configuration/function | frontend/playwright.real-stack.config.ts:3-120 | modified | specs/Playwright | missing-marker and foreign-base tests; source audit |
| 5 | AcceptanceAttachment | type | frontend/tests/phase08-acceptance-reporter.ts:14-20 | added | reporter | typecheck/fresh |
| 6 | CriterionRun | type | frontend/tests/phase08-acceptance-reporter.ts:22-26 | added | reporter | typecheck/fresh |
| 7 | Reporter constants and `parseAttachment` | constants/function | frontend/tests/phase08-acceptance-reporter.ts:28-125 | added | reporter/combiner | malformed-attachment reproduction; fresh report |
| 8 | Phase08AcceptanceReporter | class and lifecycle methods | frontend/tests/phase08-acceptance-reporter.ts:34-98 | added | Playwright | fresh report; pre-record test |
| 9 | splitEnvironment | function | frontend/tests/phase08-acceptance-reporter.ts:116-118 | added | reporter | typecheck |
| 10 | uniqueEvidence | function | frontend/tests/phase08-acceptance-reporter.ts:120-125 | added | reporter | fresh |
| 11 | SafeEnvelope | type | frontend/tests/task281-acceptance-helpers.ts:8-12 | added | API tests | typecheck/fresh |
| 12 | fixture | function | frontend/tests/task281-acceptance-helpers.ts:15-19 | added | specs | fresh |
| 13 | safeEnvelope | function | frontend/tests/task281-acceptance-helpers.ts:22-26 | added | API tests | fresh |
| 14 | signIn | function | frontend/tests/task281-acceptance-helpers.ts:30-37 | added | UI tests | fresh |
| 15 | openSidebarForControl | function | frontend/tests/task281-acceptance-helpers.ts:40-48 | added | UI tests | fresh mobile |
| 16 | screenshot | function | frontend/tests/task281-acceptance-helpers.ts:51-62 | added | evidence | fresh |
| 17 | recordAcceptance | function | frontend/tests/task281-acceptance-helpers.ts:65-76 | added | reporter | fresh |
| 18 | staleStatePath | function | frontend/tests/task281-acceptance-helpers.ts:80-82 | added | cross-phase state | fresh |
| 19 | anonymous pre-bootstrap test | test | frontend/tests/task281-prebootstrap.spec.ts:19-48 | added | criteria 01 | fresh |
| 20 | ordinary-user pre-bootstrap test | test | frontend/tests/task281-prebootstrap.spec.ts:50-102 | added | criteria 02-05 | fresh |
| 21 | pre-bootstrap SafeEnvelopeWithCsrf | type | frontend/tests/task281-prebootstrap.spec.ts:104-107 | added | ordinary mutation | typecheck |
| 22 | malformed-session test | test | frontend/tests/task281-postbootstrap.spec.ts:22-44 | added | criterion 05 | fresh |
| 23 | stale-session test | test | frontend/tests/task281-postbootstrap.spec.ts:46-71 | added | product detector | fresh |
| 24 | verified-admin/API route test | test | frontend/tests/task281-postbootstrap.spec.ts:73-120 | added | admin routes | fresh |
| 25 | fresh sign-in/UI test | test | frontend/tests/task281-postbootstrap.spec.ts:122-179 | added | client/UI | fresh |
| 26 | post-bootstrap SafeEnvelopeWithCsrf | type | frontend/tests/task281-postbootstrap.spec.ts:181-184 | added | CSRF requests | typecheck |
| 27 | PRE_CRITERIA | constant | scripts/run-task281-acceptance.py:29-34 | added | pre/combiner | unit/fresh |
| 28 | POST_CRITERIA | constant | scripts/run-task281-acceptance.py:35-41 | added | post/combiner | unit/fresh |
| 29 | PROJECTS | constant | scripts/run-task281-acceptance.py:42 | added | project/count | unit/fresh |
| 30 | RESOLVED_MOBILE_NAVIGATION_ROOT | constant | scripts/run-task281-acceptance.py:43 | added | closure link | fresh report |
| 31 | INFRASTRUCTURE_ROOT | constant | scripts/run-task281-acceptance.py:44 | added | failure reports | lifecycle tests |
| 32 | EvidenceAssertionError | exception | scripts/run-task281-acceptance.py:47-49 | added | evidence classification | mismatch tests |
| 33 | Task281Harness | class | scripts/run-task281-acceptance.py:76-353 | added | inherited harness | fresh/harness |
| 34 | Task281Harness.execute | method | scripts/run-task281-acceptance.py:79-188 | added | Harness.run | fresh/lifecycle |
| 35 | run_playwright_phase | method | scripts/run-task281-acceptance.py:190-220 | added | execute | fresh/timeout |
| 36 | write_backend_evidence | method | scripts/run-task281-acceptance.py:221-282 | added | execute | mismatch/fresh |
| 37 | combine_results and `_safe_*_evidence` helpers | static/functions | scripts/run-task281-acceptance.py:284-383 | added | finalization | complete-result and malformed-attachment reproduction |
| 38 | parse_args | function | scripts/run-task281-acceptance.py:385-393 | added | main | CLI |
| 39 | safe_playwright_diagnostics | function | scripts/run-task281-acceptance.py:396-438 | added | failed phase | fresh output |
| 40 | finalize_report | function | scripts/run-task281-acceptance.py:441-484 | added | main | lifecycle tests |
| 41 | finalize_failure | function | scripts/run-task281-acceptance.py:487-507 | added | main exception path | lifecycle tests |
| 42 | main | function | scripts/run-task281-acceptance.py:510-558 | added | entry point | fresh/lifecycle |
| 43 | Task281AcceptanceTests | class and test methods | scripts/test_task281_acceptance.py:26-407 | added | check lane | 10 tests |
| 44 | test_combiner_emits_each_sw_req_054_criterion_once | test | scripts/test_task281_acceptance.py:29-84 | added | combiner | unittest |
| 45 | test_missing_phase_is_blocked_never_skipped_or_passed | test | scripts/test_task281_acceptance.py:86-98 | added | combiner | unittest |
| 46 | test_task_flag_without_managed_harness_fails_closed | test | scripts/test_task281_acceptance.py:100-124 | added | config | unittest |
| 47 | test_backend_evidence_asserts_counts_actor_and_redis | test | scripts/test_task281_acceptance.py:126-153 | added | evidence | unittest |
| 48 | test_backend_evidence_accepts_exact_owned_results | test | scripts/test_task281_acceptance.py:155-177 | added | evidence | unittest |
| 49 | test_malformed_attachment_publishes_synchronized_nonzero_report | test | scripts/test_task281_acceptance.py:231-296 | added | reporter/combiner/Task 280 | unittest |
| 50 | test_lifecycle_failures_emit_complete_results_and_nonzero_report | test | scripts/test_task281_acceptance.py:350-389 | added | failure path | unittest |
| 51 | test_runner_uses_bootstrap_and_read_only_database_evidence | test | scripts/test_task281_acceptance.py:391-406 | added | source contract | unittest |
| 52 | validate_phase08_acceptance_contracts | function | scripts/check.py:637-645 | added | static lane | quick gate |
| 53 | Task 281 TRACEABLE_FILES entries | configuration | scripts/check.py:709-727 | modified | traceability | validator |

~~~yaml
inventory_source_count: 53
audited_symbol_count: 53
inventory_complete: true
generated_groupings:
  - "None; no generated implementation was grouped."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract/invariants | Paths/errors | State/security/concurrency | Performance/idioms | Tests/gaps | Result |
|---|---|---|---|---|---|---|
| command_report | Filters a validated manifest to one requested requirement and delegates synchronized publication. | Valid report returns its truthful code; invalid results return nonzero without a partial artifact. | Task 280 owns atomic publication and finding synchronization. | Small bounded filter. | Requirement-report test and fresh report. | PASS |
| build_parser | Constrains requirement option to report. | Rejects unsupported/missing args. | No resources. | Idiomatic argparse. | Focused test. | PASS |
| requirement report test | Proves one complete SW-REQ-054 report. | Happy path only. | Temporary output cleaned. | Small fixture. | 34-test suite. | PASS |
| Playwright config | Task mode requires a private capability and targets only the exact harness listener/evidence root. | Missing capability, mismatch, non-loopback, malformed process identity, cwd/port/token mismatch all throw before tests load. | Private file/parent, nonce, PID start token, process cwd/cmdline, loopback URL, and strict port bind the boundary; foreign-base test passes. | One bounded config-time check. | Missing-marker and foreign-base tests; process/cwd branches are source-audited. | PASS |
| AcceptanceAttachment | Carries allow-listed reporter data. | Runtime validation deferred. | No sensitive fields. | Type only. | Typecheck/fresh. | PASS |
| CriterionRun | Associates status/project/attachment. | Missing project becomes BLOCKED. | Type only. | Type only. | Fresh aggregation. | PASS |
| CRITERION_PATTERN | Matches stable Phase 08 IDs. | Unrecognized titles produce missing execution, which the reporter marks BLOCKED with the infrastructure root. | Constant. | One regex. | Fresh and pre-record fallback. | PASS |
| Phase08AcceptanceReporter | Emits one sanitized producer result per expected criterion, preserving known roots and mapping absent roots to infrastructure. | Malformed JSON or any malformed attachment field is rejected; the failed run remains represented and the producer emits a safe infrastructure root. | Process-local map and one result write; no credentials stored. | Bounded by test count. | Fresh report, missing-attachment, and malformed `evidence[].type=[]` producer regression pass. | PASS |
| splitEnvironment | Parses controlled comma-separated criterion/project labels. | Empty env is rejected by reporter preconditions. | Configuration values remain non-authoritative; ownership is checked by Playwright config. | O(n), bounded by env. | Fresh reporter env and typecheck. | PASS |
| uniqueEvidence | Dedupes and sorts evidence. | Downstream validates paths. | Pure helper. | O(n) map plus sort. | Fresh report. | PASS |
| SafeEnvelope | Minimal server envelope projection. | Optional data supports denied envelopes. | Keeps untrusted data out of authority. | Type only. | Typecheck/fresh. | PASS |
| fixture | Requires managed fixture values. | Missing value throws before attachment and is normalized by the reporter/runner failure path. | Credentials remain process-local and are sanitized. | One lookup. | Fresh; generic missing-attachment fallback tested. | PASS |
| safeEnvelope | Requires a canonical server-derived, non-spoofed request ID. | Malformed JSON or missing ID throws before attachment and is normalized by the reporter/runner failure path. | Prevents correlation spoofing. | One response decode. | Fresh API paths and pre-record fallback test. | PASS |
| signIn | Uses production controls and waits for login view removal. | Missing/failed login throws before attachment. | Playwright timeout/cancellation applies. | Small UI flow. | Fresh desktop/mobile. | PASS |
| openSidebarForControl | Opens responsive sidebar before hidden control use. | Trigger/control failure is observable. | No leaked resource. | One visibility check. | Fresh mobile. | PASS |
| screenshot | Writes selector screenshots below the capability-bound result root. | Missing selector/root throws before attachment and is normalized by the producer path. | Root deletion inherited from harness; Task 280 rejects unsafe links. | One screenshot. | Fresh screenshots and ownership source audit. | PASS |
| recordAcceptance | Attaches only the Task 280 metadata envelope. | Downstream producer and consumer guards reject malformed fields without exposing raw response data. | No credentials/raw response. | Small JSON. | Fresh, pre-record fallback, and malformed-element regression. | PASS |
| staleStatePath | Separates per-project claim/reauth state. | Missing state fails before attachment. | Temporary raw directory; cookies not persisted. | Constant path. | Fresh state reuse. | PASS |
| anonymous test | Proves anonymous 401/no data/no restricted DOM/correlation. | Early assertion failure is normalized by the closed F-281-004 path. | Harness-managed contexts. | Bounded flow. | Fresh desktop/mobile. | PASS |
| ordinary-user test | Proves 403, CSRF denial, spoofing, denied navigation/state capture. | Login/state failures are normalized by the producer fallback. | Temporary state; ordinary claims denied. | Bounded flow. | Fresh desktop/mobile. | PASS |
| pre-bootstrap CSRF type | Types CSRF envelope. | Type only. | No trust. | Type only. | Typecheck. | PASS |
| malformed test | Proves malformed cookies fail 401. | Setup failure is normalized by the producer fallback. | Explicit API context disposed normally. | One request. | Fresh desktop/mobile. | PASS |
| stale test | Detects Logic 13 violation and records known root before expected assertion. | Intentional product failure is structured. | Reused cookies temporary; context disposed. | Small flow. | Fresh two viewport failures. | PASS as detector |
| verified-admin test | Proves admin read, spoofed mutation rejection, undocumented 404. | Early failures are normalized by the producer fallback. | API context managed. | Bounded calls. | Fresh desktop/mobile. | PASS |
| fresh UI test | Proves explicit reauth, generated mutation, responsive/keyboard/theme/axe. | Early failure is normalized by the producer fallback. | Reauth context disposed; credentials not attached. | Two themes and one mutation. | Fresh desktop/mobile. | PASS |
| post CSRF type | Types post-bootstrap CSRF envelope. | Type only. | No trust. | Type only. | Typecheck. | PASS |
| PRE_CRITERIA | Exactly four pre IDs. | Downstream catches missing/extra. | Immutable constant. | Constant. | Unit/fresh. | PASS |
| POST_CRITERIA | Exactly five post IDs. | Downstream catches missing/extra. | Immutable constant. | Constant. | Unit/fresh. | PASS |
| PROJECTS | Requires desktop/mobile. | Reporter blocks missing project. | Target ownership is separately proved by the capability configuration. | Constant. | Fresh two-project run. | PASS |
| RESOLVED_MOBILE_NAVIGATION_ROOT | Links closed mobile finding only on passing criterion. | Task 280 validates closure. | Constant. | Constant. | Fresh report. | PASS |
| INFRASTRUCTURE_ROOT | Names the synchronized infrastructure blocker. | Missing/invalid producer roots and malformed consumer evidence use it; Task 280 synchronizes it. | Constant. | Constant. | Lifecycle, pre-record, and malformed-evidence tests. | PASS |
| EvidenceAssertionError | Distinguishes evidence mismatch from unavailable infra. | Mismatch FAIL; command/setup BLOCKED. | No sensitive error payload. | Exception only. | Four mismatch cases. | PASS |
| Task281Harness | Reuses owned Task 279 lifecycle. | Inherited run catches execute errors and cleans. | DB/Redis/process/temp resources owned. | One stack. | Fresh cleaned state. | PASS |
| Task281Harness.execute | Orders pre, bootstrap, post, evidence, combine. | Test exit retained; exceptions reach main after parent cleanup. | Reservations release; inherited cleanup. | Builds binaries/one stack. | Fresh/lifecycle. | PASS |
| run_playwright_phase | Executes exact specs with managed env and continues to lifecycle finalization after test failure. | CalledProcessError is retained; timeout/spawn and missing phase artifacts are normalized by main/combiner. | `run_command` kills process groups; inherited cleanup follows. | One subprocess per phase. | Fresh, timeout/lifecycle tests, and pre-record test. | PASS |
| write_backend_evidence | Requires count 2, audit 2, exact fixture actor, Redis PONG. | SQL/Docker errors BLOCKED; mismatch writes false then raises FAIL. | Read-only SQL and owned Redis ping. | Three scalar queries plus one command. | Mismatch tests/fresh artifact. | PASS |
| combine_results | Emits exactly nine criteria, sanitizes partial producer data, and assigns only synchronized roots. | Type guards run before set membership and root selection; malformed `status`, `rootCauseId`, evidence fields, and partial phase files normalize to `BLOCKED` infrastructure rows. | Pure bounded file I/O; no external state or concurrency. | Nine fixed rows and bounded evidence paths. | Missing phase, pre-record, and injected `evidence[].type=[]` tests pass. | PASS |
| parse_args | Defines timeout and defer-report CLI. | Invalid args fail before harness. | Parser only. | Idiomatic. | CLI/syntax. | PASS |
| safe_playwright_diagnostics | Bounded redacted diagnostics. | Non-CPE becomes safe marker; UUIDs and fixture values redacted. | No resource. | 40 lines x 300 chars. | Fresh failure output. | PASS |
| finalize_report | Maps status to exit 0/1/2 and invokes Task 280. | Valid and intentionally failing synchronized results publish; Task 280 rejection is surfaced for fallback. | Called after normal cleanup; publication is checked for file, run ID, and exit code. | One subprocess. | Fresh report and lifecycle tests. | PASS |
| finalize_failure | Synthesizes nine grouped rows for lifecycle errors before finalization. | Empty, missing, partial, and malformed phase paths all normalize to nine rows before Task 280 publication; error class maps to `FAIL` or `BLOCKED` intentionally. | Called after Harness.run cleanup; no resource ownership added. | Deterministic nine-row output. | Three lifecycle cases plus malformed-evidence publication pass. | PASS |
| main | Owns validation, harness, report, and safe exits. | Direct lifecycle failures use the safe fallback; malformed producer attachments are rejected and malformed consumer artifacts are normalized rather than escaping as `TypeError`. | Signal handlers are restored and inherited cleanup remains authoritative. | One lifecycle and bounded fallback. | Fresh run, quick gate, lifecycle, and malformed report tests pass. | PASS |
| Task281AcceptanceTests | Focused runner contract suite. | Covers exact output, missing/foreign inputs, backend mismatches, producer/lifecycle failures, malformed evidence, bootstrap ordering, and read-only boundaries. | Temporary directories and subprocesses are cleaned by test fixtures. | Small table-driven unittest class. | 35 focused tests across Task 280/281. | PASS |
| malformed attachment test | Proves producer rejection and consumer normalization of an unhashable evidence type. | Directly injects malformed phase JSON, then requires nine `BLOCKED` rows, one infrastructure root, synchronized Task 280 output, and nonzero exit. | Temporary evidence/report roots; subprocess producer is bounded. | Small regression fixture. | Passes. | PASS |
| combiner criterion test | Proves nine unique IDs and backend link. | Valid complete phase only. | Temp directory. | Small fixture. | Passes. | PASS |
| missing phase test | Proves absent phase is BLOCKED infrastructure. | Absent phase is explicit `BLOCKED`, never skipped or passed; malformed phase input is covered by the dedicated regression. | Temp directory. | Small fixture. | Passes. | PASS |
| missing-marker test | Proves original task-only invocation fails closed. | Separate foreign-base regression covers the caller-controlled URL case. | Isolated --list subprocess. | Short startup. | Both regressions pass; F-281-001 closed. | PASS |
| backend mismatch test | Proves count/actor/Redis mismatches fail closed. | Four mismatch cases. | Mocked external boundaries. | Table-driven small fixture. | Passes. | PASS |
| backend exact test | Proves exact count/actor/PONG success. | Fresh DB scope is assumed. | Mocked read boundaries. | Small fixture. | Passes and agrees with fresh JSON. | PASS |
| lifecycle failure test | Proves timeout/bootstrap/evidence reports. | Failure class maps to expected nonzero report and all nine rows; malformed publication has a dedicated regression. | Temp report root. | Three tiny cases. | Passes. | PASS |
| runner source-contract test | Proves order/read-only/no hard-coded counts. | Source scan, not runtime error paths. | No resources. | Constant-size scan. | Passes. | PASS |
| validate_phase08_acceptance_contracts | Registers Task 281 tests and manifest validation. | Acceptance lane includes the malformed typed-attachment regression and manifest/finding validation. | Subprocess lane. | One check step. | Quick gate and 35 focused tests pass. | PASS |
| TRACEABLE_FILES entries | Includes Task 281 scripts/tests. | Traceability is not semantic validation. | Configuration only. | Negligible. | Traceability pass. | PASS |

## 7. Findings

### F-281-001 — Closed review finding: managed target ownership

Location reviewed: frontend/playwright.real-stack.config.ts:5-120 and
scripts/run-task281-acceptance.py:124-157.

The repair is sufficient for this review surface. Task mode requires the
current harness marker, a private capability file, an exact nonce, the exact
run-scoped evidence root, an uncredentialed `http://127.0.0.1:<port>/` origin,
and a live frontend PID whose Linux start token, cwd, `--port`, and
`--strictPort` match the capability. The foreign-base regression now exits
nonzero, and the old arbitrary-base reproduction is rejected before tests load.

### F-281-002 — Prior important finding: backend and Redis evidence repaired

The prior observational-evidence finding is fixed. write_backend_evidence now
requires two mutations, two audits, exact fixture actor for all matching audits,
and Redis PONG; it writes validated=true only after all checks pass and raises
EvidenceAssertionError on mismatch. Four mismatch cases and the exact-success
case pass. Fresh backend JSON records counts 2/2/2, expected 2, redisState
reachable, and validated true. Browser metadata no longer contains hard-coded
persistence counts. No unresolved review finding remains for F-281-002.

### F-281-003 — Closed review finding: direct lifecycle report repair verified

The prior lifecycle exception finding is fixed for timeout, bootstrap, spawn,
setup, and backend-evidence failures. Inherited Harness.run cleans owned
resources, finalize_failure synthesizes nine rows under the infrastructure
root, and Task 280 report generation returns the expected nonzero code. The
three lifecycle cases pass and the fresh normal run finalizes after cleanup.

### F-281-004 — Closed review finding: unannotated producer failure

Location reviewed: frontend/tests/phase08-acceptance-reporter.ts:58-97 and
scripts/run-task281-acceptance.py:284-558.

The repair is verified for the requested pre-record failure. A test ending with
`status="failed"` and no `phase08-acceptance` attachment produces a phase
artifact; `combine_results` emits exactly nine rows, assigns the synchronized
`ROOT-T281-ACCEPTANCE-INFRASTRUCTURE`, and Task 280 publishes a nonzero report.
Missing phase files and direct timeout/bootstrap/evidence failures use the same
path. This finding is closed; the malformed-element case below is distinct.

### F-281-005 — Closed review finding: malformed evidence is fail-closed

Location reviewed: frontend/tests/phase08-acceptance-reporter.ts:100-125 and
scripts/run-task281-acceptance.py:313-377.

The repaired producer rejects `evidence[].type=[]` before it can become trusted
attachment data. The independent consumer boundary also checks the runtime type
before set membership, so a directly injected malformed phase artifact becomes
`BLOCKED` rather than raising `TypeError`. The regression then runs Task 280 and
verifies nine result rows, one synchronized
`ROOT-T281-ACCEPTANCE-INFRASTRUCTURE`, report status `BLOCKED`, and exit code 2.

Reproduction and result, run against the current source:

~~~text
Reporter rejected evidence[].type=[] and emitted a safe producer result.
After direct injection into prebootstrap.json, combine_results emitted 9 BLOCKED rows.
Task 280 published task281-malformed-attachment/report.json with status BLOCKED and exitCode 2.
~~~

This finding is closed; the regression remains in
scripts/test_task281_acceptance.py:231-296.

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit | Result | Artifact |
|---|---|---:|---|---|
| python3 -m unittest -v scripts/test_phase08_acceptance.py scripts/test_task281_acceptance.py | repository root | 0 | 35 tests passed, including malformed evidence producer/consumer regression | focused output |
| python3 scripts/phase08_acceptance.py validate | repository root | 0 | 12 scenarios, 91 criteria valid | validator |
| python3 scripts/validate-task-list.py | repository root | 0 | 286 tasks valid | validator |
| python3 scripts/validate-traceability.py | repository root | 0 | traceability valid | validator |
| python3 -m py_compile scripts/run-task281-acceptance.py scripts/test_task281_acceptance.py scripts/phase08_acceptance.py scripts/test_phase08_acceptance.py scripts/check.py | repository root | 0 | syntax valid | compiler |
| git diff --check | repository root | 0 | clean | git check |
| BUN_TMPDIR=... BUN_INSTALL=... bun run typecheck | frontend | 0 | TypeScript passed | tsc |
| BUN_TMPDIR=... BUN_INSTALL=... bun run build | frontend | 0 | 219 modules transformed | Vite |
| GOCACHE=... GOMODCACHE=... go test ./internal/httpapi -run admin-auth-session-regex -count=1 | backend | 0 | focused admin/auth tests passed | Go unit |
| GOCACHE=... GOMODCACHE=... go test -race ./internal/httpapi -run admin-auth-session-regex -count=1 | backend | 0 | focused race tests passed | Go race |
| python3 scripts/check.py --quick | repository root | 0 | static, frontend, changed Playwright (24 pass/14 intentional skip), backend, OpenAPI, and vulnerability lanes passed | aggregate gate |
| python3 scripts/run-task281-acceptance.py --timeout-seconds 240 | repository root | 1 expected | fresh managed run `325cf05bb43ce884b9bb89d4`: 9 criteria, 7 PASS, 2 product FAIL, validated backend, cleanup | fresh report |
| python3 scripts/run-real-stack-e2e.py --cleanup-stale --stale-age-seconds 900 | repository root | 0 | resources=0 | cleanup |
| missing-marker and foreign-base Playwright regressions | frontend via unittest | 0 | task-only and foreign target both rejected before tests load | F-281-001 closed |
| pre-record failure with no attachment through reporter/combiner/finalizer | repository root | 0 | nine synchronized infrastructure rows, Task 280 FAIL report, exit 1 | F-281-004 closed |
| malformed attachment regression (`evidence[].type=[]`) | repository root + frontend | 0 | producer rejects malformed attachment; consumer emits 9 BLOCKED rows; Task 280 report is BLOCKED with exit 2 and infrastructure root | F-281-005 closed |
| direct malformed consumer assertion | repository root | 0 | Injected `evidence[].type=[]` produced exactly 9 BLOCKED rows, every row carried the infrastructure root, and Task 280 returned/published exit code 2 | independent final check |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-281-review.md | repository root | 0 | review evidence structurally valid | final gate |

## 9. Files Inspected and Staleness Fingerprints

All hashes are SHA-256 captured after the fresh run and before this evidence
file was rewritten. They cover every reviewed implementation file and the
read-only evidence artifacts.

| File | Purpose | Finding | Algorithm | Content hash |
|---|---|---|---|---|
| docs/implementation/02_TASK_LIST.md | task source | status preserved | SHA-256 | b25eed9787cf8f855ed01ca167d5ae739fb6108740197376cefe88aec2a5d536 |
| req_tests.md | requirement source | none | SHA-256 | 241779579f568eecfe40a8d41b97ec2b6a87e6201bd25b8ea76c4bac5c2a8902 |
| docs/design/DESIGN-009.md | design source | none | SHA-256 | 704500175a5e465ee6d10cc1edd421f4c7c6043dfff1cd8c444a28e1d75973ea |
| docs/implementation/evidence/task-281-preparation.md | preparation | current rechecked | SHA-256 | 0c60e2505308328d21c544dfbb8e436b9a3950827cdc4478d9c34a676b9387e8 |
| docs/implementation/evidence/task-280-review.md | prior review | stale reference checked | SHA-256 | f600a68b468db44c428da960a633a5c1f5b4bc3dd3a00fce960a4526bdf94f5b |
| scripts/phase08_acceptance.py | report/filter | F-281-004 | SHA-256 | b36ea8188e3e6d7e05ae17dc992c84cf493c9ac78a8a85706d51e7e40ca34bdb |
| scripts/test_phase08_acceptance.py | report tests | gap | SHA-256 | 58d1beec596acb4006174d51c1927297220dbcd9b42a1e7bbdac4521e628ce27 |
| frontend/playwright.real-stack.config.ts | managed target | F-281-001 closed | SHA-256 | 7f7371279c66f88b6b8ee36b8749edfff8fc1662d896123f3d899eb4fbd19628 |
| frontend/tests/phase08-acceptance-reporter.ts | producer | F-281-005 closed | SHA-256 | cc601b14c3dddfb588d7a907ce24e635a20e2985a5c252e069ca04e25f2e30b7 |
| frontend/tests/task281-acceptance-helpers.ts | helpers | F-281-004 boundary | SHA-256 | 49877745607f94497a828de43d0c2d86524de84d665db03342f035f7d480594f |
| frontend/tests/task281-prebootstrap.spec.ts | pre phase | F-281-004 closed | SHA-256 | 080441b5fecdc7a4d837680b5e219edc8cd2964e13364e6d25a8fdb719c89048 |
| frontend/tests/task281-postbootstrap.spec.ts | post phase | F-281-004 closed | SHA-256 | da4ee4c379e02656bdbd6465e9ab903003be4cea9eb092520d7729a4989d9aa7 |
| scripts/run-task281-acceptance.py | runner | F-281-005 closed | SHA-256 | 30f4a17374209667bfe78988413d261658c0f97518a71e33fdf8405ea32ebdf1 |
| scripts/test_task281_acceptance.py | runner tests | F-281-005 regression | SHA-256 | 0499fe3a8385530adea3bfed2deb5ffc03c46b2be429546ac882fb54bf33461f |
| scripts/check.py | quality registration | none | SHA-256 | cf338e08b99979d4b7d8ffa08023833203d9a3696d64ba6b185240e2cd634519 |
| docs/implementation/04_OPEN.md | findings | action needed | SHA-256 | e80965330e09f778545442caa4c216c6b8f58e56480502549ff450a5199029f0 |
| docs/testing/phase08/acceptance-manifest.json | manifest | none | SHA-256 | 2f597b9545bb176e45a03df02d1b05f01fc424b479ffb4b64c43ee3b6991fa8c |
| docs/testing/phase08/acceptance-manifest.json-trace.md | sidecar | none | SHA-256 | a87505099f762aac4432cc00b3516f0758f4ac7b520c95a7cdc0eb2ad1b7c5c3 |
| docs/testing/phase08/finding-history.json | history | retain IDs | SHA-256 | c11c95c59a400acdaec0f887583c55b6650e754da5e77bad560e7df9f60e8ef6 |
| scripts/run-real-stack-e2e.py | inherited harness | inspected | SHA-256 | 52647965aead4bf16f87e6c1ae4fc4a8c6a7a9c20af4b1ea45921089f8aa140d |
| logs/phase08-acceptance/task281-325cf05bb43ce884b9bb89d4/report.json | fresh report | product defect only | SHA-256 | ac113eccdc6914eb7149ac57ff6a2b579637f5a3f84a31512b9b6b3cc72f93af |
| logs/phase08-acceptance/task281-325cf05bb43ce884b9bb89d4/evidence/backend/admin-authorization.json | backend proof | none | SHA-256 | 14b53b5224d0f7beb1b0bec47817b15063890a8c134c47f257ec357ccb36fc67 |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/evidence/task-281-review.md (prior REJECTED evidence superseded by this current review)"
  - "docs/implementation/evidence/task-281-preparation.md (prior recorded hash was stale; current content was rehashed and rechecked)"
  - "docs/implementation/02_TASK_LIST.md (preparation captured OPEN before external PREPARED transition; current PREPARED row was re-read)"
~~~

## 10. Coverage and Exceptions

- [x] Focused contract tests ran and passed.
- [x] Changed-area aggregate checks ran and passed.
- [x] Untested branches relevant to changed symbols were inspected.
- [ ] Separate line-coverage threshold is not part of this task row.

~~~yaml
coverage_required: false
coverage_exception_allowed: false
coverage_report_path: "none; focused behavioral evidence required"
observed_line_coverage: "N/A"
coverage_passed: true
~~~

Coverage finding: focused tests cover repaired count, actor, Redis, missing-phase,
foreign-base rejection, unannotated pre-record failure, direct lifecycle paths,
and both malformed attachment boundaries. No coverage exception was used to
waive an acceptance or symbol-audit branch.

## 11. Negative and Regression Checks

| Check | Evidence | Result |
|---|---|---|
| Anonymous access denied | Fresh report/API/browser assertions | PASS |
| Ordinary-user access forbidden before bootstrap | Fresh pre-bootstrap evidence | PASS |
| Spoofed identity cannot authorize | Fresh API assertions | PASS |
| Undocumented routes denied | Fresh verified-admin test | PASS |
| Stale claims not silently accepted | Fresh test fails and links product root | PASS as detector |
| Fresh sign-out/sign-in enables admin | Fresh UI/API test | PASS |
| Closed mobile finding stays closed | Fresh report links closed root | PASS |
| Backend/audit/Redis mismatch fails closed | Four mismatch cases and fresh artifact | PASS |
| Timeout/bootstrap/evidence errors finalize | Three failure cases | PASS for direct lifecycle |
| Task-only invocation fails closed | Missing-marker regression | PASS |
| Marker plus foreign target is rejected | Capability-bound `--list` regression | PASS — F-281-001 closed |
| Unexpected browser failure publishes a synchronized report | Synthetic unattached FAIL | PASS — F-281-004 closed |
| Malformed attachment publishes a synchronized report | Producer rejection plus direct consumer injection; nine `BLOCKED` rows and Task 280 exit 2 | PASS — F-281-005 closed |
| No duplicate helper or obsolete alias | Changed-surface search | PASS |
| No production authorization boundary changed | Worktree/scope inspection | PASS |

## 12. Decision

~~~yaml
decision: "PASSED"
review_decision: "PASSED"
reason: "All acceptance criteria and 53 audited symbols pass; F-281-005 now fails closed at both producer and consumer boundaries and publishes the synchronized nine-row BLOCKED report."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for Task 281 review; retain the known product defect P08-FIND-281-002 for its required authentication repair and retest."
~~~

The known product defect P08-FIND-281-002 remains correctly observed and
synchronized; it is not itself a review rejection because Task 281 is a
test-delivery task. Prior review findings F-281-001 through F-281-004 are
repaired within their direct scopes, and F-281-005 is closed by the current
malformed-evidence regression. The known product defect is still intentionally
reported as FAIL and is not a test-delivery review finding.

Validation run for this decision:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-281-review.md
~~~

## 13. Repair Context

Not applicable: this review is PASSED. The prior repair context was rechecked
against the current producer, consumer, regression, report, and hash evidence;
the task-list status remains PREPARED and was not edited.
