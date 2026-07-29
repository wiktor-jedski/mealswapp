# Review Evidence: Task 285 — Centralized Logging Acceptance Scenarios

task_id: 285
component: LogAggregator
static_aspect: Centralized Logging Acceptance Scenarios
input_status: PREPARED
review_decision: REJECTED
reviewed_at_utc: 2026-07-28T20:25:07Z
review_agent: Codex reviewer
evidence_file: docs/implementation/evidence/task-285-rereview.md
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677 plus current worktree
baseline_confidence: HIGH
code_review_skill_invoked: true
relevant_language_guide: Python and TypeScript security/concurrency guidance
repair_context_required: true

## 1. Task Source

Description: Phase 08.02 deployed-environment centralized logging acceptance verifier for every Phase 08 SW-REQ-084 step and acceptance criterion. It must generate isolated actions, correlate only safe request IDs, query the centralized sink independently, reject sensitive data, preserve bounded outcomes, and report missing deployment, sink authority, or retention as BLOCKED.

Depends On: 260, 279, 280; all are PASSED.

Testing Coverage Exceptions: None.

Verification Criteria: The task row requires bounded sink-adapter tests, correlation, pagination, delayed-ingestion, timeout, malformed/auth, redaction, artifact, and exit tests; environment-gated production action generation and independent sink evidence; seven-row synchronized BLOCKED evidence when deployment is unavailable; sanitized reports and consistency, traceability, and security checks.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PREPARED or PASSED.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy.
- [x] code-review-skill was invoked exactly once and its relevant guide read.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code changes.
- [x] git merge origin/multistep-phase-08 was checked and was already up to date.

pre_review_gates_passed: true
blocking_issue: Current implementation still has a sensitive-data acceptance bypass and a standalone Playwright DNS-publicness gap.

## 3. Review Baseline and Change Surface

Baseline/reference method: reviewed the current dirty worktree against f9a646ffede5c8a63ad787a07eaba7efc0433677, the Task 285 preparation report, the task row, configured reviewer prompt, ARCH-014, DESIGN-014, and the remote phase branch. The task-list modification was pre-existing and excluded from implementation review; its content hash was recorded to prove it was not edited.

Commands used to reconstruct the diff:
- git diff -- scripts/task285_log_sink.py scripts/run-task285-acceptance.py scripts/test_run_task285_acceptance.py frontend/playwright.task285.config.ts frontend/tests/task285-centralized-logging.spec.ts scripts/check.py
- git merge origin/multistep-phase-08
- rg -n '285' docs/implementation/02_TASK_LIST.md

Pre-existing dirty-worktree changes and exclusions: docs/implementation/02_TASK_LIST.md was already modified and remains PREPARED; no task-list status or content was changed by this review. The review evidence file is the only review artifact added.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| scripts/task285_log_sink.py | Task 285 repair | HIGH | configuration, receipt contract, Cloud Logging client, raw-entry sanitizer, polling, contract verifier |
| scripts/run-task285-acceptance.py | Task 285 repair | HIGH | target guard, action producer, evidence/report lifecycle, seven-row result handling |
| scripts/test_run_task285_acceptance.py | Task 285 repair tests | HIGH | fixtures, fake client, 21 focused tests and adversarial seams |
| frontend/playwright.task285.config.ts | Task 285 repair | HIGH | deployment target guard and Playwright configuration |
| frontend/tests/task285-centralized-logging.spec.ts | Task 285 acceptance producer | HIGH | action envelope, provenance, lifecycle, four browser scenarios |
| scripts/check.py | Phase 08 gate wiring in task-owned change surface | MEDIUM | Phase 08 acceptance gate and synchronized output helper |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Unit coverage proves bounded configuration/query windows, correlation, pagination, delayed ingestion, cumulative timeout, malformed/auth handling, redaction, sanitized artifacts, and exit semantics. | Focused tests, dependency tests, function audit, direct adversarial probes. | FAIL | 21 focused and 46 dependency tests pass, but the raw-entry sanitizer accepts four sensitive or diagnostic nested-field probes; the claimed full scan is not complete. |
| 2 | Deployed actions are generated independently from browser/API/local output, target scope is exact public owned deployment, and centralized sink evidence is independently queried. | Runner/config inspection, action receipt checks, target probes, deployment run. | FAIL | Public Python target guard, canonical receipt/provenance, internal action-file handling, and independent sink query pass inspection; the standalone Playwright config does not resolve the approved hostname or reject private DNS addresses. |
| 3 | Missing deployment, sink authority, or retention yields truthful seven-row synchronized BLOCKED evidence and exit 2. | Fresh clean-environment run and JSON inspection. | PASS | Fresh run exit 2 produced 7 BLOCKED rows, 0 PASS or FAIL, empty request IDs, per-row P08-FIND-285-001, root ROOT-T285-DEPLOYED-ENVIRONMENT, and sink observations empty. |
| 4 | Report, consistency, traceability, and security checks pass with no local console or browser response accepted as sink evidence. | Full gate, validators, raw-output and artifact audit. | FAIL | Full gate and repository validators pass; action provenance and output suppression pass; the sensitive nested metadata and diagnostic bypass violates the security boundary. |

## 5. Changed-Symbol Inventory

The inventory groups only small declarations or inseparable generated browser units. All production control-flow and security boundaries are separately listed.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | FORBIDDEN_KEY_RE and CANONICAL_EVENT_CONTRACTS | policy constants | scripts/task285_log_sink.py:26 | modified | sanitizer and receipt validator | redaction and contract tests |
| 2 | SinkError hierarchy and SinkConfig.from_environment | types and function | scripts/task285_log_sink.py:55 | modified | client and runner | configuration, malformed, auth tests |
| 3 | QueryWindow, ExpectedEvent, SafeEvent | behavioral types | scripts/task285_log_sink.py:119 | modified | query, receipt, sanitizer | window and fixture tests |
| 4 | format_timestamp and parse_timestamp | helpers | scripts/task285_log_sink.py:156 | modified | receipt and projection | timestamp tests |
| 5 | validate_expected_events, load_expected_events, secrets_compare | functions | scripts/task285_log_sink.py:174 | modified | runner verification | canonical, provenance, order tests |
| 6 | CloudLoggingClient setup, _load_token, _open | methods | scripts/task285_log_sink.py:238 | modified | query | adapter and auth tests |
| 7 | CloudLoggingClient.query | method | scripts/task285_log_sink.py:280 | modified | polling | pagination and cumulative-timeout tests |
| 8 | extract_payload, scan_forbidden | functions | scripts/task285_log_sink.py:356 | modified | sanitizer | payload and nested redaction tests |
| 9 | sanitize_entries | function | scripts/task285_log_sink.py:393 | modified | polling and report projection | artifact and raw-entry tests |
| 10 | poll_for_events, verify_event_contract | functions | scripts/task285_log_sink.py:435 | modified | verifier and evaluator | delay, timeout, exact-outcome tests |
| 11 | SafeArgumentParser, module loading, and runner constants | CLI setup | scripts/run-task285-acceptance.py:39 | modified | main | invalid-input test |
| 12 | resolve_addresses, is_public_address, validate_deployed_url | target functions | scripts/run-task285-acceptance.py:61 | modified | verify | public, DNS, ownership tests |
| 13 | blocked_results, write_json, write_blocked_evidence | artifact functions | scripts/run-task285-acceptance.py:116 | modified | main and fallback | seven-row and artifact tests |
| 14 | run_playwright | function | scripts/run-task285-acceptance.py:152 | modified | verify | timeout, spawn, provenance tests |
| 15 | query_window, safe_probes, retention_probe | helpers | scripts/run-task285-acceptance.py:192 | modified | verify | input and retention tests |
| 16 | evaluate, verify | functions | scripts/run-task285-acceptance.py:239 | modified | main | complete contract and verifier tests |
| 17 | finalize_report, write_fallback_report, result_exit_code, main | functions | scripts/run-task285-acceptance.py:355 | modified | CLI | report, failure, and exit tests |
| 18 | Playwright target validation and defineConfig | config logic | frontend/playwright.task285.config.ts:5 | modified | browser runner | typecheck and fail-closed config run |
| 19 | fixture, envelope, record, csrf, login, manualItem | browser helpers | frontend/tests/task285-centralized-logging.spec.ts:32 | modified | four scenarios | Playwright suite |
| 20 | Playwright beforeAll and afterAll action lifecycle | browser lifecycle | frontend/tests/task285-centralized-logging.spec.ts:84 | modified | action receipt | Playwright suite |
| 21 | Four centralized-logging scenarios | browser scenarios | frontend/tests/task285-centralized-logging.spec.ts:110 | modified | Playwright runner | full browser gate |
| 22 | timestamp, action_receipt, cloud_entry, FakeClient | test fixtures and double | scripts/test_run_task285_acceptance.py:23 | modified | focused tests | 21 tests |
| 23 | Task285SinkTests | test class | scripts/test_run_task285_acceptance.py:94 | modified | unittest discovery | 21 focused tests |
| 24 | _write_text, safe_print, and Phase 08 gate functions | gate functions | scripts/check.py:26 | modified | aggregate static lane | full gate and check tests |

inventory_source_count: 24
audited_symbol_count: 24
inventory_complete: true
generated_groupings:
- Small declarations and browser helper/lifecycle units are grouped only where they share one contract; all four browser scenarios remain represented in row 21.

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| FORBIDDEN_KEY_RE and canonical contracts | Fixed event vocabulary and redaction policy. | Handles known forbidden keys and tuple mutation. | Immutable constants. | Pattern policy is too permissive for complete raw-entry rejection. | Constant lookup-sized policy. | Centralized and small. | Misses accepted displayName, authorization, userAgent, and text probes. | FAIL |
| SinkError and SinkConfig.from_environment | Separates auth, availability, malformed, and redaction failures; enforces project/resource and numeric bounds. | Missing or malformed config is rejected. | No retained resources. | Least-scope project/resource checks. | Bounded settings. | Idiomatic validation. | Configuration and error tests pass. | PASS |
| QueryWindow, ExpectedEvent, SafeEvent | Closed windows, safe IDs, timestamps, and fixed projection shape. | Reversed, malformed, or unsafe values reject. | Immutable dataclasses. | Safe request IDs only. | Fixed-shape values. | Minimal public shapes. | Window and fixture tests pass. | PASS |
| Timestamp helpers | Canonical UTC conversion and parse contract. | Bad timezone/value paths reject. | No resources. | No sensitive output. | Constant work. | Standard datetime use. | Timestamp tests pass. | PASS |
| Receipt validators and secrets_compare | Exact 11 canonical tuples, provenance, ordered times, and safe IDs. | Mutated, reversed, wrong-provenance, or malformed receipts reject. | No external state. | Constant-time provenance comparison. | Linear receipt validation. | Clear invariant boundary. | Adversarial receipt tests pass. | PASS |
| Cloud client setup and token/open methods | Configured endpoint and auth are bounded and credentials stay out of request bodies and errors. | Auth, malformed, and open errors classify safely. | Local token state only; deadlines supplied by query. | Least-scope auth and redaction. | Per-request I/O bounded. | Small adapter. | Adapter/auth tests pass. | PASS |
| CloudLoggingClient.query | One monotonic deadline spans token and all pages. | Caps pages, entries, bytes, malformed responses, and timeout. | Remaining budget passed to each page. | No credential in evidence. | Bounded pagination and body. | Correct cumulative timeout. | Slow two-page test passes. | PASS |
| extract_payload and scan_forbidden | Intended complete raw-entry scan and exact payload extraction. | Recurses maps and lists but accepts sensitive-looking variants and ordinary diagnostics. | No resources. | Blocking: four direct nested raw-field probes returned accepted. | Linear traversal. | Regex policy is incomplete. | Existing tests do not cover camelCase name, authorization, userAgent, or text. | FAIL |
| sanitize_entries | Must scan raw entry before safe projection. | Inherits scan bypass and returns SafeEvent for unsafe raw entries. | No external state. | Blocking raw sensitive metadata can cross the projection boundary. | No explicit raw allowlist. | Readable flow, unsafe policy. | Fresh four-probe reproduction fails invariant. | FAIL |
| poll_for_events and verify_event_contract | Delayed ingestion, exact one-event correlation, and no false PASS. | Missing, duplicate, wrong outcome, and timeout paths are explicit. | One deadline per poll; bounded sleep. | Only projected events are reported. | Bounded loop. | Explicit matching. | Delay, timeout, and contract tests pass. | PASS |
| Runner parser and module setup | Restricts CLI and task module inputs. | Invalid options become safe blocked reports. | No resources. | No public caller receipt or action-file input. | Minimal setup. | Clear parser override. | Invalid-input test passes. | PASS |
| Target resolvers and validate_deployed_url | Exact ack and host, HTTPS, no URL extras, and every resolved address global. | Rejects private, loopback, link-local, reserved, mapped, mismatch, and private DNS targets. | Resolver is test-injectable; DNS timeout is not explicit. | Strong Python entry-point boundary. | One address set. | Good ipaddress use. | Target adversarial tests pass. | PASS |
| Blocked and artifact helpers | Exactly seven safe rows plus sink evidence. | All blocked roots preserve nonzero exit and no traceback. | Per-run files. | No credentials or sensitive fields. | Fixed output. | Deterministic schema. | Seven-row tests pass. | PASS |
| run_playwright | Internally owns action file and 48-hex per-run provenance. | Timeout, spawn, and nonzero failures block safely. | Subprocess timeout bounds execution. | Does not trust caller receipt or local output. | Suppressed output and bounded subprocess. | Private test seam only. | Actions-file rejection and process tests pass. | PASS |
| Window, probe, and retention helpers | Bounded query windows, safe environment probes, and 90-day retention evidence. | Short, malformed, young, or unsafe values reject. | Bounded sink query. | No raw retention payload. | Fixed checks. | Explicit age validation. | Retention and input tests pass. | PASS |
| evaluate and verify | Maps seven criteria and orchestrates target, generated actions, receipt, sink, retention, and projection. | Action, sink, and config failures remain structured. | Per-run provenance/directories; bounded phase operations. | Independent sink evidence and canonical receipt. | Bounded subprocess and query paths. | Cohesive orchestration. | End-to-end fake-client test passes; sanitizer gap remains. | PASS |
| Report and exit functions | Task 280 report or safe fallback, with 0, 1, and 2 semantics. | Report timeout/spawn and unexpected errors preserve seven rows and finding. | Bounded report subprocess. | No traceback or secret leakage. | Fixed report timeout. | Clear top-level behavior. | Report and exit tests pass. | PASS |
| Playwright target validation and config | Requires ack, exact host, HTTPS, no URL extras, and rejects literal local/IP targets. | Does not resolve hostname or check resolved public addresses. | Browser config starts independently of Python resolver. | Important: exact approved private DNS name can pass this guard. | No DNS I/O. | Duplicated boundary is incomplete. | Literal tests pass; private-DNS config case missing. | FAIL |
| Browser helpers, lifecycle, and scenarios | Fixed action/resource/outcome envelopes; generated provenance; four required workflows. | Success/failure/audit distinctions and cleanup are represented. | afterAll writes internal receipt. | No caller-controlled receipt accepted. | Browser output not used as sink evidence. | Explicit contract. | Typecheck and browser gate pass; deployed gate correctly blocked. | PASS |
| Test fixtures, FakeClient, and Task285SinkTests | Deterministic safe events and bounded failure seams. | Covers malformed/auth/delay/timeout/report paths. | No external shared state. | Missing four raw-field regressions is material. | Fast focused suite. | Well-scoped tests. | 21 pass but suite is insufficient for full sensitive scan. | FAIL |
| scripts/check.py output and Phase 08 gate functions | Complete synchronized output and aggregate Task 285 validation wiring. | Gate failures propagate; lane output remains complete. | Lock and nonblocking write loop. | No sensitive transformation. | Bounded writes and parallel lanes. | Fits existing framework. | Full gate and validators pass. | PASS |
| Browser action lifecycle | beforeAll and afterAll own per-run action state and receipt provenance. | Cleanup and receipt-write errors are surfaced through the producer. | Browser lifecycle is scoped to one run. | Caller cannot supply the receipt path as public evidence. | Fixed-size action receipt. | Clear lifecycle seam. | Playwright suite and typecheck pass. | PASS |
| Four browser scenarios | Required authentication, manual-item, classification/admin, and external search/import/dependency workflows are represented. | Success, failure, and audit outcomes are distinct. | Playwright owns the page lifecycle. | Fixture data is bounded and no local output is accepted as sink evidence. | Browser run is gate-bounded. | Scenario ownership is explicit. | Full browser gate passes; deployment gate is correctly blocked. | PASS |
| Task285SinkTests class | Focused regression class covers repaired contracts and error semantics. | Includes target, receipt, timeout, malformed, and report paths. | No shared external state. | Missing four raw-field regressions remains material. | Fast unit suite. | Well-scoped standard unittest class. | 21 pass but suite is insufficient for full sensitive scan. | FAIL |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| blocking | scripts/task285_log_sink.py:364-403 | scan_forbidden, sanitize_entries | Complete nested sensitive-data scanning is not enforced. | Fresh direct probes with a valid safe jsonPayload plus labels.displayName, textPayload INFO user Alice, httpRequest.userAgent, and labels.authorization each returned ACCEPTED and a SafeEvent. These are names, console or diagnostic text, request metadata, and credential material within the raw entry. | Enforce an allowlist for raw entry fields or complete the recursive key/value policy for camelCase names, credential/key material, diagnostics, short probes, text payloads, and request metadata; add regression tests for all four shapes and nested variants. |
| important | frontend/playwright.task285.config.ts:5-25 | standalone target guard | The browser configuration checks hostname equality and literal IPs but never resolves the approved hostname or verifies all resolved addresses are public/global. | A direct acknowledged HTTPS config using the exact approved hostname can pass this guard even when that name resolves to a private address; only the Python runner performs the DNS publicness check. | Share one resolver-backed target guard with the browser entry point, or require a verified public-target attestation from the runner. Add a private-DNS config test. |

blocking_findings: 1
important_findings: 1
optional_findings: 0

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| python3 -m unittest -v scripts/test_run_task285_acceptance.py | repo | 0 | PASS: 21 tests | focused suite |
| python3 -m unittest -v scripts/test_run_task285_acceptance.py scripts/test_phase08_acceptance.py | repo | 0 | PASS: 46 tests | focused plus dependency suite |
| python3 scripts/phase08_acceptance.py validate | repo | 0 | PASS: 12 scenarios, 91 criteria | manifest validator |
| python3 scripts/validate-task-list.py | repo | 0 | PASS: 286 tasks | task-list validator |
| python3 scripts/validate-traceability.py | repo | 0 | PASS | traceability validator |
| bun run typecheck | frontend | 0 | PASS | TypeScript gate |
| git diff --check | repo | 0 | PASS | whitespace check |
| env -i PATH=$PATH python3 scripts/run-task285-acceptance.py | repo | 2 | PASS: truthful blocked path | logs/phase08-acceptance/task285-b69b85608c127c87861408ab/report.json |
| python3 scripts/check.py --output logs/task285-review-full-check.html | repo | 0 | PASS: full gate, backend race lane, frontend 537 tests/build, browser 309 passed and 77 skipped | logs/task285-review-full-check.html |
| GOCACHE=... GOMODCACHE=... MEALSWAPP_REDIS_URL=redis://localhost:6379/11 go test -race ./... -p 1 -count=1 | backend | 0 | PASS: isolated full backend race | command output |
| direct sanitize_entries raw-entry probes for nested name, text, http, and authorization fields | repo | 0 | FAIL: four unsafe entries accepted | reviewer reproduction |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-285-rereview.md | repo | 0 | PASS: structural evidence validation | this file |

## 9. Files Inspected and Staleness Fingerprints

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| scripts/task285_log_sink.py | sink scope, query, timeout, raw scan, projection | blocking nested scan bypass | SHA256 | a596cbfac48940cc2895122cda84e28d2df1aecfe58168b41a3446a13b0b2f88 |
| scripts/run-task285-acceptance.py | target/action/report/exit orchestration | runner checks pass | SHA256 | 29f8cce15ef70e5e208dad874673f36a9033eeefb70eebad13d1553de57c0f58 |
| scripts/test_run_task285_acceptance.py | focused regression suite | missing four raw-field regressions | SHA256 | a3161142c6af64014bdaa28335466006f917ae6bba92aed7c338d5ad1369b7e6 |
| frontend/playwright.task285.config.ts | independent browser target guard | important private-DNS gap | SHA256 | c90b0c07254a5d8f8bcd801fdc01a595948975c1b00b74ae0deca1cd1393052c |
| frontend/tests/task285-centralized-logging.spec.ts | action producer and four scenarios | no blocking finding | SHA256 | 040f0e5763f9f73c65f1f8b701de57dc0854bbac936ba008ac3e469fbbe5137e |
| scripts/check.py | aggregate Phase 08 gate wiring | no blocking finding | SHA256 | 4343808d63552d481671815303456185ef76ef0aa0752cb5ae84756fa480d281 |
| docs/implementation/02_TASK_LIST.md | status integrity check | pre-existing PREPARED, not edited | SHA256 | 64803ce4d4af01bcda803dc843623c2ab20fae73de75b4701153c6602946aa63 |
| logs/phase08-acceptance/task285-b69b85608c127c87861408ab/report.json | fresh seven-row report | truthful BLOCKED | SHA256 | 3bc71f7fb7082ea466a3e7c796b1d71e46ef773cff9bb4caf4e15d6e2d6d8b40 |
| logs/task285-deployed/b69b85608c127c87861408ab/sink-evidence.json | fresh sink evidence | zero observations, blocked | SHA256 | 124b899b1a6ad0405b38ad91b835db9b665b83bc2e4131f2d83fb5001c0a5585 |
| logs/task285-review-full-check.html | fresh full-gate report | all aggregate lanes passed | SHA256 | 26dc12d9bea2599964ebb087ce54c3b7d99efa752e5389f7baf76586cf20eef3 |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
- docs/implementation/evidence/task-285-review.md and task-285-preparation.md contain earlier hashes and were not treated as current proof.

## 10. Coverage and Exceptions

- [x] Required focused and aggregate coverage/test commands ran.
- [x] Full gate recorded backend aggregate coverage 87.4% and Phase 08 coverage 93.2% with 4645/4986 lines; the task row has no coverage exception.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] No coverage exception was invented; the redaction gap is a correctness finding.

coverage_required: true
coverage_exception_allowed: false
coverage_report_path: logs/task285-review-full-check.html
observed_line_coverage: backend aggregate 87.4%; Phase 08 93.2%
coverage_passed: true

Coverage finding: global and Phase 08 gates pass, but focused coverage does not prove the required complete sensitive-data policy because the missing adversarial inputs are not tested.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by Task 285.
- [x] Source-of-truth task/design documents were inspected.
- [x] No generated/cache/build/temporary artifact was counted as an implementation change.
- [x] Public actions-file input was searched and is absent from the production parser; only the intentional invalid-input regression test mentions it.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: cumulative pagination timeout, per-run provenance and canonical tuples, seven-row fallback reports, truthful P08-FIND-285-001 synchronization, and full/isolated race validation pass. The raw sensitive-data policy and independent browser DNS boundary do not.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains.

decision: REJECTED
reason: The repaired verifier still accepts sensitive nested raw-entry fields and its independent Playwright target guard does not enforce public DNS resolution.
failed_criteria:
- Complete nested sensitive-data scanning and security/redaction acceptance.
- Strict public/owned target enforcement at every entry point.
failed_or_unaudited_symbols:
- scan_forbidden
- sanitize_entries
- frontend/playwright.task285.config.ts target validation
recommended_next_action: Repair both findings, add the four raw-entry and private-DNS regression tests, rerun focused, full, and isolated-race checks, and request another re-review without changing task-list status.

## 13. Repair Context

### Failure Summary

The implementation recursively walks raw entries, but its regex policy does not reject all names, credentials, diagnostics, or text/request metadata. A safe payload combined with each of four sensitive nested shapes was accepted. The independent TypeScript Playwright config also trusts an approved hostname without resolving its addresses, so it does not itself prove public reachability.

### Minimal Repair Goal

Make the sanitizer fail closed for every raw field except the explicitly required safe event projection, or implement a complete recursive policy covering exact and camelCase sensitive keys, credential/key material, names, diagnostics, short probes, text payloads, and request metadata. Make the browser entry point use the same resolver-backed public/owned target attestation as the Python runner. Add direct regression tests for the observed inputs.

### Evidence to Reuse

The fresh seven-row blocked report and sink evidence, focused/dependency test results, full aggregate report, isolated backend race result, action receipt/provenance tests, cumulative timeout test, and hashes in section 9 remain valid for unchanged passing behavior.

### Required Re-Review Surface

scan_forbidden, sanitize_entries, extract_payload, FORBIDDEN_KEY_RE, the Playwright target validation, new adversarial tests, and all callers in run-task285-acceptance.py and the Playwright config. Recheck seven-row failure semantics and fresh blocked evidence after repair.

### Do Not Change

Do not weaken canonical action/provenance validation, cumulative deadlines, bounded query limits, sanitized evidence, seven-row report shape, truthful P08-FIND-285-001 BLOCKED synchronization, full/isolated race gates, or the PREPARED task-list status.
