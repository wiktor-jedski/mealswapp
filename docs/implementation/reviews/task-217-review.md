# Review Evidence: Task 217 — CLP Process Boundary and Codec Hardening

~~~yaml
task_id: 217
component: "DESIGN-004: LPSolverWrapper"
static_aspect: "LPSolverWrapper"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-14T21:32:11Z"
review_agent: "Codex GPT-5 independent review"
evidence_file: "docs/implementation/reviews/task-217-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill/reference/go.md"
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 07.01: make LPSolverWrapper, NewLPSolverWrapper, and CheckVersion the only solver API; enforce the approved command-runner deadline contract and bounded cleanup observability; treat the machine-readable solution as authoritative; stream bounded deterministic LP serialization; and harden version and solution parsing with the Go 1.25 iterator APIs and exact CLP grammar.

**Depends On:** Task 212 — PASSED.

**Testing Coverage Exceptions:** The task row says None. The repository has an accepted Phase 07 aggregate backend coverage exception in docs/implementation/04_OPEN.md; this review reran the aggregate measurement and introduced no new exception.

**Verification Criteria:** Repository-wide search finds no CLPSolver, NewCLPSolver, or StartupCheck; tests cover non-cooperative runner behavior under the chosen trusted seam, cleanup failure without primary-result replacement, solution-file precedence over misleading stdout, deliberate stdout-only fallback if retained, exact/over-limit early serialization rejection, deterministic constraints/bounds and canonical generated names, exact version/status/row grammar, unknown/duplicate/conflicting output, tiny-negative tolerance and material-negative rejection, timeout child termination, normal cleanup, bounded diagnostics, and packaged CLP success; Go Doc validation, gofmt, go vet, focused tests, and go test -race ./... pass.

## 2. Pre-Review Gates

- [x] Input status is PREPARED in the live source-of-truth task list. The user described the task as pending preparation, but the current row observed during review is PREPARED; no status was changed.
- [x] Dependency 212 is PASSED.
- [x] docs/implementation/preparation/task-217-preparation.md claims completion and identifies the task-owned surface.
- [x] Baseline a4e31367485b03269e90b5607f2057c9568bb5b1 resolves to HEAD; the task-owned diff is therefore the current worktree delta against that commit.
- [x] code-review-skill was invoked exactly once and its Go guide was read completely. The additional golang-security guidance was read for the child-process and codec security boundary.
- [x] The review is independent from implementation and repair; no repair was performed.
- [x] Current repository state and fresh commands were used rather than trusting preparation logs alone.
- [x] No production code or task-list status was changed by this review. Only this evidence document was added.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: git rev-parse HEAD and the supplied baseline both returned a4e31367485b03269e90b5607f2057c9568bb5b1. The task-owned change was reconstructed from the four tracked implementation/test paths named by the preparation report. The untracked preparation report was inspected as evidence, not treated as production implementation.

Commands used to reconstruct the diff:

~~~bash
git rev-parse HEAD
git diff --name-status a4e31367485b03269e90b5607f2057c9568bb5b1 -- \
  backend/internal/optimization/clp_wrapper.go \
  backend/internal/optimization/clp_wrapper_test.go \
  backend/internal/worker/worker.go \
  backend/internal/app/task206_backend_integration_test.go
git diff --unified=0 a4e31367485b03269e90b5607f2057c9568bb5b1 -- <same four paths>
git status --short --branch
~~~

Pre-existing dirty-worktree changes and exclusions:

The worktree contains unrelated Phase 07.01 changes in API, Daily Diet, repository, frontend, migration, OpenAPI, and other worker files. Those paths were excluded. The preparation report's exact task-owned paths are the four tracked paths below plus its own evidence file. No task-owned change could not be distinguished reliably.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/optimization/clp_wrapper.go | Worktree diff from supplied baseline | HIGH | Canonical solver API, trusted runner seam, cleanup, bounded serializer, version codec, solution codec, diagnostics |
| backend/internal/optimization/clp_wrapper_test.go | Worktree diff from supplied baseline | HIGH | 8 added tests and 5 modified tests covering the task criteria |
| backend/internal/worker/worker.go | Worktree diff from supplied baseline | HIGH | RunWithProcessor readiness caller changed from StartupCheck to CheckVersion |
| backend/internal/app/task206_backend_integration_test.go | Worktree diff from supplied baseline | HIGH | Real executable timeout fixture, canonical readiness caller, removed exported-runner fixture |

The preparation record docs/implementation/preparation/task-217-preparation.md was also read. It matches the reconstructed implementation surface and records no task-list edit.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Only LPSolverWrapper, NewLPSolverWrapper, and CheckVersion remain as the solver API. | Repository-wide Go-source search and caller inspection | PASS | rg found no CLPSolver, NewCLPSolver, or StartupCheck; current callers use NewLPSolverWrapper and readiness uses CheckVersion. |
| 2 | The chosen trusted runner contract handles a non-cooperative runner without a deadline-enforcement goroutine leak. | Source inspection plus focused regression test | PASS | commandRunner is package-private at clp_wrapper.go:172-176; production defaults to runOSCommand; TestLPSolverWrapperTrustedRunnerContractDoesNotLeakDeadlineGoroutine waits for a runner that ignores cancellation, then verifies timeout classification and settled return. |
| 3 | Cleanup failure is bounded/observable and cannot replace the primary result or error. | Focused success and primary-error tests | PASS | cleanupSolverDirectory sanitizes, redacts, and caps the diagnostic; TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult covers both successful result and non-zero primary error. |
| 4 | The solution file is authoritative, with only deliberate stdout fallback when the file is absent. | Focused precedence/fallback tests and Solve inspection | PASS | Solve reads solution.txt first; a present empty file is malformed; absent file falls back to stdout; the test supplies conflicting stdout and verifies the file wins. |
| 5 | LP serialization rejects exact-limit and over-limit models while writing, including early rejection before later invalid input. | Deterministic serializer tests and bounded-writer inspection | PASS | serializeLPWithLimit bounds every write; exact-limit, one-byte-over, and early-limit-before-later-invalid-constraint cases pass with ErrSolverOutputLimit. |
| 6 | Constraints, bounds, and generated names are deterministic and canonical. | Repeated byte-for-byte serializer test | PASS | TestSerializeLPIsDeterministicAndUsesCanonicalGeneratedNames repeats serialization three times and verifies x000001-style variables, c000001 constraints, ranged upper names, signed coefficients, and zero omission. |
| 7 | Version, status, and row codecs follow the exact supported CLP grammar and Go 1.25 iterator APIs. | Source inspection, tests, Go version/toolchain check | PASS | clpVersion uses strings.FieldsSeq; solution parsing uses bytes.Lines and bytes.FieldsSeq; exact headers, punctuation, lookalikes, indexed rows, and ** rows pass; go.mod targets Go 1.25.0 and the available Go 1.26 toolchain provides the APIs. |
| 8 | Unknown, duplicate, and conflicting machine output is rejected while unrelated diagnostics are ignored. | Table-driven parser tests | PASS | Tests cover unknown generated variables, duplicate variables, duplicate statuses, conflicting statuses, and a diagnostic line containing a generated-looking token. |
| 9 | Tiny negative numerical residuals clamp to zero while material negatives fail. | Epsilon boundary tests and downstream tolerance inspection | PASS | parseCLPSolutionLine uses the documented SolutionValidationEpsilon of 1e-9; -0.0000000005 clamps and -0.000001 rejects. |
| 10 | A timed-out native child terminates and normal/timeout cleanup removes temporary state. | Real executable fixture and temp-root assertions | PASS | TestLPSolverWrapperTerminatesRealChildAndCleansDeadlineDirectory runs an executable sleep 30 fixture under a 20ms wrapper timeout, asserts prompt return and empty temp root; the optimal-path test asserts the job directory is gone. |
| 11 | Solver-controlled output and cleanup diagnostics remain bounded and sanitized. | Output overflow, cleanup failure, and source inspection | PASS | limitedBuffer, bounded solution-file reads, solverDiagnostic, and sanitizeSolverOutput cap captured data; tests reject ANSI escapes and oversized diagnostics. |
| 12 | Packaged CLP version/readiness and solve behavior succeed. | Focused packaged executable test and full backend tests | PASS | TestLPSolverWrapperRunsPackagedExecutableWhenAvailable passed against local pinned CLP 1.17.11; Task 206 integration callers also use CheckVersion; full backend test and race suites passed. |
| 13 | Go Doc validation, formatting, vet, focused tests, and full race checks pass. | Fresh commands | PASS | validate-phase07-go-doc.py, gofmt -d, go vet ./..., focused tests, go test ./..., and go test -race ./... all exited successfully. |

## 5. Changed-Symbol Inventory

The inventory includes every added, modified, and removed executable/type/API unit in the four task-owned diffs. Unchanged helper units used to establish the caller/data-flow contract are listed in the inspected-files section, not duplicated here.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | commandRunner | private function type | clp_wrapper.go:176 | modified from exported CommandRunner | CLPConfig, Solve, CheckVersion | package tests inject it |
| 2 | CLPConfig | configuration type | clp_wrapper.go:182 | modified | worker, cmd worker, app integration, wrapper tests | wrapper and worker tests |
| 3 | CLPSolver | removed type alias | baseline clp_wrapper.go:195-197 | removed | none after repository search | removal search |
| 4 | NewCLPSolver | removed constructor | baseline clp_wrapper.go:205-209 | removed | none after repository search | removal search |
| 5 | (*LPSolverWrapper).Solve | method | clp_wrapper.go:207 | modified | optimization_processor.go:477, wrapper tests | focused and packaged tests |
| 6 | (*LPSolverWrapper).CheckVersion | method | clp_wrapper.go:298 | modified | worker readiness, Task 206 fixture, wrapper tests | pinned-version tests |
| 7 | (*LPSolverWrapper).StartupCheck | removed method | baseline clp_wrapper.go:343-347 | removed | worker/app callers replaced | removal search and caller tests |
| 8 | (*LPSolverWrapper).validatedConfig | method | clp_wrapper.go:332 | modified | Solve, CheckVersion | wrapper boundary tests |
| 9 | cleanupSolverDirectory | function | clp_wrapper.go:372 | added | deferred from Solve | cleanup failure test |
| 10 | serializeLP | function | clp_wrapper.go:389 | modified | Solve, serializer tests | deterministic/validation tests |
| 11 | serializeLPWithLimit | function | clp_wrapper.go:395 | added | serializeLP, limit tests | exact/over-limit tests |
| 12 | boundedModelWriter | behavioral type | clp_wrapper.go:484 | added | serializer and writeExpression | limit tests |
| 13 | (*boundedModelWriter).WriteString | method | clp_wrapper.go:492 | added | serializer | limit tests |
| 14 | (*boundedModelWriter).AppendByte | method | clp_wrapper.go:502 | added | serializer and expression writer | limit tests |
| 15 | (*boundedModelWriter).Bytes | method | clp_wrapper.go:512 | added | serializer return path | serializer tests |
| 16 | (*boundedModelWriter).Err | method | clp_wrapper.go:516 | added | serializer limit checks | limit tests |
| 17 | canonicalConstraintName | function | clp_wrapper.go:520 | added | serializer constraint/bound rendering | deterministic serializer test |
| 18 | serializedModelLimitError | function | clp_wrapper.go:524 | added | serializer error path | limit tests |
| 19 | writeExpression | function | clp_wrapper.go:530 | modified | objective/constraint rendering | deterministic/limit tests |
| 20 | clpVersion | function | clp_wrapper.go:564 | modified | CheckVersion | version-token tests |
| 21 | parseCLPSolution | function | clp_wrapper.go:576 | modified | Solve | parser and wrapper tests |
| 22 | parseCLPSolutionLine | function | clp_wrapper.go:623 | added | parseCLPSolution | exact grammar table |
| 23 | exactCLPStatus | function | clp_wrapper.go:683 | added | parseCLPSolutionLine | exact grammar table |
| 24 | TestLPSolverWrapperMapsOptimalOutputAndUsesGeneratedArguments | test | clp_wrapper_test.go:19 | modified | exercises Solve boundary | generated-name/cleanup assertions |
| 25 | TestLPSolverWrapperUsesSolutionFileAsAuthoritativeResult | test | clp_wrapper_test.go:66 | added | exercises precedence/fallback | three subtests |
| 26 | TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult | test | clp_wrapper_test.go:117 | added | exercises deferred cleanup | success/error subtests |
| 27 | TestLPSolverWrapperTrustedRunnerContractDoesNotLeakDeadlineGoroutine | test | clp_wrapper_test.go:149 | added | exercises private runner contract | non-cooperative runner |
| 28 | TestLPSolverWrapperMapsTerminalStatuses | test | clp_wrapper_test.go:168 | modified | exercises status mapping | infeasible/unbounded |
| 29 | TestLPSolverWrapperMapsCanceledTimeoutMalformedMissingAndNonZero | test | clp_wrapper_test.go:191 | modified | exercises error classification | cancellation/timeout/error cases |
| 30 | TestLPSolverWrapperBoundsAndSanitizesOutput | test | clp_wrapper_test.go:275 | modified | exercises bounded diagnostics | oversized stderr |
| 31 | TestLPSolverWrapperChecksPinnedVersion | test | clp_wrapper_test.go:290 | modified | exercises CheckVersion | accepted/mismatch/malformed |
| 32 | TestCLPVersionUsesFirstExactPunctuatedToken | test | clp_wrapper_test.go:323 | added | exercises clpVersion | punctuation/multiple/malformed |
| 33 | TestSerializeLPIsDeterministicAndUsesCanonicalGeneratedNames | test | clp_wrapper_test.go:340 | added | exercises serializer/name map | repeated exact output |
| 34 | TestSerializeLPEnforcesLimitWhileWriting | test | clp_wrapper_test.go:372 | added | exercises bounded writer | exact/over/early limit |
| 35 | TestSerializeLPRejectsInvalidReferencesBoundsAndCoefficients | test | clp_wrapper_test.go:395 | added | exercises validation | bounds/finite/reference cases |
| 36 | TestParseCLPSolutionUsesExactHeadersAndRows | test | clp_wrapper_test.go:426 | added | exercises parser codecs | grammar/adversarial table |
| 37 | RunWithProcessor | function | worker.go:25 | modified | worker bootstrap | worker readiness tests and full suite |
| 38 | TestTask206TimeoutAndOwnershipGate | integration test | task206_backend_integration_test.go:172 | modified | real wrapper timeout path | Task 206 integration gate |
| 39 | task206TimeoutRunner | removed test helper | baseline task206_backend_integration_test.go:212-216 | removed | no consumers | replaced by real executable fixture |
| 40 | task206CLP | integration fixture helper | task206_backend_integration_test.go:212 | modified | Task 206 integration gate | readiness setup |

~~~yaml
inventory_source_count: 40
audited_symbol_count: 40
inventory_complete: true
generated_groupings:
  - "None; every changed executable/type/API unit is listed individually."
~~~
## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| commandRunner | Package-only trusted seam; runner must settle supplied context. | Normal and non-cooperative return are explicit. | No wrapper goroutine; Solve waits for trusted seam. | External callers cannot inject a process runner. | One direct call. | Minimal private function type. | Trusted non-cooperative test; production path inspected. | PASS |
| CLPConfig | Deployment fields remain public; test/cleanup seams private. | Defaults and invalid executable/version/timeout handled by validation. | Function fields copied into validated local config. | Executable is a single configured path, not shell text. | No request-sized state. | Public API no longer exposes runner injection. | Boundary tests and caller search. | PASS |
| CLPSolver | N/A — removed redundant alias. | No remaining use. | N/A. | Removes alternate public vocabulary. | N/A. | Canonical API is simpler. | Repository search has no match. | PASS |
| NewCLPSolver | N/A — removed forwarding constructor. | No remaining use. | N/A. | No new boundary. | N/A. | Callers use named constructor. | Repository search has no match. | PASS |
| (*LPSolverWrapper).Solve | Validates context/config/model, serializes generated names, maps authoritative result. | Handles mkdir/write, timeout/cancel, unavailable/non-zero/output-limit, missing/empty/malformed solution, terminal status. | Context timeout is deferred-cancelled; temp directory is deferred-cleaned; trusted runner is synchronous by contract. | No shell; fixed args and generated file/name IDs; diagnostics sanitized. | Model and process output bounded; solution file bounded. | Single orchestration path and stable sentinels. | Focused boundary matrix, real child, packaged CLP, full race. | PASS |
| (*LPSolverWrapper).CheckVersion | Requires exact expected semantic version before readiness. | Handles nil context, config, timeout/cancel, missing/non-zero, output-limit, malformed/mismatched version. | Timeout context cancelled; direct runner contract applies. | Version command has fixed -version arg and bounded output. | Captures bounded streams; token iterator. | Canonical readiness operation. | Pinned-version and packaged tests. | PASS |
| (*LPSolverWrapper).StartupCheck | N/A — removed forwarding method. | No remaining use. | N/A. | Removes ambiguous readiness surface. | N/A. | Callers use CheckVersion. | Repository search and worker/app tests. | PASS |
| (*LPSolverWrapper).validatedConfig | Enforces executable, exact version shape, and 0-to-30s timeout contract; installs trusted defaults. | Nil receiver and invalid values fail closed. | Returns an immutable-by-convention copy with all private hooks initialized. | Rejects whitespace/control/option-like executable strings. | No unbounded work. | Centralized defaults avoid duplicate checks. | Existing boundary tests plus real executable tests. | PASS |
| cleanupSolverDirectory | Best effort; never changes prior Solve return. | Reports only failure; success is silent. | Deferred execution occurs on every post-mkdir path. | Redacts job directory and sanitizes error text. | Diagnostic capped at 4 KiB. | Small single-purpose helper. | Success and primary-error cleanup-failure cases. | PASS |
| serializeLP | Uses the production 1 MiB ceiling. | Delegates all validation/rendering to bounded implementation. | No external state mutation. | Generated IDs isolate caller IDs. | Bounded writer. | Compatibility-preserving wrapper. | Deterministic and packaged solve tests. | PASS |
| serializeLPWithLimit | Validates variables/objective/constraints and emits supported LP syntax under limit. | Rejects empty/duplicate/unknown/non-finite/invalid bounds and output overflow. | Pure function over caller data; no I/O. | Caller IDs never become solver tokens. | Stops retaining bytes at configured limit; deterministic slice order. | Validation and rendering remain local and readable. | Exact/over/early-limit and invalid-input tests. | PASS |
| boundedModelWriter | Buffer cannot grow past its configured limit through its methods. | Sets stable output-limit error on overflow and remains failed closed. | No shared state; not concurrency-safe or required to be. | Only internal serialization data. | At most configured model bytes retained. | Minimal writer abstraction. | Limit tests exercise boundary. | PASS |
| (*boundedModelWriter).WriteString | Writes only when remaining capacity is sufficient. | Prior error and overflow are sticky. | No blocking/cancellation needed for bounded pure write. | No untrusted execution. | Avoids allocating an oversized input into buffer. | bytes.Buffer error intentionally ignored because it cannot fail. | Exact/over-limit serializer tests. | PASS |
| (*boundedModelWriter).AppendByte | Appends one byte only under capacity. | Sticky overflow on full/failed writer. | Pure local state. | No boundary crossing. | Constant work. | Clear name avoids io.ByteWriter signature confusion. | Serializer tests. | PASS |
| (*boundedModelWriter).Bytes | Returns serialized bytes after caller checks Err. | Empty/partial buffer is only used on error paths. | Slice aliases internal buffer but writer is local and returned only on success. | No external data exposure beyond intended model. | Zero-copy return. | Idiomatic buffer accessor. | Serializer exact-limit test. | PASS |
| (*boundedModelWriter).Err | Reports sticky serialization failure. | Nil until overflow; stable thereafter. | Local state only. | N/A. | Constant time. | Idiomatic error accessor. | Serializer limit test. | PASS |
| canonicalConstraintName | Maps zero-based index to generated c%06d name. | Deterministic for normal indices; indices come from bounded slice iteration. | Stateless. | Caller constraint names are not trusted solver identifiers. | One small format allocation. | Centralizes name reuse. | Exact serialized fixture. | PASS |
| serializedModelLimitError | Returns stable ErrSolverOutputLimit classification with safe diagnostic. | Only called after writer overflow. | No resources. | Diagnostic contains no model content. | Constant-size message. | Avoids duplicated error construction. | Exact/over-limit test uses errors.Is. | PASS |
| writeExpression | Emits variables in model slice order, omits zero coefficients, preserves signs. | Empty expression becomes 0; writer overflow is sticky. | Pure rendering. | Uses generated map, never raw IDs. | Linear in model variables; output bounded by writer. | No map iteration nondeterminism. | Signed/zero/repeated deterministic test. | PASS |
| clpVersion | Returns first punctuation-trimmed exact major.minor.patch, else unknown. | Handles empty/malformed/multiple/punctuated tokens. | No resources. | Parses bounded child output only. | FieldsSeq avoids token-slice allocation; input conversion is bounded. | Matches Go 1.25 iterator requirement. | Dedicated version table and readiness tests. | PASS |
| parseCLPSolution | Produces sparse known-variable solution and one terminal status. | Rejects missing/duplicate/conflicting status, unknown/duplicate rows, malformed values, and optimal no-variable output. | Pure over bounded byte payload; deterministic input order. | Only generated names map back to internal IDs. | bytes.Lines and fixed field storage bound parsing allocations per line. | Parser state is explicit and small. | Full adversarial parser table and packaged fixture. | PASS |
| parseCLPSolutionLine | Accepts exact status header and indexed CLP row, including ** marker. | Ignores unrelated diagnostics; rejects malformed exact headers/rows, non-finite/reduced-cost data, and material negatives; clamps tiny negatives. | Pure line parser. | Unknown solver names are surfaced to caller for rejection. | Fixed six-field storage; bounded input line from solution cap. | Exact grammar is isolated from aggregate parser. | Header/row/lookalike/negative table. | PASS |
| exactCLPStatus | Recognizes only Optimal, Infeasible, Unbounded. | Unknown token is not a status. | Stateless. | Prevents prefix misclassification. | Constant work. | Small explicit switch. | Exact-status table. | PASS |
| TestLPSolverWrapperMapsOptimalOutputAndUsesGeneratedArguments | Verifies optimal mapping and fixed process boundary. | Exercises solution write, stdout/stderr diagnostics, cleanup. | Temp directory is checked after solve. | Caller ID is asserted absent from model/args. | Small fixture. | Direct assertions. | Covers injection-shaped ID. | PASS |
| TestLPSolverWrapperUsesSolutionFileAsAuthoritativeResult | Defines file precedence and absent-file fallback. | Covers valid file with conflicting stdout, stdout-only, and present-empty file. | Each subtest gets isolated temp state. | Prevents human output from overriding machine result. | Small table. | Table-driven. | Directly covers criterion. | PASS |
| TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult | Cleanup telemetry is bounded and non-authoritative. | Covers success and non-zero primary result. | Deferred cleanup runs on both paths. | ANSI/path redaction asserted. | Oversized synthetic error tests cap. | Table-driven subtests. | Does not test a blocked observer; observer is package-private trusted seam and production logger receives bounded input. | PASS |
| TestLPSolverWrapperTrustedRunnerContractDoesNotLeakDeadlineGoroutine | Establishes chosen trusted seam semantics. | Non-cooperative runner returns after delay; settled context classifies timeout. | No background enforcement goroutine is introduced. | Seam cannot be supplied by external production packages. | Delay is 25ms and bounded. | Test documents contract. | Adversarial runner behavior covered. | PASS |
| TestLPSolverWrapperMapsTerminalStatuses | Maps exact terminal status files to sentinels. | Infeasible and unbounded paths. | Temp cleanup via wrapper. | No diagnostic leak asserted here; safe mapping covered elsewhere. | Small fixtures. | Table-driven. | Both non-optimal terminal states. | PASS |
| TestLPSolverWrapperMapsCanceledTimeoutMalformedMissingAndNonZero | Verifies stable error classification. | Cancellation, deadline, malformed, missing executable, non-zero. | Context cancellation and cleanup exercised. | Raw process errors are not exposed through classification. | Small injected seams. | Table-driven. | All requested error classes. | PASS |
| TestLPSolverWrapperBoundsAndSanitizesOutput | Enforces process-output cap and sanitized diagnostics. | Oversized ANSI-containing stderr. | No lingering process. | Control chars removed and length bounded. | Synthetic output exceeds 64 KiB. | Direct assertions. | Adversarial output covered. | PASS |
| TestLPSolverWrapperChecksPinnedVersion | Verifies exact configured version readiness. | Accepted, mismatch, malformed and fixed args. | Runner is local trusted seam. | Output bounded by wrapper. | Small fixtures. | Table-driven. | Readiness contract covered. | PASS |
| TestCLPVersionUsesFirstExactPunctuatedToken | Verifies codec token policy. | Punctuation, multiple tokens, extra version components, unknown. | Pure function. | Bounded input. | Iterator behavior exercised indirectly. | Compact table. | Malformed token cases covered. | PASS |
| TestSerializeLPIsDeterministicAndUsesCanonicalGeneratedNames | Verifies exact generated model bytes and reverse map. | Equality/ranged constraints, signs, zero coefficients. | Pure repeated serialization. | Raw caller IDs absent from output by generated naming. | Repeated small fixture. | Byte-for-byte golden assertion. | Determinism is direct. | PASS |
| TestSerializeLPEnforcesLimitWhileWriting | Verifies inclusive exact limit and early overflow classification. | Exact, one-byte-over, later-invalid constraint. | Pure bounded writer. | No unbounded serialization retained. | 100-constraint stress fixture. | Direct boundary assertions. | Early rejection is adversarially ordered. | PASS |
| TestSerializeLPRejectsInvalidReferencesBoundsAndCoefficients | Verifies fail-closed model validation. | Negative bounds, NaN coefficient, unknown objective reference. | Pure input copies isolate subtests. | Unknown IDs cannot reach solver syntax. | Small table. | Table-driven mutations. | Relevant malformed inputs covered. | PASS |
| TestParseCLPSolutionUsesExactHeadersAndRows | Verifies complete status/row contract and sparse mapping. | Lookalikes, unknown/duplicate/conflict, malformed/non-finite/missing values, tiny/material negatives, zero omission. | Pure parser. | Unknown generated row rejected; diagnostics do not become rows. | Bounded table inputs. | Table-driven. | Directly covers all parser clauses. | PASS |
| RunWithProcessor | Worker readiness must use canonical CheckVersion before queue bootstrap. | Readiness errors return before processor run; existing nil/dependency paths preserved. | Context propagates to Redis and version check; heartbeat/queue lifecycle unchanged. | Solver remains worker-only. | One readiness process check. | Canonical call removes alias. | Full worker tests and integration callers. | PASS |
| TestTask206TimeoutAndOwnershipGate | Integration timeout gate must use production child-process boundary. | Real executable fixture sleeps and wrapper maps timeout to safe failure. | Temp fixture and process cancellation are real. | No exported runner seam remains. | 10ms timeout avoids 30s test wait. | Fixture is closer to production path. | Full backend app tests passed. Optional stale comment noted in Findings. | PASS |
| task206TimeoutRunner | N/A — removed obsolete exported-runner-dependent fixture helper. | No remaining call. | N/A. | Removes test-only dependency on public process injection. | N/A. | Real executable fixture is simpler boundary evidence. | Removal verified by diff/search. | PASS |
| task206CLP | Task 206 setup must use canonical readiness. | Missing executable still fails test setup; version mismatch fails clearly. | Uses direct wrapper readiness. | Configured executable remains separate arg/path. | One bounded version check. | Caller aligns with design API. | Full app tests passed. | PASS |

Mandatory audit outcome: boundary/malformed inputs, return/error paths, resource cleanup, cancellation, concurrency implications, trusted-data boundaries, bounded loops/output/memory, API minimality, and adversarial tests were considered for every inventory entry. No changed symbol is unaudited.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | backend/internal/app/task206_backend_integration_test.go:169-171 | TestTask206TimeoutAndOwnershipGate | The modified comment still says “the injected runner shortens only this integration fixture's wait,” but the test now creates a real executable fixture and uses the production runner. | Current code at lines 199-203 writes an executable shell fixture and omits CLPConfig.runner; the stale sentence can mislead future reviewers about boundary coverage. | Update the comment to describe the real executable fixture. Non-blocking documentation-only cleanup; no effect on behavior or acceptance. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| git rev-parse HEAD and baseline resolution | repository root | 0 | PASS — both resolve to a4e31367485b03269e90b5607f2057c9568bb5b1 | command output recorded during review |
| git diff --name-status and git diff --unified=0 against baseline for four task paths | repository root | 0 | PASS — exactly four tracked task-owned changed paths | preparation report cross-check |
| rg '\b(CLPSolver\|NewCLPSolver\|StartupCheck)\b' backend --glob '*.go' | repository root | 1 expected | PASS — no matches | clean repository search |
| gofmt -d on four task-owned Go files | repository root | 0 | PASS — no formatting delta | stdout empty |
| git diff --check a4e31367485b03269e90b5607f2057c9568bb5b1 -- four task paths | repository root | 0 | PASS — no whitespace errors | stdout empty |
| python3 scripts/validate-phase07-go-doc.py | repository root | 0 | PASS | Phase 07 exported Go Doc validation passed. |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS | Traceability validation passed. |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS — 237 sequential tasks and dependencies | task list not edited |
| GOCACHE=... GOMODCACHE=... go test ./internal/optimization ./internal/worker ./cmd/worker -count=1 -v | backend/ | 0 | PASS — focused wrapper, worker, and packaged CLP tests | verbose test output; packaged CLP passed |
| GOCACHE=... GOMODCACHE=... go vet ./... | backend/ | 0 | PASS | stdout empty |
| GOCACHE=... GOMODCACHE=... go test ./... -count=1 | backend/ | 0 | PASS — all backend packages | fresh full test run |
| GOCACHE=... GOMODCACHE=... go test -race ./... -count=1 | backend/ | 0 | PASS — all backend packages | fresh full race run |
| GOCACHE=... GOMODCACHE=... go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | backend/ | 0 | PASS — no vulnerabilities found in called code | govulncheck output |
| go test ./internal/optimization -coverprofile=/tmp/task-217-optimization.coverage.out -count=1 and go tool cover -func=... | backend/ | 0 | PASS — 83.0% package line coverage; changed wrapper units were inspected for uncovered defensive branches | /tmp/task-217-optimization.coverage.out |
| go test ./internal/... -coverprofile=/tmp/task-217-backend.coverage.out -count=1 and go tool cover -func=... | backend/ | 0 | PASS — 87.8% aggregate internal coverage | /tmp/task-217-backend.coverage.out, /tmp/task-217-backend-coverage.log |

No required command was skipped. The aggregate coverage threshold is not a hard gate in this repository; the accepted Phase 07 exception is recorded in docs/implementation/04_OPEN.md and was remeasured above.

## 9. Files Inspected and Staleness Fingerprints

Hash algorithm: SHA-256. Hashes were taken after the final review commands; the evidence-document write does not alter these files.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| backend/internal/optimization/clp_wrapper.go | Task production boundary, serializer, codecs, cleanup | No blocking/important finding | SHA-256 | da9ae4b9862c67ab18848a2829b763034d54d684add4eefbe08dae868b8451c7 |
| backend/internal/optimization/clp_wrapper_test.go | Task-focused unit/adversarial tests | Coverage supports all criteria | SHA-256 | ad201e23848593fe5f783dda419b7ffc5ea9d969f9f98e6152422e535e18664f |
| backend/internal/worker/worker.go | Readiness caller and worker-only boundary | Canonical CheckVersion caller | SHA-256 | 54d011aaa192b5050c10590f4b51b023897dacb635c1f9dd4710e81d5213cda8 |
| backend/internal/app/task206_backend_integration_test.go | Real child timeout and readiness integration caller | Optional stale comment only | SHA-256 | 2b9af6b5c61a82ca254637d3620fe60bc80193116850cb5f3790f344737d1a26 |
| backend/internal/optimization/validator.go | Downstream epsilon and safe solver-error projection | SolutionValidationEpsilon=1e-9; no contradiction | SHA-256 | 3e2ae3a00366524a748c0e6b801954831af4a987e38c9192a71010defc673ba2 |
| backend/internal/optimization/diversity.go | AlternativeSolveFunc and solver call contract | Caller contract unchanged and compatible | SHA-256 | 4dd3a5d35abab682b21a7b4a4614a5cd08f4837e3312ef4a762b6d89218b5e66 |
| backend/internal/optimization/objective.go | Objective fields consumed by serializer | No task-217 regression; later objective work remains separate | SHA-256 | 2ad8b1011c43875b39c3bc428c4d88a7759aa2f6553ae294be1bff6ec29fbbb4 |
| backend/internal/worker/optimization_processor.go | Concrete p.solver.Solve caller and error projection | Context and safe status flow remain compatible | SHA-256 | 5e77caad8f6cb7cdf5fea9a7bd054dedde223ea4b190792cc8f8f60fd4e96be |
| backend/cmd/worker/main.go | Production wrapper construction | Worker composes canonical wrapper; readiness remains in worker | SHA-256 | 15f97d1b320903f1b44ab93a3c0230912efedfe65fca9e53640262eff5767f3a |
| docs/implementation/preparation/task-217-preparation.md | Preparation scope and claimed evidence | Current and consistent with diff | SHA-256 | ead0e0c6498e07ad1845a968e95fe2fc6ed13988c9a24d3cbee3437549e7b0e1 |
| docs/design/DESIGN-004.md | Source-of-truth LPSolverWrapper responsibilities/interfaces | No contradiction | SHA-256 | 47ab62398f77413f295ac9e0b56d1d9cf92000f7f3edeed3355cf5c56a550410 |
| docs/architecture/ARCH-004.md | Architecture boundary/deadline/codec contract | No contradiction | SHA-256 | bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867 |
| docs/implementation/04_OPEN.md | Review action and accepted coverage exception | Task-217 actions are recorded implemented; coverage exception is accepted | SHA-256 | c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527 |
| docs/implementation/02_TASK_LIST.md | Status/dependency source of truth | Task 217 PREPARED; 212 PASSED | SHA-256 | edd8329f6746b9a2dff3e935d27d5f85d4611895e9fa5c597295b0c633986f0c |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "None; task-201 and task-212 evidence were consulted as historical references, and no prior task-217 review evidence existed."
~~~

## 10. Coverage and Exceptions

- [x] Required focused and aggregate coverage commands ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested branches relevant to changed symbols were inspected; remaining gaps are defensive/configuration/process-bootstrap branches covered by the accepted Phase 07 exception.
- [x] The exception used is the repository's existing accepted Phase 07 exception, not a new task-specific waiver.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-217-backend.coverage.out"
observed_line_coverage: "87.8% aggregate internal; 83.0% internal/optimization"
coverage_passed: true
~~~

Coverage finding: go test ./internal/... -coverprofile=... reports 87.8% aggregate internal line coverage and the focused optimization package reports 83.0%. docs/implementation/04_OPEN.md already accepts the Phase 07 below-100% defensive/dependency/process-bootstrap coverage exception and specifically requires rerunning the aggregate gate after Phase 07 production changes. The rerun passed and current coverage is higher than the recorded 86.6% aggregate measurement; no new exception is needed.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced; production still uses a worker-owned child process and no shell.
- [x] No source-of-truth documentation was contradicted; DESIGN-004 and ARCH-004 agree with the implementation.
- [x] No generated/cache/build/temporary artifact was unintentionally added by the task or review.
- [x] Public API additions are necessary and used; the task removes redundant public aliases and keeps only the documented wrapper API.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged. The trusted test seam is intentionally package-private and synchronous; production uses exec.CommandContext.

Findings: no blocking or important regression. The only finding is the optional stale comment in the modified Task 206 test; it does not alter the tested behavior or the production boundary.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those conditions are met.

Before accepting the decision, run:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py \
  docs/implementation/reviews/task-217-review.md
~~~

~~~yaml
decision: "PASSED"
reason: "Task 217 is PREPARED, dependency 212 is PASSED, all 13 criteria and all 40 changed units pass independent audit, and no blocking or important finding remains."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for acceptance; optionally correct the stale Task 206 timeout-fixture comment before the next review."
~~~

## 13. Repair Context

Not applicable because the decision is PASSED. No production repair, task-list edit, or status transition was performed by this review.

