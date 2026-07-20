# Review Evidence: Task 231 — Optimization Retry and Identity Lifecycle

~~~yaml
task_id: 231
phase: "07.01"
component: "DESIGN-017: RetryManager"
static_aspect: "RetryManager optimization retry policy, lifecycle, ownership, and bounded polling"
input_status: "OPEN (preserved; task-status edits were prohibited)"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-18T13:04:54Z"
review_agent: "Codex independent owner re-review after F-231-01/F-231-02/F-231-03 repair"
evidence_file: "docs/implementation/reviews/task-231-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus cumulative dirty Phase 07.01 worktree"
baseline_confidence: "HIGH"
inventory_symbol_count: 20
audited_symbol_count: 20
inventory_source_count: 20
code_review_skill_invoked: true
code_review_skill_invocation_count: 1
relevant_language_guides: "TypeScript, Svelte, security, async/concurrency, common-bugs, architecture, performance, and universal-quality guidance applied"
review_template_path: "docs/implementation/reviews/REVIEW_TEMPLATE.md"
review_template_available: false
fallback_review_template_path: "review.txt"
fallback_review_template_sha256: "f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20"
prior_rejected_review_sha256_before_rewrite: "b7b7b53b3a763e6743bd8d4db32f459da3466c74cba7cb970211ba2d97bc35d7"
preparation_sha256_at_re_review: "1cc8c542cc273059d6fb655faeca5173f8abed21735e393d1b50db8aee82708a"
prior_evidence_checked_for_staleness: true
all_reviewed_files_hashed: true
repair_context_required: true
~~~

## 1. Task Source

Task 231 is the OPEN row at docs/implementation/02_TASK_LIST.md:238:

> Phase 07.01: define an explicit failure-code-to-retry action policy using current user input, make controller disposal/remount and authenticated identity changes deliberately reset or resume shared optimization state, prevent multiple controllers from racing, and validate bounded polling configuration with leak-free abortable delays.

The row requires exact ambiguity handling before and after acknowledgement, explicit action mapping for transport and terminal outcomes, current-input retry behavior, submit/queued/processing remount behavior, logout and account cleanup, controller ownership and handoff, bounded polling configuration, and exactly-once abortable delay settlement.

The design owner is the full docs/design/DESIGN-017.md, including the RetryManager responsibilities, retry algorithm, error/state model, and component interfaces. The optimization status and terminal-code boundary is inherited from the already-passed Task 230 client contract and was inspected only as a supporting boundary.

The refreshed docs/implementation/preparation/task-231-preparation.md claims that F-231-01, F-231-02, and F-231-03 were repaired. The prior rejected review was read in full and its three findings were independently rechecked against the current source and focused tests:

- F-231-01: authenticated teardown now calls clearOptimizationIdentity before the workflow is removed.
- F-231-02: owner disposal now releases the owner and resumes the first still-registered successor.
- F-231-03: every deliberate fresh submission resets public and private operation artifacts before secure key allocation.

The requested docs/implementation/reviews/REVIEW_TEMPLATE.md is absent from this checkout and from the current tree. The complete available code-review template at /home/wiktor/.agents/skills/code-review-skill/assets/pr-review-template.md, the complete root review.txt fallback, docs/implementation/reviewer-prompt.md, and the repository's validator-backed 13-section evidence structure were read. No missing template was created or modified.

The task-list preparation fingerprint is stale only because concurrent work appended Task 213 through Task 237 rows after the preparation snapshot. The current Task 231 row remains byte-identical, at line 238, with status OPEN; the current task-list hash is authoritative and is recorded in Section 9.

## 2. Pre-Review Gates

- [x] The exact Task 231 row was read; it remains OPEN and was not edited.
- [x] Full docs/design/DESIGN-017.md was read, including RetryManager responsibilities, algorithms, state/error model, and interfaces.
- [x] The refreshed Task 231 preparation evidence was read in full, including its acceptance matrix, commands, and hashes.
- [x] The prior rejected Task 231 review was read in full; its three findings were checked for staleness and reproduced against the repaired source/tests.
- [x] The available complete review template, fallback review instructions, and validator-backed 13-section schema were read; the requested repository template path was checked and is absent.
- [x] The current optimization runtime/controller, retry policy, identity boundary, workflow, Daily Diet auth handoff, supporting client boundary, focused tests, and browser fixtures were audited.
- [x] code-review-skill was invoked exactly once; its applicable TypeScript, Svelte, security, concurrency, common-bug, architecture, performance, and universal-quality guidance was applied.
- [x] F-231-01 was directly checked through the real Daily Diet auth transition call site and the shared-runtime teardown test.
- [x] F-231-02 was directly checked through the mounted successor-controller test, including owner release, late-result invalidation, and same-job resume.
- [x] F-231-03 was directly checked for both direct fresh intent and policy-driven retry over an existing completed result.
- [x] Focused Task 231 tests pass: 40 tests and 190 expectations.
- [x] Focused coverage reports optimization.ts at 100.00% lines and 98.00% functions.
- [x] Full frontend tests pass: 436 tests and 1,985 expectations; full frontend coverage is 94.86% lines and 94.01% functions.
- [x] Frontend typecheck, production build, 18 desktop/mobile Chromium tests, task-list validation, traceability validation, and scoped diff checks pass.
- [x] The root aggregate check was run. Its Task 231-relevant stages pass; it exits at the unrelated Phase 07 Go coverage gate for existing backend packages below 100%.
- [x] No production code, task status, design source, preparation evidence, generated contract, or unrelated worktree change was edited by this review.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "None. F-231-01, F-231-02, and F-231-03 are repaired and independently verified."
~~~

## 3. Review Baseline and Change Surface

The baseline is commit a4e31367485b03269e90b5607f2057c9568bb5b1 plus the cumulative dirty Phase 07.01 worktree. The branch is ahead of its remote by one commit and contains concurrent backend, API, migration, frontend, preparation, and review changes. The task-owned surface was reconstructed from the exact row, preparation manifest, prior review, current symbols, direct callers, tests, and content hashes rather than from the aggregate dirty diff.

The Task 231 implementation surface is five tracked modified files plus the untracked focused frontend/src/lib/components/DailyDietControls.test.ts. The scoped tracked diff is 971 insertions and 116 deletions; the additional test file is 27 lines. These counts include the complete Task 231 implementation and its prior repair work, not unrelated Phase 07.01 changes.

No merge was performed because merging into this shared dirty worktree would mutate unrelated user work. No reset, checkout, staging, cleanup, production-code edit, or task-status edit was performed.

The audited contract is:

1. Every approved failure code and operation stage maps to exactly one retry action: no retry, exact pre-ack replay, acknowledged-job polling, or fresh submission.
2. Retry compares current tolerance and ordered exclusions with the immutable request snapshot and rotates only when policy or current intent requires it.
3. One WeakMap runtime owns identity, diet scope, pending request/key, retry action, operation generation, abort controller, registrations, and one live owner per writable store.
4. Disposal aborts and invalidates active work while preserving only approved remount intent; authenticated teardown clears all protected state, scopes, retry material, and late completions.
5. Polling configuration is finite, bounded, total-window-safe, exactly limited, abortable, and leak-free.

Architecture and performance assessment: the store remains the single retry/lifecycle boundary, UI components only provide identity, diet, and current form input, and the caller-owned idempotency key remains closure-local. Polling is bounded by construction and uses one request per configured attempt. No new persistent storage, network boundary, unbounded loop, or user-data logging path was introduced.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Explicit failure-code and stage retry policy covers ambiguity, transport, terminal, auth, entitlement, expiry, and not-found outcomes. | Policy source and table-driven policy test. | PASS | optimizationRetryAction maps every listed stage/code family to none, replay_submission, poll_job, or new_submission; the exhaustive policy test passes. |
| 2 | Ambiguous submission before acknowledgement reuses the exact key and request. | Controller test observing API keys and cloned request data. | PASS | The pre-ack network/queue tests replay the same key and request without allocating another key. |
| 3 | Retry after acknowledgement polls the exact job without resubmission or key allocation. | Poll-outage, repeated-outage, timeout, and acknowledged-job tests. | PASS | Post-ack transport failures use poll_job, preserve the acknowledged job ID, and perform no submission or key allocation. |
| 4 | Expiry/not-found and terminal solver, worker, and cancellation outcomes use fresh current-input submission; validation, auth, and entitlement do not retry. | Table-driven terminal and operation tests plus safe-message inspection. | PASS | result_expired, optimization_not_found, terminal solver/worker/cancelled states rotate; validation, session, entitlement, and failed-validation states hide retry. |
| 5 | Edited tolerance or ordered exclusions use current input and rotate at both ambiguity boundaries. | Pre-ack and post-ack edited-input test. | PASS | Both boundaries allocate key-2 and submit the edited request; snapshot comparison preserves ordered exclusions. |
| 6 | Repeated active submit clicks are suppressed and deliberate later intent gets a fresh key. | Deferred concurrent-submit test and deliberate-submit assertion. | PASS | One active call is observed; a later deliberate submit rotates from key-1 to key-2. |
| 7 | Retry/new button visibility follows policy and the workflow passes current form input. | Workflow source contract, component tests, and browser workflow. | PASS | Retry is hidden for none and completed states, the completed fresh-generation action is separate, and retry receives activeRequest. |
| 8 | Disposal/remount during submission replays exact request/key; queued and processing remount resumes the acknowledged job. | Three remount lifecycle tests. | PASS | Submission replay uses one memory-only key; queued and processing remounts poll the same job with no second submission. |
| 9 | Direct identity and diet scope changes abort, clear, and invalidate late work. | Store scope-reset and late-completion tests. | PASS | setIdentity/setDiet increment the shared generation, abort active work, clear pending/retry/result state, and reject stale commits. |
| 10 | Auth-driven logout and account replacement clear shared optimization state, private keys, retry intent, and polls before workflow removal or re-scoping. | Actual Daily Diet auth transition call site plus shared teardown test. | PASS | DailyDietControls calls clearOptimizationIdentity on logout and user change; the reset aborts work, nulls identity/diet/pending/action, clears registered scopes, and sets empty public state. |
| 11 | Multiple controllers cannot race; a mounted successor receives ownership and resumes safely after owner disposal. | Active-owner race test and mounted-successor handoff test. | PASS | Non-owner mutations are suppressed; disposal releases ownership, selects the registered successor, resumes the same acknowledged job, and late old-owner results cannot commit. |
| 12 | Empty, invalid, non-finite, over-minute, excessive, or over-total polling configurations fail before controller use. | Configuration table test. | PASS | Empty schedule, negative/NaN/infinite/over-minute delay, invalid poll count, and over-ten-minute total all throw RangeError. |
| 13 | Polling performs exactly the configured maximum number of polls and reuses the final delay. | Deterministic delay/poll-count test. | PASS | Four polls produce delays [5, 10, 10, 10] and a stable optimization_poll_timeout. |
| 14 | Abortable delays install one listener/timer and settle exactly once on timer, abort, and pre-abort paths. | Fake timer/listener tests. | PASS | Timer and abort paths each remove one listener and clear one timer; repeated callbacks are no-ops and pre-abort installs neither resource. |
| 15 | Secure-random/key-generation failure after a completed result clears stale job, DTO, alternatives, pending key/request, and retry material. | Existing-result direct-fresh and policy-retry regression. | PASS | Both repaired paths leave only a safe secure_random_unavailable failure and empty initial artifacts; no submit or poll occurs. |
| 16 | Review evidence, source hashes, focused/full tests, browser/static checks, and status boundaries are accurate. | Current commands, status, and SHA-256 fingerprints. | PASS | Current hashes are recorded in Section 9; Task 231 remains OPEN; frontend and focused gates pass. The unrelated aggregate Go coverage failure is explicitly scoped out. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | Optimization phases, retry modes/actions, failure stages, state, and initial state | Types/state factory | frontend/src/lib/stores/optimization.ts:19-65 | Modified | Workflow and controller store | Store and component suites |
| 2 | optimizationRetryAction | Policy function | frontend/src/lib/stores/optimization.ts:142-162 | Added | Failure projection and terminal polling | Exhaustive policy test |
| 3 | Controller options, SharedOptimizationRuntime, registrations, WeakMap constants, and runtimeFor | Runtime/config boundary | frontend/src/lib/stores/optimization.ts:67-126,395-412 | Modified | Every controller and auth teardown | Scope, config, race tests |
| 4 | clearOptimizationIdentity | Auth-boundary reset | frontend/src/lib/stores/optimization.ts:128-140 | Added | DailyDietControls and logout/account transition | Teardown and component source tests |
| 5 | createOptimizationController and claim | Controller factory/ownership | frontend/src/lib/stores/optimization.ts:164-181 | Modified | OptimizationWorkflow and remounts | Concurrent-owner tests |
| 6 | setIdentity, setDiet, and resetScope | Scope lifecycle | frontend/src/lib/stores/optimization.ts:183-202 | Added/modified | Workflow scope effect and auth boundary | Identity, diet, logout tests |
| 7 | resume and resumeOwned | Remount/handoff | frontend/src/lib/stores/optimization.ts:204-215 | Added/modified | Workflow mount and owner disposal | Submit/queued/processing/handoff tests |
| 8 | submit and secure key failure path | Submission lifecycle | frontend/src/lib/stores/optimization.ts:217-237 | Modified | Submit and fresh-generation button | Key, concurrency, stale-result tests |
| 9 | retry, submitFresh, sameRequest, and retryMode | Retry/request comparison | frontend/src/lib/stores/optimization.ts:239-249,422-432 | Added/modified | Workflow retry and fresh action | Policy, edited-input, key tests |
| 10 | runSubmission | Acknowledgement boundary | frontend/src/lib/stores/optimization.ts:252-277 | Modified | Submit and pre-ack resume | Ambiguity, acknowledgement, abort tests |
| 11 | pollExistingJob and pollJob | Bounded polling | frontend/src/lib/stores/optimization.ts:279-327 | Modified | Acknowledged submit, retry, remount | Status, timeout, expiry, terminal tests |
| 12 | beginOperation, cancelOperation, and isCurrent | Generation/abort control | frontend/src/lib/stores/optimization.ts:329-352 | Modified | Scope changes, disposal, polling | Late-result and cancellation tests |
| 13 | setOperationFailure, displayFailure, displayError, and abort classification | Error projection | frontend/src/lib/stores/optimization.ts:354-363,443-465 | Modified | Submission/poll catches and UI | Safe-message and failure-policy tests |
| 14 | dispose and successor registration/handoff | Ownership teardown | frontend/src/lib/stores/optimization.ts:365-390 | Modified | Svelte workflow destruction | Remount and successor tests |
| 15 | validatePollingConfiguration | Configuration validation | frontend/src/lib/stores/optimization.ts:434-441 | Added | Controller construction | Invalid-boundary test |
| 16 | waitForOptimizationPoll | Timer/abort primitive | frontend/src/lib/stores/optimization.ts:467-485 | Added/modified | Production polling sleep | Fake timer/listener tests |
| 17 | Optimization identity/diet effects, retry, submit, and disposal | Svelte workflow | frontend/src/lib/components/OptimizationWorkflow.svelte:19-44,63-81,139-201 | Modified | Authenticated Daily Diet view | Component source and browser suites |
| 18 | Authenticated identity transition and optimization handoff | Svelte auth boundary | frontend/src/lib/components/DailyDietControls.svelte:31-47,121 | Modified | SearchShell auth props and workflow | Daily Diet controls tests |
| 19 | Store controller and lifecycle test suite | Unit tests | frontend/src/lib/stores/optimization.test.ts:116-894 | Added/modified | All controller/runtime symbols | 31 tests, 190-expectation subset |
| 20 | Component and browser workflow evidence | Component/browser tests | frontend/src/lib/components/OptimizationWorkflow.test.ts:1-75, frontend/src/lib/components/DailyDietControls.test.ts:1-27, frontend/tests/optimization-workflow.spec.ts, frontend/tests/phase07-browser-acceptance.spec.ts | Added/modified | Workflow and auth presentation | 9 focused component tests and 18 browser tests |

~~~yaml
inventory_source_count: 20
audited_symbol_count: 20
inventory_symbol_count: 20
inventory_complete: true
generated_groupings:
  - "Rows group only tightly coupled types, helpers, or test fixtures at one behavior boundary; retry policy, runtime ownership, scope lifecycle, submission, polling, UI handoff, and evidence remain separate."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| Optimization phases, retry modes/actions, state, and initial state | Public state contains only selected scope, lifecycle, validated job/result, safe failure, and retry mode; key stays private. | Initial state is deterministic and empty; terminal alternatives are bounded by the controller. | State reset starts from a complete fresh object. | No key or credential enters state or DOM. | Constant-size state plus at most three alternatives. | Narrow discriminated unions and one factory. | Focused store/component assertions; PASS. | PASS |
| optimizationRetryAction | Every accepted stage/code/retryability combination maps to one explicit action. | Auth and entitlement fail closed; expiry/not-found rotate; terminal validation hides retry; terminal solver/worker/cancelled rotate. | Pure function has no resources or races. | Safe codes and retryability are consumed after client mapping. | O(1), no I/O. | One policy boundary avoids scattered retry decisions. | Table-driven exhaustive policy test; PASS. | PASS |
| Runtime/config boundary and runtimeFor | One WeakMap runtime is shared per writable store; bounds are checked before controller use. | Defaults and injected values are copied; invalid values reject before operations. | Runtime tracks one owner, registrations, generation, abort, and private pending data. | Pending key is closure/runtime memory only. | WeakMap avoids retaining arbitrary stores; polling sums are bounded. | Shared runtime is cohesive and injectable. | Scope/config/race tests; PASS. | PASS |
| clearOptimizationIdentity | Auth teardown clears identity, diet, pending key/request, retry action, public state, and all registered scopes. | Works for active submit/poll and already-completed state; late completions fail generation check. | Increments generation, aborts active controller, nulls abort, and resets every registration. | Prevents authenticated artifacts crossing logout/account replacement. | One registration walk and one store update. | Explicit boundary is clearer than overloading remount-friendly dispose. | Teardown test plus real Daily Diet call-site assertion; PASS. | PASS |
| createOptimizationController and claim | Only the owner symbol may mutate a shared store. | First live method claims; non-owner methods return without state/API mutation. | Registration is removed on dispose; successor can claim after release. | Shared state cannot be overwritten by an unrelated mounted controller. | Map operations are O(1); no duplicate work while owned. | Small owner-token protocol. | Concurrent owner test and handoff test; PASS. | PASS |
| setIdentity, setDiet, and resetScope | Scope changes invalidate all previous operation artifacts. | Same scope is a no-op after desired scope recording; changed identity/diet resets to empty. | Reset aborts and increments generation before replacing state. | Identity and selected diet are server/request ownership inputs, never caller-trusted result fields. | One abort and one store write per scope change. | Explicit identity/diet methods match component lifecycle. | Direct identity, diet, logout, and late-result tests; PASS. | PASS |
| resume and resumeOwned | A mounted owner resumes only approved pre-ack key replay or acknowledged queued/processing poll. | Desired identity/diet are reapplied on handoff; completed/failed/no-intent states do no I/O. | Claim precedes reset/resume; successor starts after previous owner cancellation. | Auth teardown clears desired scopes and pending material before future resume. | At most one resumed operation per owner. | Public resume makes remount semantics explicit. | Submission, queued, processing, logout, and successor tests; PASS. | PASS |
| submit and secure key failure path | A deliberate intent clears old artifacts before allocating one fresh caller-owned key. | Busy phases suppress clicks; key failure becomes safe failed state with no network call. | Cancels old operation, nulls pending/action, writes initial state, then stores only the new pending snapshot. | No weak fallback or persistent key; failure message is safe. | One key allocation and one API submission per accepted intent. | Atomic fail-closed ordering closes F-231-03. | Initial and existing-completed key-failure tests; PASS. | PASS |
| retry, submitFresh, sameRequest, and retryMode | Retry uses preserved request/key only for exact approved reuse and current input for changed/fresh intent. | Missing action/pending state is a no-op; ordered exclusions and tolerance changes rotate. | Fresh retry delegates to the same clearing path as submit. | Retry cannot resurrect cleared private key after auth teardown. | O(length of exclusion list) comparison and snapshot copy. | One request comparator and mode projection. | Policy, ambiguity, edited-input, terminal, expiry tests; PASS. | PASS |
| runSubmission | Acknowledgement is the key lifecycle boundary: key becomes non-reusable and job polling owns the operation. | API errors map by submission/poll stage; malformed strict-client responses remain safe. | Generation/owner checks contain late responses; abort errors do not overwrite state. | API receives only caller key and current immutable request snapshot. | One submit plus bounded polling. | Clear transition from pending to acknowledged job. | Ambiguity, acknowledgement, malformed response, and abort tests; PASS. | PASS |
| pollExistingJob and pollJob | Poll exact job ID for configured attempts, project queued/processing/terminal states, and timeout safely. | Final configured delay is reused; expiry/not-found and terminal codes project explicit policy. | Each poll checks generation before and after sleep and API; terminal state stops immediately. | Job ID comes from validated acknowledgement/state; safe failures are rendered. | Max 1,000 polls and max ten-minute configured wait; one API call per attempt. | Deterministic loop and bounded schedule. | Status, timeout, outage, expiry, terminal, remount, browser tests; PASS. | PASS |
| beginOperation, cancelOperation, and isCurrent | Every operation gets a new generation and abort signal; only current owner may commit. | Scope, fresh submit, retry, and disposal cancel prior operations; stale results return harmlessly. | Abort controller is replaced and old listeners/work are invalidated. | Prevents prior identity/result data from crossing scope boundaries. | Constant-size lifecycle metadata. | Small generation protocol is easy to reason about. | Late poll, account switch, disposal, race tests; PASS. | PASS |
| setOperationFailure, display helpers, and abort classification | Errors become bounded user-safe state and one retry action. | Known terminal/queue/expiry codes use fixed messages; unknown errors use generic text; abort is ignored when stale. | Failure state retains only approved job context and action; no active resource is left as current after terminal handling. | Backend/Redis/URL/private diagnostics are not rendered. | Pure mapping and bounded strings. | Central display projection follows DESIGN-017. | Safe-message, validation/auth/entitlement, and unknown-error tests; PASS. | PASS |
| dispose and successor registration/handoff | Disposal releases one owner while preserving only approved remount intent and hands off to a live successor. | Submitting becomes interrupted/replayable; queued/processing state remains resumable; completed state remains terminal. | Cancels/increments generation, deletes registration, sets owner null, then resumes first remaining registration. | Auth path invokes clear before disposal, so private keys are destroyed rather than preserved. | One map walk for successor selection. | Explicit handoff closes the prior permanently-busy owner gap. | Submission/queued/processing remount and mounted-successor tests; PASS. | PASS |
| validatePollingConfiguration | Schedule non-empty, delays finite/nonnegative/at most 60,000, polls integer 1..1,000, total at most 600,000. | NaN, infinity, negative, empty, excessive, and fractional values throw before controller creation. | No resources are allocated before validation. | Limits caller/injected timing inputs. | Validation loop is bounded by max 1,000. | Constants make policy visible. | Nine invalid configurations and valid capped schedule test; PASS. | PASS |
| waitForOptimizationPoll | One timer/listener settles once and cleans both resources on timer or abort; pre-abort installs none. | Timer resolves; abort rejects with reason; repeated callback cannot settle twice. | settled guard, clearTimeout, and listener removal cover all completion paths. | Abort reason never reaches user state because stale operation handling contains it. | One timer and one listener per wait. | Small promise primitive with explicit cleanup. | Fake timer/listener and pre-abort tests; PASS. | PASS |
| OptimizationWorkflow lifecycle and controls | Component supplies authenticated identity, selected diet, current request, resume, retry, and disposal. | Input validation handles tolerance bounds; retry/fresh buttons follow policy; form errors stay local. | Scope effect orders identity before diet and resumes after configuration; cleanup disposes controller. | UI does not expose key; authenticated identity is passed separately from request. | Rendered alternatives are bounded and polling is delegated. | Component remains presentation/form boundary. | Six source-contract tests, 18 browser cases, build; PASS. | PASS |
| DailyDietControls auth handoff | Authenticated Daily Diet owner supplies identity and clears optimization before logout/account replacement. | Initial load, user change, logout, anonymous/authenticating transition, and re-scope are covered by effect branches. | Clear occurs before protected workflow branch disappears or reuses the new account scope. | Shared optimization state cannot persist across authenticated users. | One clear call per identity transition; no new I/O. | Parent owns cross-component auth teardown. | Three focused source-contract tests plus store teardown test; PASS. | PASS |
| Store controller/lifecycle tests | Tests exercise public controller behavior and private key lifecycle at the required boundaries. | Covers success, all retry classes, malformed client output, key failures, identity, scope, remount, config, and timer errors. | Deferred promises and signals challenge late completion and ownership. | Storage access and safe message checks protect key/diagnostic boundaries. | Deterministic injected API/sleep avoids uncontrolled I/O. | Fixtures are local and readable; 31 store tests. | Focused suite: 40 tests and 190 expectations total; PASS. | PASS |
| Component and browser workflow evidence | UI contract is wired to the store and safe terminal/presentation states. | Source assertions cover auth/retry wiring; browser fixtures cover nominal, infeasible, timeout, expiry, anonymous/free, responsive, keyboard, and axe paths. | Browser workflow observes terminal/polling transitions; no stale results after diet change. | No key in rendered output; safe messages are asserted. | Desktop/mobile suites pass with bounded result rendering. | No duplicate client lifecycle implementation. | Six workflow tests, three controls tests, 18 Playwright tests; PASS. | PASS |

~~~yaml
audit_rows: 20
audit_complete: true
~~~

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| None | N/A | N/A | No unresolved correctness, security, behavior-regression, performance, or Task 231 coverage finding remains. | Prior F-231-01, F-231-02, and F-231-03 were rechecked and the repaired focused regressions pass. | None. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts src/lib/components/DailyDietControls.test.ts | frontend/ | 0 | PASS — 40 tests and 190 expectations. | Focused Task 231 suite |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts src/lib/components/DailyDietControls.test.ts | frontend/ | 0 | PASS — optimization.ts 100.00% lines and 98.00% functions. | Focused coverage stdout |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test | frontend/ | 0 | PASS — 436 tests and 1,985 expectations across 37 files. | Full frontend suite |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage | frontend/ | 0 | PASS — 94.01% functions and 94.86% lines overall; optimization.ts 100.00% lines and 98.00% functions. | Full coverage stdout |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck | frontend/ | 0 | PASS — no TypeScript errors. | Typecheck |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build | frontend/ | 0 | PASS — Vite production build, 205 modules transformed. | Build output |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/optimization-workflow.spec.ts tests/phase07-browser-acceptance.spec.ts --reporter=line | frontend/ | 0 | PASS — 18 desktop/mobile Chromium tests. | Browser/axe workflow |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS — 237 sequential tasks; Task 231 remains OPEN. | Task-list validator |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS. | Traceability validator |
| git diff --check -- frontend/src/lib/stores/optimization.ts frontend/src/lib/stores/optimization.test.ts frontend/src/lib/components/OptimizationWorkflow.svelte frontend/src/lib/components/OptimizationWorkflow.test.ts frontend/src/lib/components/DailyDietControls.svelte frontend/src/lib/components/DailyDietControls.test.ts | repository root | 0 | PASS. | Scoped whitespace/diff check |
| python3 scripts/check.py | repository root | 1 | OUT-OF-SCOPE AGGREGATE FAILURE — traceability, task-list, Go Doc, OpenAPI validity, verifier tests, vet, vulnerability scan, local stack, UAT, focused backend checks, and frontend verifier passed; the aggregate gate stopped at Phase 07 Go coverage below 100% for dailydiet 80.1%, optimization 84.1%, queue 75.8%, and worker 63.7%. | Existing/concurrent backend Phase 07 coverage; no Task 231 frontend failure. |
| sha256sum over all files listed in Section 9 | repository root | 0 | PASS — current hashes recorded after the audit; the review artifact itself is excluded while being written. | SHA-256 fingerprints |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-231-review.md | repository root | 0 | PASS — validator-clean structural evidence. | Final review evidence validator |

## 9. Files Inspected and Staleness Fingerprints

The hashes below are SHA-256 hashes of current contents captured for this re-review. The review artifact itself is excluded because it is the file being written. The prior rejected artifact hash is captured in the metadata before replacement.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| docs/implementation/02_TASK_LIST.md | Authoritative Task 231 row/status and acceptance criteria | Current row remains OPEN; preparation task-list hash is stale due concurrent rows 213-237 | SHA-256 | 12b9e2d32f6ed249be048909e2bfd89e2bdf938d3800ca3cc47b579c84aaa2d3 |
| docs/design/DESIGN-017.md | RetryManager design source | Current and preparation design source agree | SHA-256 | 5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c |
| docs/design/DESIGN-004.md | Supporting optimization status/terminal contract | Inherited boundary inspected; unchanged for this review | SHA-256 | 45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474 |
| docs/design/DESIGN-001.md | Supporting SearchView component boundary | Inherited UI boundary inspected | SHA-256 | 34d699ae93a8e5465199f3494ed41813c675f1cb3c9c1c6b6e611ba66c6142c7 |
| docs/architecture/ARCH-001.md | Supporting SPA/state boundary | Inherited architecture boundary inspected | SHA-256 | 03fcbae9676ecf278e72a621c7fd9911d0c3dc6ee15eaebcec308d010cc76833 |
| docs/design/01_TECH_STACK.md | Supporting frontend/runtime boundary | Inherited stack boundary inspected | SHA-256 | 64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338 |
| docs/implementation/preparation/task-231-preparation.md | Refreshed repair manifest and evidence | Preparation content is current; embedded task-list fingerprint is stale | SHA-256 | 1cc8c542cc273059d6fb655faeca5173f8abed21735e393d1b50db8aee82708a |
| docs/implementation/reviewer-prompt.md | Repository review procedure | Read as procedural input | SHA-256 | 92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d |
| review.txt | Full repository fallback review checklist | Read as procedural input | SHA-256 | f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20 |
| frontend/src/lib/stores/optimization.ts | Task 231 runtime/controller implementation | F-231 repairs present and pass | SHA-256 | 766d11e87663108381dae36dfc9cc4705d6b73e8d46c531933f101f25083f4bc |
| frontend/src/lib/stores/optimization.test.ts | Controller/retry/lifecycle regression suite | F-231-01/02/03 tests present and pass | SHA-256 | db3e6ce541bf5e83f726281fd8d9d4ee9eb62ba58fc8e4caa07461addf761800 |
| frontend/src/lib/components/OptimizationWorkflow.svelte | Current identity/diet/retry/resume UI handoff | Current-input and lifecycle wiring pass | SHA-256 | c8d7428c9280ec5e5379f006bfb1c6a5c6bb4184915ee64d32c5c2952e0df82c |
| frontend/src/lib/components/OptimizationWorkflow.test.ts | Workflow source contract tests | Current retry/fresh/identity wiring pass | SHA-256 | 0682288b529d27bd394c1ffb9bfb6c94fef04bc3b96334fd17dd9b4be61289a3 |
| frontend/src/lib/components/DailyDietControls.svelte | Real auth/logout/account handoff | Calls shared teardown before workflow removal/re-scope | SHA-256 | a3a4a6111fa6a8b5a6dff97fb27e8e68f1f7ce5df567f91c7999cbdecd20a98d |
| frontend/src/lib/components/DailyDietControls.test.ts | Auth teardown source contract tests | Teardown wiring test passes | SHA-256 | b1db99593ac911b4e41682bbf51b399883212995967614a28139ad1c3cb6ce28 |
| frontend/src/lib/components/SearchShell.svelte | Parent auth/session props and Daily Diet branch | Supporting boundary inspected; no Task 231 edit | SHA-256 | 584b0e0dba4ec6a8d38217816daa910b09a2bdefc7f3d0d26cf11adbba5fc6e8 |
| frontend/src/lib/stores/auth-session.ts | Existing authenticated identity projection | Supporting boundary unchanged | SHA-256 | 97944edf13db85c71873e0dcd1a93a5a62335df1f26e3bce7f04995341be1323 |
| frontend/src/lib/api/optimization-client.ts | Inherited strict key/ack/poll boundary | Supporting Task 230 contract inspected | SHA-256 | c047e9ab5bd97ac381b8efa72d6d99fa362e4973c3b60d785348715bac2b4c09 |
| frontend/src/lib/api/error-message-mapper.ts | Inherited safe code/retryability mapping | Supporting Task 227 contract inspected | SHA-256 | 7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e |
| frontend/src/lib/api/generated.ts | Inherited optimization DTO/status variants | Supporting generated contract inspected | SHA-256 | 166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae |
| frontend/tests/optimization-workflow.spec.ts | Desktop/mobile optimization browser scenarios | 18 listed browser tests pass with phase acceptance fixtures | SHA-256 | 03d6899f199eb079f91d8e2aafd6b6d3be7682150e2c826c97d20d93ed34b46d |
| frontend/tests/phase07-browser-acceptance.spec.ts | Phase 07 browser acceptance fixtures | Auth/entitlement/terminal/responsive scenarios pass | SHA-256 | b29b2ade34b1dd4b5bfca7220fc3bd43f55cd01f77014255f063ea18492a45b7 |

~~~yaml
all_reviewed_files_hashed: true
hash_scope: "All Task 231 implementation, test, identity-handoff, supporting contract, design, planning, and procedural evidence files listed above; the new review artifact is excluded while being written."
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The requested docs/implementation/reviews/REVIEW_TEMPLATE.md is absent; the complete available review template, review.txt fallback, and validator-backed schema were used."
  - "The prior rejected Task 231 artifact had blocking F-231-01 and important F-231-02/F-231-03 findings; its pre-rewrite hash is recorded in the metadata, and all three repairs were independently rechecked."
  - "The refreshed preparation embedded task-list hash 3bdabe886facb2b96875489dcce7186d13acecc0a1582ff8e2fc8cc1dff62ebf, while the current task-list hash is 12b9e2d32f6ed249be048909e2bfd89e2bdf938d3800ca3cc47b579c84aaa2d3 because concurrent rows 213-237 were appended. The Task 231 row and status remain unchanged."
  - "The cumulative worktree remains dirty across Phase 07.01; attribution was kept at exact symbols and interaction boundaries."
~~~

## 10. Coverage and Exceptions

- [x] Focused Task 231 controller/component tests pass: 40 tests and 190 expectations.
- [x] Focused coverage passes: optimization.ts is 100.00% lines and 98.00% functions.
- [x] Full frontend tests pass: 436 tests and 1,985 expectations.
- [x] Full frontend coverage passes its current aggregate evidence: 94.01% functions and 94.86% lines; optimization.ts is 100.00% lines and 98.00% functions.
- [x] Frontend typecheck, build, generated-contract use, browser/axe checks, task-list, traceability, and scoped diff checks pass.
- [x] F-231-01, F-231-02, and F-231-03 repaired branches are directly covered by focused regressions and source-level call-site assertions.
- [x] The existing Phase 07 frontend coverage deviation is unchanged; this review adds no coverage exception.
- [ ] The root aggregate scripts/check.py is green. It exits at the out-of-scope Phase 07 Go coverage gate for existing backend packages; this does not fail the Task 231 frontend scope.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "Bun test --coverage stdout; no persistent frontend coverage artifact committed"
observed_line_coverage: "100.00% optimization.ts; 94.86% aggregate frontend"
coverage_passed: true
coverage_reason: "All Task 231-specific changed-symbol and repaired-edge branches pass focused and full frontend coverage. The only nonzero aggregate command is an out-of-scope Phase 07 Go coverage gate; no Task 231 exception or status change was added."
~~~

## 11. Negative and Regression Checks

- [x] No idempotency key is stored in OptimizationState, rendered DOM, localStorage, or sessionStorage.
- [x] Pre-ack ambiguity reuses the exact key/request; post-ack retry polls the exact acknowledged job without a new key.
- [x] Fresh intent clears old job/result/error/retry artifacts before key allocation, including secure-random failure after an existing completed result.
- [x] Auth-driven logout/account replacement calls shared teardown before the authenticated workflow disappears or re-scopes.
- [x] Shared owner release invalidates late old-owner work and hands an already-mounted successor the same queued/processing job.
- [x] Identity/diet changes abort active work, clear private request/key lifecycle, and prevent stale completion commits.
- [x] Poll delays, poll count, total wait, final-delay reuse, timer clear, listener removal, and pre-abort behavior are bounded and tested.
- [x] Terminal and transport messages do not expose queue, Redis, URL, stack, or other infrastructure details.
- [x] No new persistent storage, dependency, API schema, generated artifact, or unrelated architectural boundary was introduced by this review.
- [x] Current Task 231 row remains OPEN; the current row and design source were not edited.

The root aggregate nonzero exit is documented in Sections 8 and 10 and is outside the frontend Task 231 boundary: the failing threshold is existing Phase 07 Go coverage in dailydiet, optimization, queue, and worker.

## 12. Decision

All Task 231 acceptance criteria and all 20 audited symbol units pass. The three prior findings are closed:

- F-231-01 is closed by the real DailyDietControls auth/account transition call to clearOptimizationIdentity, which aborts and clears the shared runtime before workflow removal/re-scoping.
- F-231-02 is closed by registered successor handoff in dispose, with generation invalidation preventing the old owner from committing after release.
- F-231-03 is closed by clearing operation artifacts before fresh key allocation, so secure-random failure cannot display stale completed results.

The retry policy, exact key/request reuse and rotation, current-input comparison, identity and diet resets, authenticated teardown, owner handoff, remount recovery, bounded polling, safe terminal projection, and exactly-once abortable delay behavior are all independently evidenced. Task 231 remains OPEN exactly as requested.

~~~yaml
decision: "PASSED"
review_decision: "PASSED"
blocking_findings: 0
important_findings: 0
optional_findings: 0
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Leave Task 231 OPEN until the phase orchestrator or project owner performs the separate status transition; do not change task status in this review."
~~~

## 13. Repair Context

This is a post-repair re-review of the prior rejected Task 231 evidence. The prior findings and their current closure are:

1. **F-231-01 — authenticated teardown preserved protected optimization state.** The repaired DailyDietControls effect invokes clearOptimizationIdentity on logout and when loadedUserId changes. The reset increments the shared operation generation, aborts active work, clears identity/diet/pending/action, clears every registered controller scope, and installs empty public state before the protected workflow is removed or reconfigured.
2. **F-231-02 — a mounted successor could remain permanently busy.** The repaired controller registration stores a resumable successor. dispose cancels and invalidates the old operation, releases ownership, then invokes the first still-registered successor. The focused handoff test observes the same acknowledged job being polled and completed once, while the late old-owner result cannot commit.
3. **F-231-03 — key-generation failure retained stale completed results.** The repaired submit clears the prior operation and writes initial scoped state before calling the key factory. The focused regression runs both direct fresh intent and policy-driven retry over a completed result and observes only the safe secure_random_unavailable failure with no stale job, DTO, alternatives, pending request, key, or retry action.

No production code, generated output, task row/status, design source, preparation evidence, dependency evidence, or unrelated worktree change was edited by this review. The only requested repository write is this fresh review artifact.
