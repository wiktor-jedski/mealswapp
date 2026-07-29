# Review Evidence: Task 285 — DESIGN-014 LogAggregator

~~~
task_id: 285
component: "Phase 08.02 Centralized Logging Acceptance Scenarios"
static_aspect: "DESIGN-014: LogAggregator"
input_status: "PREPARED"
review_decision: "REJECTED"
reviewed_at_utc: "2026-07-28T19:41:37Z"
review_agent: "independent-task-285-reviewer"
evidence_file: "docs/implementation/evidence/task-285-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677 plus current task-owned working-tree paths"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Python, TypeScript, security, async/concurrency, common-bugs"
repair_context_required: true
~~~

## 1. Task Source

Description: Task 285 implements a deployed-only SW-REQ-084 acceptance verifier. It must generate production actions through Playwright/API, query GCP Cloud Logging independently, correlate safe request IDs, enforce timestamps, outcomes, redaction and retention, remain truthful when deployment evidence is absent, and synchronize non-pass results through Task 280.

Depends On: 260, 279, 280. Task 280 is PASSED and its synchronization boundary was inspected.

Testing Coverage Exceptions: None.

Verification Criteria: Unit tests cover bounded sink configuration and query windows, request correlation, pagination, delayed ingestion, timeout, malformed and authorization failures, redaction, safe artifacts, and exit semantics. Acknowledged deployed execution must generate every required action and prove centralized events without accepting local console output. Missing deployment configuration must produce seven BLOCKED results plus a synchronized finding. Reports must link sanitized sink evidence, agree with 04_OPEN.md, and pass focused verifier, Playwright, consistency, traceability, and security checks.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Dependencies are PASSED or PREPARED.
- [x] Preparation evidence and Task 280 review evidence were read.
- [x] Task-owned scope was reconstructed from baseline, status, paths, diffs, and symbol discovery.
- [x] code-review-skill was invoked exactly once; Python, TypeScript, security, concurrency, and common-bug guidance was read.
- [x] Review used current source and fresh executions.
- [x] No production code or task-list status was changed by this reviewer.
- [x] The configured instruction to merge an unspecified [PHASE-ID] was not executed because the worktree is a dirty prepared worktree and no concrete merge target was supplied; merging would mutate unrelated task work.

~~~
pre_review_gates_passed: true
blocking_issue: "Three blocking and two important findings remain."
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD is f9a646ffede5c8a63ad787a07eaba7efc0433677 on multistep-phase-08. New Task 285 files and unique shared-file additions were separated from earlier Phase 08 work. The preparation hashes match current task files except the task-list file, whose status changed from OPEN to PREPARED before this review. The current row was read and not edited.

Commands used:

    git status --short --untracked-files=all
    git rev-parse HEAD
    git diff --numstat f9a646ffede5c8a63ad787a07eaba7efc0433677
    git diff -- scripts/check.py docs/implementation/04_OPEN.md docs/testing/phase08/finding-history.json
    rg -n '285|P08-FIND-285|ROOT-T285|SW-REQ-084'
    Python AST symbol discovery for the three Python task files
    sha256sum of reviewed source and evidence files

Pre-existing dirty-worktree changes and exclusions: The worktree contains earlier Phase 08 backend, frontend, operator, E2E, Task 280, and Tasks 281–284 changes. They were not attributed to Task 285 except the Task 285 gate registration, Task 285 ledger/history additions, and the files listed below. Task 280 implementation files were inspected as dependency code.

| Changed file | Change source | Confidence | Symbols or units |
|---|---|---|---|
| scripts/task285_log_sink.py | new sink adapter | HIGH | 24 |
| scripts/run-task285-acceptance.py | new runner | HIGH | 14 |
| scripts/test_run_task285_acceptance.py | new focused tests | HIGH | 24 |
| frontend/playwright.task285.config.ts | new deployed config | HIGH | 1 |
| frontend/tests/task285-centralized-logging.spec.ts | new action producer | HIGH | 16 |
| scripts/check.py | Task 285 gate registration | MEDIUM | 1 |
| docs/operations/task285-centralized-logging-acceptance.md | operator contract | HIGH | 1 |
| docs/implementation/04_OPEN.md | three finding records | MEDIUM | 1 |
| docs/testing/phase08/finding-history.json | append-only IDs | MEDIUM | 1 |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Focused tests cover sink bounds, correlation, pagination, delayed ingestion, timeout, malformed/auth failures, redaction, artifacts, and exit semantics without secrets. | Focused tests plus adversarial source inspection. | FAIL | 15 tests pass, but mixed top-level metadata redaction, cumulative pagination timeout, and local/private target variants are not tested. |
| 2 | Deployed actions generate every required centralized event with timestamps, safe IDs, fixed tuples, and distinguishable outcomes; local console output is not evidence. | Deployed config/spec and independent GCP query. | FAIL | Local Playwright has 8 intentional skips and the direct deployed config fails without acknowledgement. Public actions-file input and non-canonical receipt tuples can bypass action provenance; no deployed evidence exists. |
| 3 | Missing deployment configuration returns seven BLOCKED results and a synchronized owner/evidence/retest finding. | Fresh runner, report, sink artifact, Task 280, and 04_OPEN.md. | PASS | Fresh run task285-86b62bc64372b36ce57adf1a exited 2 with 7 BLOCKED rows, P08-FIND-285-001, ROOT-T285-DEPLOYED-ENVIRONMENT, and zero observations. |
| 4 | Report evidence is sanitized, findings agree with 04_OPEN.md, and focused verifier, Playwright, consistency, traceability, and security checks pass. | Fresh report, validators, focused tests, Playwright, and full-gate artifact. | FAIL | Task 280 consistency and traceability pass, but the aggregate HTML is not deployed evidence and the verifier has the blocking security and evidence-integrity defects below. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Change | Caller or consumer | Test or evidence |
|---:|---|---|---|---|---|---|
| 1 | SinkError | exception | scripts/task285_log_sink.py:40 | added | sink taxonomy | focused tests |
| 2 | SinkAuthorizationBlocked | exception | scripts/task285_log_sink.py:44 | added | auth boundary | auth test |
| 3 | SinkUnavailableBlocked | exception | scripts/task285_log_sink.py:48 | added | runner | timeout/block tests |
| 4 | SinkMalformed | exception | scripts/task285_log_sink.py:52 | added | sanitizer/runner | malformed tests |
| 5 | RedactionViolation | exception | scripts/task285_log_sink.py:56 | added | sanitizer/runner | redaction tests |
| 6 | SinkConfig | dataclass | scripts/task285_log_sink.py:60 | added | CloudLoggingClient | config test |
| 7 | SinkConfig.from_environment | classmethod | scripts/task285_log_sink.py:77 | added | verify | config test |
| 8 | QueryWindow | dataclass | scripts/task285_log_sink.py:98 | added | query/retention | window test |
| 9 | QueryWindow.__post_init__ | method | scripts/task285_log_sink.py:104 | added | QueryWindow | window test |
| 10 | ExpectedEvent | dataclass | scripts/task285_log_sink.py:113 | added | receipt contract | verifier tests |
| 11 | SafeEvent | dataclass | scripts/task285_log_sink.py:124 | added | report projection | artifact test |
| 12 | format_timestamp | function | scripts/task285_log_sink.py:135 | added | filters/evidence | indirect tests |
| 13 | parse_timestamp | function | scripts/task285_log_sink.py:140 | added | receipt/sink | malformed tests |
| 14 | load_expected_events | function | scripts/task285_log_sink.py:153 | added | verify | receipt fixtures |
| 15 | CloudLoggingClient | class | scripts/task285_log_sink.py:180 | added | verify | adapter tests |
| 16 | CloudLoggingClient.__init__ | method | scripts/task285_log_sink.py:185 | added | production/tests | adapter tests |
| 17 | CloudLoggingClient._load_token | method | scripts/task285_log_sink.py:196 | added | production ADC | source audit |
| 18 | CloudLoggingClient._open | method | scripts/task285_log_sink.py:214 | added | default HTTP | source audit |
| 19 | CloudLoggingClient.query | method | scripts/task285_log_sink.py:227 | added | poll_for_events | pagination tests |
| 20 | extract_payload | function | scripts/task285_log_sink.py:289 | added | sanitizer | console test |
| 21 | scan_forbidden | function | scripts/task285_log_sink.py:297 | added | sanitizer | redaction test |
| 22 | sanitize_entries | function | scripts/task285_log_sink.py:326 | added | polling/report | mixed-entry finding |
| 23 | poll_for_events | function | scripts/task285_log_sink.py:368 | added | verify | delayed/timeout tests |
| 24 | verify_event_contract | function | scripts/task285_log_sink.py:394 | added | evaluate | exact-event test |
| 25 | load_module | function | scripts/run-task285-acceptance.py:51 | added | runner import | import |
| 26 | validate_deployed_url | function | scripts/run-task285-acceptance.py:65 | added | verify guard | target test |
| 27 | blocked_results | function | scripts/run-task285-acceptance.py:83 | added | blocked paths | seven-row test |
| 28 | write_json | function | scripts/run-task285-acceptance.py:98 | added | artifacts/results | artifact test |
| 29 | write_blocked_evidence | function | scripts/run-task285-acceptance.py:105 | added | blocked path | blocked test |
| 30 | run_playwright | function | scripts/run-task285-acceptance.py:119 | added | verify | source audit |
| 31 | query_window | function | scripts/run-task285-acceptance.py:150 | added | verify | complete test |
| 32 | safe_probes | function | scripts/run-task285-acceptance.py:157 | added | sanitizer | artifact test |
| 33 | retention_probe | function | scripts/run-task285-acceptance.py:170 | added | verify | retention test |
| 34 | evaluate | function | scripts/run-task285-acceptance.py:197 | added | verify | pass mapping |
| 35 | verify | function | scripts/run-task285-acceptance.py:236 | added | main/tests | complete test |
| 36 | finalize_report | function | scripts/run-task285-acceptance.py:301 | added | main | fresh report |
| 37 | result_exit_code | function | scripts/run-task285-acceptance.py:324 | added | deferred path | exit test |
| 38 | main | function | scripts/run-task285-acceptance.py:334 | added | CLI | fresh run |
| 39 | deployed config guard | module logic | frontend/playwright.task285.config.ts:4-36 | added | Playwright | direct guard |
| 40 | ActionEvent | interface | frontend/tests/task285-centralized-logging.spec.ts:18 | added | receipt | typecheck |
| 41 | Envelope | interface | frontend/tests/task285-centralized-logging.spec.ts:26 | added | API parser | typecheck |
| 42 | receipt state and constants | configuration | frontend/tests/task285-centralized-logging.spec.ts:12-16 | added | all scenarios | typecheck |
| 43 | fixture | function | frontend/tests/task285-centralized-logging.spec.ts:32 | added | setup/actions | scenarios |
| 44 | envelope | function | frontend/tests/task285-centralized-logging.spec.ts:38 | added | API actions | scenarios |
| 45 | record | function | frontend/tests/task285-centralized-logging.spec.ts:44 | added | receipt | scenarios |
| 46 | csrf | function | frontend/tests/task285-centralized-logging.spec.ts:51 | added | mutations | scenarios |
| 47 | login | function | frontend/tests/task285-centralized-logging.spec.ts:59 | added | auth/admin | auth scenario |
| 48 | manualItem | function | frontend/tests/task285-centralized-logging.spec.ts:71 | added | manual actions | manual scenario |
| 49 | test.beforeAll | hook | frontend/tests/task285-centralized-logging.spec.ts:84 | added | four tests | fixture checks |
| 50 | test.afterAll | hook | frontend/tests/task285-centralized-logging.spec.ts:94 | added | receipt publication | action file |
| 51 | authentication/UI scenario | Playwright test | frontend/tests/task285-centralized-logging.spec.ts:108 | added | auth/admin | local skip/deployed |
| 52 | manual mutation scenario | Playwright test | frontend/tests/task285-centralized-logging.spec.ts:126 | added | item routes | cleanup |
| 53 | classification/user scenario | Playwright test | frontend/tests/task285-centralized-logging.spec.ts:178 | added | classification/users | cleanup |
| 54 | external provider scenario | Playwright test | frontend/tests/task285-centralized-logging.spec.ts:209 | added | provider routes | cleanup |
| 55 | timestamp | test helper | scripts/test_run_task285_acceptance.py:19 | added | fixtures | all tests |
| 56 | action_receipt | test helper | scripts/test_run_task285_acceptance.py:23 | added | expected events | sink tests |
| 57 | cloud_entry | test helper | scripts/test_run_task285_acceptance.py:57 | added | fake sink | sink tests |
| 58 | FakeClient | test class | scripts/test_run_task285_acceptance.py:70 | added | polling | polling tests |
| 59 | FakeClient.__init__ | method | scripts/test_run_task285_acceptance.py:71 | added | FakeClient | polling tests |
| 60 | FakeClient.query | method | scripts/test_run_task285_acceptance.py:81 | added | polling | polling tests |
| 61 | Task285SinkTests | test class | scripts/test_run_task285_acceptance.py:89 | added | unittest | 15 methods |
| 62 | configuration test | test | scripts/test_run_task285_acceptance.py:90 | added | SinkConfig | bounds |
| 63 | target guard test | test | scripts/test_run_task285_acceptance.py:114 | added | URL guard | target cases |
| 64 | query window test | test | scripts/test_run_task285_acceptance.py:132 | added | QueryWindow | boundaries |
| 65 | pagination/token test | test | scripts/test_run_task285_acceptance.py:139 | added | query | pages/token |
| 66 | auth/malformed test | test | scripts/test_run_task285_acceptance.py:165 | added | query | 403/shape |
| 67 | delayed-ingestion test | test | scripts/test_run_task285_acceptance.py:191 | added | polling | retry |
| 68 | timeout test | test | scripts/test_run_task285_acceptance.py:205 | added | polling | timeout |
| 69 | redaction test | test | scripts/test_run_task285_acceptance.py:219 | added | sanitizer | forbidden values |
| 70 | sanitizer test | test | scripts/test_run_task285_acceptance.py:232 | added | sanitizer | console/fields |
| 71 | event contract test | test | scripts/test_run_task285_acceptance.py:247 | added | event contract | exact events |
| 72 | retention test | test | scripts/test_run_task285_acceptance.py:264 | added | retention | age |
| 73 | blocked-artifact test | test | scripts/test_run_task285_acceptance.py:277 | added | blocked output | seven rows |
| 74 | exit-semantics test | test | scripts/test_run_task285_acceptance.py:286 | added | exit code | precedence |
| 75 | complete-pass mapping test | test | scripts/test_run_task285_acceptance.py:292 | added | evaluate | seven pass |
| 76 | sanitized-projection test | test | scripts/test_run_task285_acceptance.py:310 | added | verify | private values |
| 77 | opener | nested helper | scripts/test_run_task285_acceptance.py:146 | added | pagination test | body |
| 78 | denied | nested helper | scripts/test_run_task285_acceptance.py:166 | added | auth test | 403 |
| 79 | validate_phase08_acceptance_contracts | gate function | scripts/check.py:672 | added | static/full gate | focused suite/validator |
| 80 | Task 285 operator contract | documentation | docs/operations/task285-centralized-logging-acceptance.md:1-67 | added | operator | preconditions |
| 81 | Task 285 finding ledger records | data contract | docs/implementation/04_OPEN.md:779-824 | added | Task 280 | three roots |
| 82 | Task 285 finding history IDs | data contract | docs/testing/phase08/finding-history.json:19-21 | added | Task 280 | append-only IDs |

~~~
inventory_source_count: 82
audited_symbol_count: 82
inventory_complete: true
generated_groupings:
  - "None; all discovered production, test, configuration, gate, and Task 285 data-contract units are listed. Task 280 dependency files are excluded from Task 285 ownership."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundary | Performance/idioms | Tests and gaps | Result |
|---|---|---|---|---|---|---|---|
| SinkError | Closed sink error base. | Subclasses only. | No resources. | Safe taxonomy. | Minimal. | Used by focused tests. | PASS |
| SinkAuthorizationBlocked | Authority failure maps to BLOCKED. | Safe fixed message. | No state. | Least privilege. | Minimal. | Auth test. | PASS |
| SinkUnavailableBlocked | Incomplete deployment/ingestion maps to BLOCKED. | Safe fixed message. | No state. | No diagnostics. | Minimal. | Timeout test. | PASS |
| SinkMalformed | Unsafe shape is rejected. | Normal malformed paths caught. | No state. | Evidence boundary. | Minimal. | Malformed test. | PASS |
| RedactionViolation | Sensitive evidence is rejected. | Mixed entry metadata is not inspected by caller. | No state. | Incomplete raw-entry boundary. | Minimal. | Redaction gap. | FAIL |
| SinkConfig | Stores one scoped resource and numeric bounds. | Direct construction bypasses env checks intentionally. | Immutable. | Project scope. | Bounded. | Config test. | PASS |
| SinkConfig.from_environment | Validates project resource and bounds. | Invalid environment blocks. | No state. | No secret output. | Bounded. | Config test. | PASS |
| QueryWindow | Positive aware interval at most 15 minutes. | Naive/zero/large rejected. | Immutable. | Filter boundary. | Minimal. | Window test. | PASS |
| QueryWindow.__post_init__ | Enforces interval invariant. | Odd tzinfo can raise raw type errors. | No resources. | No output. | Minimal. | Boundary test. | PASS |
| ExpectedEvent | Carries correlation tuple. | Tuple fields only regex-bounded, not canonical. | Immutable. | ID validated. | Small. | No canonical tuple test. | FAIL |
| SafeEvent | Fixed sanitized projection. | Constructed after current sanitizer. | Immutable. | Raw fields dropped. | Small. | Artifact test. | PASS |
| format_timestamp | UTC RFC3339 filter/evidence output. | Aware input expected. | No state. | No secrets. | Minimal. | Indirect tests. | PASS |
| parse_timestamp | Rejects malformed or naive timestamps. | SinkMalformed is intentional. | No resources. | Safe errors. | Minimal. | Malformed tests. | PASS |
| load_expected_events | Validates receipt shape, IDs, uniqueness, string bounds. | Does not enforce canonical tuples or start/end order. | Allocates event list; count only rejected later at query. | Safe receipt fields. | Small. | Complete fixture only. | FAIL |
| CloudLoggingClient | Owns independent GCP entries:list adapter. | Classifies token/HTTP failures. | Sequential local state. | Fixed endpoint/resource. | Page caps. | Adapter tests. | PASS |
| CloudLoggingClient.__init__ | Selects production or injected seams. | None defaults are correct. | No ownership. | Test seams. | Minimal. | Indirect. | PASS |
| CloudLoggingClient._load_token | Runs fixed ADC command with 15 second bound. | OSError/SubprocessError blocks. | Subprocess bounded. | Token never emitted. | Captured output. | No direct command test. | PASS |
| CloudLoggingClient._open | Bounded response and safe HTTP classification. | Unexpected read exceptions can escape. | Context manager closes response. | No body diagnostics. | 4 MiB cap. | No incomplete-read test. | PASS |
| CloudLoggingClient.query | Builds ID/time/resource scoped paginated request. | Per-page timeout is not an overall deadline. | 20 pages can outlive configured timeout. | Token header only. | 20 page and 5000 entry caps. | Pagination lacks slow-page timeout. | FAIL |
| extract_payload | Requires structured jsonPayload. | Mixed metadata is ignored. | No state. | Incomplete raw-entry scan. | Minimal. | Console-only test. | FAIL |
| scan_forbidden | Recursively scans keys, probes, UUIDs, known patterns. | Deep recursion and pattern gaps escape. | No shared state. | Payload-only due caller. | Recursion unbounded. | Missing deep/mixed cases. | FAIL |
| sanitize_entries | Validates fixed payload and projects safe event. | Sensitive textPayload/httpRequest passes with safe JSON payload. | No state. | Blocking redaction bypass. | Sorted projection. | Reproduction passes unsafe entry. | FAIL |
| poll_for_events | Retries until IDs observed or deadline. | Query and sleep can exceed deadline. | Deadline checked between full queries. | No local fallback. | No independent poll cap. | Fake timeout only. | FAIL |
| verify_event_contract | Requires exactly one expected tuple and admin outcome set. | Expected tuple comes from receipt. | Local maps only. | Provenance not independent. | Linear. | No tuple mutation. | FAIL |
| load_module | Loads colocated sink module. | Loader failure is raw RuntimeError. | sys.modules mutation. | Fixed path. | Minimal. | No failure test. | PASS |
| validate_deployed_url | Intended credential-free acknowledged HTTPS target. | Accepts loopback/private IP variants. | No state. | Local evidence bypass. | Minimal. | Missing adversarial IP tests. | FAIL |
| blocked_results | Emits all seven blocked rows and safe root. | Fixed empty IDs and evidence. | No state. | No private diagnostics. | Tiny. | Seven-row test. | PASS |
| write_json | Writes deterministic private JSON. | Filesystem error escapes main; not atomic. | Creates directory and file. | 0600 after write. | Bounded callers. | Artifact test only. | PASS |
| write_blocked_evidence | Emits fixed code and empty observations. | Fixed schema. | Delegates write_json. | Safe. | Tiny. | Blocked test. | PASS |
| run_playwright | Runs dedicated no-artifact deployed spec. | TimeoutExpired/OSError escape main. | No cancellation cleanup hook. | Child output suppressed. | 300 second process cap. | No subprocess failure test. | FAIL |
| query_window | Expands action interval under cap. | Reversed timestamps raise raw ValueError. | No state. | Safe timestamps. | Minimal. | No reversed receipt test. | FAIL |
| safe_probes | Holds fixture values for in-memory scan. | Omits strings shorter than four chars. | No persistence. | Known values only. | Small. | No short-secret test. | PASS |
| retention_probe | Requires safe UUID and 90-day-old timestamp. | Missing/fresh probe blocks. | No state. | Safe correlation. | Two-minute query. | Retention test. | PASS |
| evaluate | Maps seven criteria to observed events. | Accept-03 unconditional; tuple semantics receipt-controlled. | Local lists/sets. | Independent semantics absent. | Linear. | Self-authored complete fixture. | FAIL |
| verify | Orchestrates target, actions, GCP, retention, projection. | Public actions-file skips Playwright; malformed values can raise raw errors. | Shared client lacks end-to-end deadline. | GCP independent but action provenance not. | Bounds incomplete. | Fake-client test masks bypass. | FAIL |
| finalize_report | Hands results to Task 280. | Spawn failure escapes. | No subprocess timeout. | Task 280 sanitizes report. | Local subprocess. | Fresh report. | PASS |
| result_exit_code | Fail before blocked before pass. | Unknown statuses return zero. | No state. | No secrets. | Minimal. | Precedence test. | PASS |
| main | Publishes safe failure or final report. | TimeoutExpired, OSError, ValueError, RecursionError and report failures can traceback. | Artifact dir created before handling. | Normal child streams suppressed. | No outer sink/report deadline. | Missing-config only. | FAIL |
| deployed config guard | Rejects missing ack and browser artifacts. | Same local/private IP gap as Python. | One worker, no retry, no trace. | Incomplete non-local guard. | Minimal. | Direct missing-ack test. | FAIL |
| ActionEvent | Five receipt fields. | All strings, no literal vocabularies. | Local object. | Safe shape only. | Small. | Typecheck only. | FAIL |
| Envelope | Request ID plus data/error response shape. | Runtime cast after JSON parse. | No state. | Body not retained. | Small. | Scenario assertions. | PASS |
| receipt state and constants | Controls skip and collected events. | Ack-only enable flag. | Global list, one worker. | Marker stays in memory. | Small. | Local skips. | PASS |
| fixture | Requires each private deployed input. | Missing values throw. | In memory. | No output. | Minimal. | beforeAll. | PASS |
| envelope | Parses production JSON and checks UUID. | Invalid body aborts receipt. | Response in memory. | No receipt body. | Small. | All scenarios. | PASS |
| record | Enforces unique category/request ID and stores fields. | Caller can choose arbitrary tuple values. | Mutates global event list. | Provenance gap. | O(n) for 11. | No wrong-tuple test. | FAIL |
| csrf | Gets production CSRF token. | Non-200/body failure throws. | Token local. | Mutation protection. | One request. | Scenarios. | PASS |
| login | Uses generated login request. | Status asserted by caller. | Shared cookie context. | Credentials in request memory only. | One request. | Auth scenario. | PASS |
| manualItem | Builds canonical solid request. | Extras can override fields but callers fixed. | Pure allocation. | Fixture name not receipt. | Tiny. | Manual scenario. | PASS |
| test.beforeAll | Requires all fixtures. | Missing fixture aborts. | No kill cleanup. | Values not printed. | Minimal. | Setup. | PASS |
| test.afterAll | Requires 11 events and writes receipt. | Partial/test-kill behavior lacks atomic cleanup. | Global list and direct write. | Receipt projection safe. | Tiny. | No cancellation test. | PASS |
| authentication/UI scenario | Failed/success auth, admin route, focus, theme, axe. | Local config skips. | Shared session. | Credentials not recorded. | Bounded. | No deployed evidence. | PASS |
| manual mutation scenario | Success, validation, audit failure, item cleanup. | Cleanup failure fails. | finally cleanup for item. | IDs omitted from receipt. | Bounded. | Deployment unavailable. | PASS |
| classification/user scenario | Classification mutation and restricted lookup. | Cleanup failure fails. | finally cleanup. | Lookup omitted from receipt. | Bounded. | Deployment unavailable. | PASS |
| external provider scenario | Search/import/dependency failure and cleanup. | Dependency action depends on import success. | finally cleanup. | Provider payload omitted. | Bounded. | Deployment unavailable. | PASS |
| timestamp | Test UTC formatter. | Test-only. | No state. | No secrets. | Minimal. | Fixtures. | PASS |
| action_receipt | Synthetic 11-event receipt. | Synthetic tuples demonstrate production trust gap. | Local data. | Safe IDs. | 11 entries. | No noncanonical case. | PASS |
| cloud_entry | Synthetic structured sink entry. | Omits sensitive top-level metadata. | Local data. | Test does not model bypass. | Tiny. | Mixed-entry gap. | FAIL |
| FakeClient | Deterministic page/query seam. | Cannot model blocking or real page timeout. | Mutable call count, test-local. | No network. | Tiny. | Poll tests. | PASS |
| FakeClient.__init__ | Configures fake bounds. | Direct construction bypasses env validation. | Local state. | No secrets. | Minimal. | Poll tests. | PASS |
| FakeClient.query | Returns pages or injected errors. | No slow multi-page behavior. | Mutable counter. | No network. | Tiny. | Timeout gap. | FAIL |
| Task285SinkTests | Focused 15-test suite. | Passes current assertions. | Test-local. | Fixtures safe. | Small. | Three adversarial classes absent. | FAIL |
| configuration test | Least-scope and numeric bounds. | No NaN/duplicate/IP cases. | No state. | Partial least privilege. | Tiny. | Current assertions. | PASS |
| target guard test | HTTPS/credential/path/ack cases. | Omits 127/8/private/mapped IPv6. | No state. | Security gap. | Tiny. | Reproduction fails guard. | FAIL |
| query window test | Positive 15-minute boundaries. | Not elapsed-operation timeout. | No state. | No data. | Tiny. | Boundaries. | PASS |
| pagination/token test | Two-page token/body behavior. | Immediate pages only. | Local list. | Token absent from body. | Two pages. | No page-time bound. | PASS |
| auth/malformed test | 403 and malformed array. | Does not cover 401/read failures. | No state. | Safe class. | Tiny. | Basic coverage. | PASS |
| delayed-ingestion test | Retry after empty page. | Fake query instant. | Injected clock/sleeper. | No local fallback. | Two polls. | Good basic case. | PASS |
| timeout test | Missing event blocks by fake monotonic clock. | Cannot catch slow query/page. | Injected clock. | No secrets. | Tiny. | Cumulative gap. | FAIL |
| redaction test | Forbidden key/value/UUID/probe. | No top-level metadata/deep/short cases. | No state. | Partial. | Tiny. | Mixed-entry gap. | FAIL |
| sanitizer test | Console-only, extra field, timestamp checks. | Safe JSON plus sensitive metadata passes. | No state. | Blocking gap. | Tiny. | Reproduction. | FAIL |
| event contract test | Missing/duplicate/outcome check. | Self-authored expected tuple. | Local list. | Provenance gap. | Linear. | No canonical mutation. | FAIL |
| retention test | 90-day safe UUID and window. | No real GCP. | No resources. | Safe. | Tiny. | Retention. | PASS |
| blocked-artifact test | Seven blocked rows and no credential word. | Does not finalize Task 280. | Temp directory. | Safe artifact. | Tiny. | Fresh report covers sync. | PASS |
| exit-semantics test | PASS/FAIL/BLOCKED precedence. | Unknown status falls through. | No state. | Safe. | Tiny. | Precedence. | PASS |
| complete-pass mapping test | Seven PASS for fixture events. | Cannot catch noncanonical tuple. | Local objects. | Gap. | Linear. | Self-authored. | FAIL |
| sanitized-projection test | Private fixture values absent from artifact. | Fake entries omit top-level metadata. | Temp directory/fake client. | Partial. | 12 events. | Mixed gap. | FAIL |
| opener | Fake paginated opener. | Immediate fixed pages. | Local list. | Body assertion. | Two calls. | Pagination only. | PASS |
| denied | Fake 403 opener. | One status class. | No state. | No response body. | Tiny. | Auth basic. | PASS |
| validate_phase08_acceptance_contracts | Runs Task 285 suite and Task 280 validator. | Does not run deployed acceptance. | Static lane caller. | No secrets. | One suite. | Full gate registration. | PASS |
| operator contract | Documents deployment, GCP scope, retention, no-console rule. | Documentation cannot enforce code gaps. | External state. | Intended boundary clear. | N/A. | Inspected. | PASS |
| finding ledger records | Three roots with owner, evidence, retest. | Current run links only deployment root correctly. | Append-only doc. | Safe IDs/no PII. | N/A. | Task 280 validator. | PASS |
| finding history IDs | Retains 285-001 through 003 sorted. | Current validator accepts. | Append-only data. | Safe IDs. | Tiny. | Traceability. | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence or trigger | Required repair or disposition |
|---|---|---|---|---|---|
| [blocking] | scripts/task285_log_sink.py:335-365 | sanitize_entries | Only jsonPayload is scanned. Sensitive textPayload, httpRequest, or other top-level metadata can pass beside a safe payload. | A safe four-field payload plus textPayload leak@example.test and httpRequest.requestUrl returned SafeEvent instead of RedactionViolation. | Scan the complete raw entry or reject non-allowlisted top-level fields. Add mixed-payload regression tests. |
| [blocking] | scripts/run-task285-acceptance.py:65-80 and frontend/playwright.task285.config.ts:9-19 | deployed target guards | Loopback and private targets such as 127.0.0.2, 127.1.2.3, 10.0.0.2, and mapped IPv6 loopback are accepted as deployed. | Direct Python guard probe accepted all four targets. The operator contract requires non-loopback and the task forbids local evidence substitution. | Use IP classification in both guards and reject loopback, private, link-local, reserved, unspecified, multicast, and mapped loopback addresses. Add shared adversarial tests. |
| [blocking] | scripts/run-task285-acceptance.py:197-265 and :337-346 | receipt loading, evaluate, verify, actions-file | The verifier trusts action/resource/outcome from the receipt and exposes actions-file as a production CLI bypass. A synthetic receipt with external_search.action=arbitrary still returns seven PASS rows when matching sink events are supplied. | Reproduction mutated one receipt action and evaluate returned seven PASS statuses. verify uses the supplied file instead of running Playwright. | Enforce a canonical category-to-tuple map. Remove the public bypass or make it unmistakable test-only below the production CLI, and require receipt provenance from the same run. |
| [important] | scripts/task285_log_sink.py:227-287 and :368-391 | query and poll_for_events | timeout_seconds is only per HTTP request and between polls, not an end-to-end paginated deadline. Twenty pages can each consume up to 30 seconds and sleep can overshoot the deadline. | Source inspection; tests use immediate fake pages and cannot catch cumulative page latency. | Carry one monotonic deadline through pages, use remaining time for each request and sleep, and add a slow multi-page timeout test. |
| [important] | scripts/run-task285-acceptance.py:119-154 and :334-379 | run_playwright, query_window, main | TimeoutExpired, OSError, reversed timestamps, recursion errors, and report-spawn failures can escape as tracebacks without a synchronized result. | main catches only SinkError subclasses; run_playwright uses subprocess timeout without local normalization and query_window can raise ValueError. | Normalize all expected subprocess/input/report failures to fixed safe statuses and always finalize synchronized evidence. Add timeout, spawn, malformed, and reversed-receipt tests. |

~~~
blocking_findings: 3
important_findings: 2
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log or artifact |
|---|---|---:|---|---|
| python3 -m unittest -v scripts/test_run_task285_acceptance.py | repository root | 0 | 15 passed | focused sink/runner suite |
| python3 -m unittest -v scripts/test_run_task285_acceptance.py scripts/test_phase08_acceptance.py | repository root | 0 | 40 passed | Task 285 plus Task 280 consistency |
| python3 scripts/phase08_acceptance.py validate | repository root | 0 | 12 scenarios and 91 criteria valid | Task 280 validator |
| python3 scripts/validate-task-list.py | repository root | 0 | 286 sequential tasks valid | status not edited |
| python3 scripts/validate-traceability.py | repository root | 0 | passed | traceability |
| git diff --check | repository root | 0 | passed | whitespace |
| bun run typecheck | frontend | 0 | passed | TypeScript |
| bun run test:e2e -- tests/task285-centralized-logging.spec.ts | frontend | 0 | 8 intentional local-config skips | not deployed evidence |
| bunx playwright test -c playwright.task285.config.ts tests/task285-centralized-logging.spec.ts without acknowledgement | frontend | 1 | expected fail-closed guard | missing deployment ack |
| python3 scripts/run-task285-acceptance.py with no deployment configuration | repository root | 2 | fresh seven-row BLOCKED report | task285-86b62bc64372b36ce57adf1a |
| python3 scripts/check.py --quick | repository root | 0 | static and changed-area lanes completed; focused output showed 30 passed and 74 skipped | current quick gate |
| recorded full gate represented by logs/task285-full-check.html | repository root | 0 in recorded run | HTML marked quality gate passed; 91/91 manifest requirements, 87.4 percent Go internal coverage, 96.06 percent frontend line coverage, race and vulnerability lanes present | aggregate evidence only |

The ordinary Playwright run intentionally used the ordinary config and therefore only proves local skip behavior. The dedicated config rejected missing acknowledgement. No local log, browser trace, API response, Vite output, or aggregate HTML was used as centralized-log evidence.

## 9. Files Inspected and Staleness Fingerprints

Hashes were computed from current contents after review. The preparation hash for the task-list file was stale because its status changed to PREPARED before review; the current hash is recorded and no status was changed here.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| scripts/task285_log_sink.py | sink query, redaction, projection | blocking/important | SHA-256 | 9e06067b67fd5584dd8078ba13bf7896e7da54a829db4f105b0a7cb338d53d83 |
| scripts/run-task285-acceptance.py | runner and report handoff | blocking/important | SHA-256 | 3e68e309d4a21a44a69c13def2fab2cfba959a67277fc47b59ed47d500236ba6 |
| scripts/test_run_task285_acceptance.py | focused tests | missing adversarial cases | SHA-256 | 855222b8a358ec3a43c894ae912285b2cfb6c38979e088144de1ba7860fd23dd |
| frontend/playwright.task285.config.ts | deployed target/artifact guard | blocking | SHA-256 | f3f4ebd2548a303494b0e30f49fc91db4fbdc06f91dae3237d430564dc4df570 |
| frontend/tests/task285-centralized-logging.spec.ts | deployed action producer | provenance gap | SHA-256 | 80ec228a93190b6ee1e12e043e4384b71545a080226f23fc350a103178c80e7c |
| scripts/check.py | aggregate registration | none | SHA-256 | 4343808d63552d481671815303456185ef76ef0aa0752cb5ae84756fa480d281 |
| scripts/phase08_acceptance.py | Task 280 caller inspected | dependency boundary | SHA-256 | b36ea8188e3e6d7e05ae17dc992c84cf493c9ac78a8a85706d51e7e40ca34bdb |
| docs/operations/task285-centralized-logging-acceptance.md | operator preconditions | intended guard | SHA-256 | 0d92e384aa7f1831c995391e61aeb1a9d792c079b566a50266bfb67681d3cbba |
| docs/implementation/04_OPEN.md | Task 285 findings | synchronized roots | SHA-256 | de951074a8dc70f49c01eedc99af2cf56c0d7b5894733c838190fa93defbdf15 |
| docs/testing/phase08/finding-history.json | retained finding IDs | history | SHA-256 | d18286a98a3e957cf7a2f4ece91c517337513c2b78a3c5dbd1a9983492d77e5b |
| docs/implementation/02_TASK_LIST.md | current PREPARED status | status untouched | SHA-256 | 64803ce4d4af01bcda803dc843623c2ab20fae73de75b4701153c6602946aa63 |
| docs/implementation/evidence/task-285-preparation.md | preparation report | prior task-list hash stale | SHA-256 | 6b58d7d25daaa6c858647edb2cc38ab80370c582fd8c53ffa0274ad2f7e386ae |
| logs/phase08-acceptance/task285-86b62bc64372b36ce57adf1a/report.json | fresh truthful deployment result | seven BLOCKED | SHA-256 | 99d90c1019f5ada390d1505c8412a6520965b56d97a3c7956a0f70c47eaf6f9f |
| logs/task285-deployed/86b62bc64372b36ce57adf1a/sink-evidence.json | fresh blocker artifact | zero observations | SHA-256 | 124b899b1a6ad0405b38ad91b835db9b665b83bc2e4131f2d83fb5001c0a5585 |
| logs/task285-full-check.html | recorded aggregate gate | not deployed evidence | SHA-256 | a21dc27cead6b6fafb60bc9d790fbe8187dadde89f22ea039a3f9d7465ffb297 |

~~~
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "task-285-preparation.md: task-list hash only; current task hashes otherwise match"
~~~

## 10. Coverage and Exceptions

- [x] Focused verifier coverage ran.
- [x] Full-gate evidence was inspected, including race, vulnerability, coverage, traceability, and Playwright lanes.
- [x] Untested target, mixed-metadata, canonicalization, pagination-timeout, and CLI-error branches were manually inspected and recorded.
- [x] No task-row coverage exception exists.

~~~
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "logs/task285-full-check.html and focused unittest output"
observed_line_coverage: "aggregate 96.06 percent frontend and 87.4 percent Go internal; Task 285 Python is not included"
coverage_passed: false
~~~

Coverage finding: The aggregate gate passes its project coverage contract, but it does not provide Task 285 Python line coverage and focused tests omit the blocking adversarial cases. No exception is authorized by the task row.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] Task 280 synchronization and finding validation were independently exercised.
- [x] Local logs, browser traces, Playwright responses, and Vite output were not accepted as sink evidence.
- [x] Fresh no-configuration execution is truthfully BLOCKED with all seven criteria and one synchronized deployment finding.
- [x] Dedicated Playwright config fails closed without acknowledgement.
- [x] Duplicate-helper and source searches were performed.
- [x] Error, cleanup, timeout, concurrency, pagination, and malformed-input paths were challenged; missing paths are findings.

Findings: The fresh report proves only that deployment prerequisites are unavailable. The code does not yet make eventual PASS trustworthy under local/private target variants, mixed raw log metadata, synthetic action receipts, or long/failed query/subprocess paths.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains.

Validator command:

    python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-285-review.md

~~~
decision: "REJECTED"
reason: "Truthful seven-row BLOCKED result and Task 280 synchronization pass, but unsafe redaction, incomplete target guards, receipt/Playwright bypass, unbounded timeout, and error leakage prevent acceptance."
failed_criteria:
  - "Focused security and boundary coverage is incomplete."
  - "Deployed action evidence is not independently guaranteed."
  - "Full focused/security acceptance cannot pass while blocking findings remain."
failed_or_unaudited_symbols:
  - "sanitize_entries"
  - "validate_deployed_url"
  - "load_expected_events"
  - "verify"
  - "CloudLoggingClient.query"
  - "poll_for_events"
  - "main"
recommended_next_action: "Repair five findings, add adversarial tests, rerun focused and aggregate gates, rerun acknowledged deployed sink acceptance, refresh hashes, and obtain independent re-review. Do not change Task 285 status in this review."
~~~

## 13. Repair Context

### Failure Summary

Three blocking and two important findings make the acceptance boundary bypassable or unsafe: raw entry metadata is not redacted, local/private IP targets pass, receipt semantics and Playwright provenance are caller-controlled, sink deadline is not end-to-end bounded, and expected runner failures can escape as tracebacks without synchronized results.

### Minimal Repair Goal

Keep truthful seven-row BLOCKED behavior, but make every eventual PASS depend on canonical production action tuples and a non-local acknowledged target, scan the complete raw centralized entry before projection, enforce one monotonic deadline across pages/polls/sleeps, and normalize expected subprocess/input/report failures.

### Evidence to Reuse

Reuse docs/implementation/evidence/task-285-preparation.md, the fresh blocked report and sink artifact, Task 280 report/validator, focused tests, the reproductions in this review, and current hashes.

### Required Re-Review Surface

Re-review task285_log_sink.py query, scan_forbidden, sanitize_entries, poll_for_events; run-task285-acceptance.py target validation, receipt loading, verify, subprocess boundaries, and main; both deployed Playwright guards; adversarial tests; Task 280 linkage; the three finding records; fresh blocked and acknowledged deployment evidence; and every changed-file hash.

### Do Not Change

Do not accept local logs or browser/API responses as sink evidence, weaken seven-criterion BLOCKED behavior, delete historical findings, or edit Task 285 task-list status as part of repair or review.
