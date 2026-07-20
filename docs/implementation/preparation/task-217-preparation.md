# Task 217 Preparation Evidence

## Assignment, Baseline, and References

- Assigned task: **217 — Phase 07.01 CLP Process Boundary and Codec Hardening**.
- Task source: row 217 of `docs/implementation/02_TASK_LIST.md`; dependency 212 was `PASSED` before implementation.
- Design source: `docs/design/DESIGN-004.md`, static aspect `LPSolverWrapper`, especially the pure-Go worker-only child-process boundary, 30-second ceiling, generated solver names, bounded output, exact startup version, private temporary files, and cleanup responsibilities.
- Review-action source: `docs/implementation/04_OPEN.md` actions at lines 285-289: canonical API removal, trusted command-runner boundary and cleanup reporting, authoritative solution file, bounded serialization, Go 1.25 iterators, and exact CLP codecs.
- Historical reference: `docs/implementation/reviews/task-201-review.md` for the original CLP boundary and packaged solver fixture; `docs/implementation/reviews/task-212-review.md` for the passed Phase 07 baseline gate.
- Baseline: `a4e31367485b03269e90b5607f2057c9568bb5b1` (`git rev-parse HEAD`).
- Baseline confidence: **high**. The shared worktree was dirty before Task 217. Existing changes, including `docs/implementation/02_TASK_LIST.md`, were preserved. No task status was edited.
- Evidence-path correction: this path previously contained mistakenly named Task 216 evidence. The authoritative Task 216 record remains `task-216-preparation.md`; this file now records its assigned Task 217 scope.

## Outcome

`LPSolverWrapper` now has one canonical readiness vocabulary and an enforceable production process boundary. `CLPSolver`, `NewCLPSolver`, and `StartupCheck` were removed; worker and integration readiness call `CheckVersion`. The command runner and cleanup hooks are package-private trusted test seams, while production always defaults to `exec.CommandContext` and `os.RemoveAll`.

The machine-readable solution file is authoritative whenever it exists. Human stdout is retained only as a deliberate compatibility fallback when the solution file is absent, and is never concatenated with a valid solution. Cleanup failure is reported once through a bounded, sanitized, job-directory-redacted observation without replacing the solve result or primary error.

LP output is rejected during bounded writing instead of after an oversized buffer has already been built. Rendering remains deterministic, uses generated names from the canonical map for constraints and bounds, and preserves equality/ranged constraints and signed/zero coefficient behavior.

Version and solution decoding use Go 1.25 iterator APIs (`strings.FieldsSeq`, `bytes.Lines`, and `bytes.FieldsSeq`). The solution parser accepts only exact pinned-CLP terminal headers and indexed row shapes (including CLP's `**` row marker), rejects unknown/duplicate/conflicting machine output, ignores unrelated diagnostics, clamps negative residuals no smaller than `SolutionValidationEpsilon` to zero, and rejects materially negative quantities.

## Exact Task-Owned Changed Paths

Modified:

- `backend/internal/optimization/clp_wrapper.go`
- `backend/internal/optimization/clp_wrapper_test.go`
- `backend/internal/worker/worker.go`
- `backend/internal/app/task206_backend_integration_test.go`
- `docs/implementation/preparation/task-217-preparation.md`

No other path is owned by Task 217. In particular, Task 217 did not edit `docs/implementation/02_TASK_LIST.md`, any task status, `docs/design/DESIGN-004.md`, constraint construction, optimization eligibility, domain request types, or any Task 218-or-later surface.

`task206_backend_integration_test.go` is in scope only to replace the removed public runner seam with a real `exec.CommandContext` timeout fixture and to use canonical `CheckVersion`; its Task 206 behavior was not otherwise changed.

## Added, Modified, and Removed Symbols

Production executable symbols added:

- `optimization.cleanupSolverDirectory`
- `optimization.serializeLPWithLimit`
- `optimization.(*boundedModelWriter).WriteString`
- `optimization.(*boundedModelWriter).AppendByte`
- `optimization.(*boundedModelWriter).Bytes`
- `optimization.(*boundedModelWriter).Err`
- `optimization.canonicalConstraintName`
- `optimization.serializedModelLimitError`
- `optimization.parseCLPSolutionLine`
- `optimization.exactCLPStatus`

Production executable symbols materially modified:

- `optimization.(*LPSolverWrapper).Solve`
- `optimization.(*LPSolverWrapper).CheckVersion`
- `optimization.(*LPSolverWrapper).validatedConfig`
- `optimization.serializeLP`
- `optimization.writeExpression`
- `optimization.clpVersion`
- `optimization.parseCLPSolution`
- `worker.RunWithProcessor`

Production types/contracts added or modified:

- `optimization.commandRunner` — replaces exported `CommandRunner` with a package-only trusted test seam.
- `optimization.CLPConfig` — removes the exported runner and adds package-private runner, cleanup, and cleanup-observation seams.
- `optimization.boundedModelWriter` — owns the serialization byte ceiling.

Production symbols removed:

- `optimization.CLPSolver`
- `optimization.NewCLPSolver`
- `optimization.(*LPSolverWrapper).StartupCheck`
- `optimization.CommandRunner`

Test executable symbols added:

- `optimization.TestLPSolverWrapperUsesSolutionFileAsAuthoritativeResult`
- `optimization.TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult`
- `optimization.TestLPSolverWrapperTrustedRunnerContractDoesNotLeakDeadlineGoroutine`
- `optimization.TestCLPVersionUsesFirstExactPunctuatedToken`
- `optimization.TestSerializeLPIsDeterministicAndUsesCanonicalGeneratedNames`
- `optimization.TestSerializeLPEnforcesLimitWhileWriting`
- `optimization.TestSerializeLPRejectsInvalidReferencesBoundsAndCoefficients`
- `optimization.TestParseCLPSolutionUsesExactHeadersAndRows`

Test executable symbols materially modified:

- `optimization.TestLPSolverWrapperMapsOptimalOutputAndUsesGeneratedArguments`
- `optimization.TestLPSolverWrapperMapsTerminalStatuses`
- `optimization.TestLPSolverWrapperMapsCanceledTimeoutMalformedMissingAndNonZero`
- `optimization.TestLPSolverWrapperBoundsAndSanitizesOutput`
- `optimization.TestLPSolverWrapperChecksPinnedVersion`
- `app.TestTask206TimeoutAndOwnershipGate`
- `app.task206CLP`

Test executable symbol removed:

- `app.task206TimeoutRunner`

## Criteria Coverage

- **Single canonical API:** repository Go-source search returns no `CLPSolver`, `NewCLPSolver`, or `StartupCheck`; worker startup and Task 206 readiness use `CheckVersion`.
- **Approved deadline contract:** runner injection is no longer available outside package tests. Production uses the real `exec.CommandContext` path. A regression test records the trusted seam's behavior when a package test runner ignores cancellation: `Solve` waits for that runner to return, then classifies the settled deadline, without spawning a leak-prone enforcement goroutine.
- **Timeout child termination and normal cleanup:** the real sleeping-child fixture still terminates promptly and leaves the temporary root empty. Task 206 now also uses a real executable fixture instead of exported runner injection.
- **Cleanup observability and primary-result preservation:** injected cleanup failure tests cover successful and failed solves; observation is bounded, sanitized, and path-redacted, while success or `ErrSolverNonZero` remains authoritative.
- **Authoritative machine result:** tests prove valid solution-file output wins over conflicting/duplicate stdout, absent-file stdout fallback remains deliberate, and a present empty file does not silently fall back.
- **Bounded deterministic serialization:** exact-limit output succeeds, one-byte-over fails as `ErrSolverOutputLimit`, and a small ceiling rejects during earlier writes before a later invalid constraint is inspected. Repeated serialization proves stable generated variable/constraint names, canonical bound-name reuse, equality/ranged grammar, signed coefficients, and zero omission. Invalid bounds, non-finite coefficients, and unknown objective references remain rejected.
- **Exact version codec:** `strings.FieldsSeq` preserves punctuation trimming, first-valid-token selection, exact `major.minor.patch`, mismatch handling, and `unknown` fallback.
- **Exact solution codec:** table tests cover exact optimal/infeasible/unbounded headers, prefix/hyphen lookalikes, ordinary and `**` rows, ignored diagnostics, unknown/duplicate variables, duplicate/conflicting statuses, missing/malformed/non-finite quantities, tiny-negative clamping, material-negative rejection, sparse zero omission, missing status, and missing optimal rows.
- **Bounded diagnostics and terminal mapping:** existing output-limit/sanitization and infeasible/unbounded/error-classification tests remain green.
- **Packaged CLP:** the local pinned executable passes version check and solves the generated fixture.
- **Go quality gates:** Go Doc validation, formatting, diff whitespace, focused tests, full vet, and full backend race tests pass.

## Commands and Results

- `git rev-parse HEAD` → `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- `gofmt -w backend/internal/optimization/clp_wrapper.go backend/internal/optimization/clp_wrapper_test.go backend/internal/worker/worker.go backend/internal/app/task206_backend_integration_test.go` → PASS; final files formatted.
- `python3 scripts/validate-phase07-go-doc.py` → PASS: `Phase 07 exported Go Doc validation passed.`
- `python3 scripts/validate-traceability.py` → PASS. Its first run identified missing adjacent DESIGN-004/doc comments on the newly added private serialization and codec helpers; those comments were added and the final run passed.
- `python3 scripts/validate-task-list.py` → PASS: 237 sequential tasks with ordered dependencies; no status was edited.
- `rg -n '\b(CLPSolver|NewCLPSolver|StartupCheck)\b' backend --glob '*.go'` → PASS: no matches.
- `git diff --check` → PASS.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./cmd/worker -count=1` → PASS.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` → PASS for every backend package.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` → PASS. An earlier focused vet run found the private writer's `WriteByte` name implied the standard `io.ByteWriter` signature; it was renamed to `AppendByte`, and the final full vet passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` → PASS for every backend package.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run TestLPSolverWrapperRunsPackagedExecutableWhenAvailable -count=1 -v` → PASS against local CLP `1.17.11`.

## Residual Risks and Deliberate Boundaries

- Package-internal tests can still supply a non-cooperative `commandRunner`; this is an explicit trusted seam. It is inaccessible to production callers, and production deadline enforcement remains `exec.CommandContext` without a wrapper goroutine.
- Temporary-directory cleanup is best effort. If `os.RemoveAll` fails, the primary result is preserved and one bounded/redacted diagnostic is emitted; residue may remain until host/container cleanup.
- Stdout fallback remains for compatibility only when no solution file exists. Because it must satisfy the same exact machine grammar, a future CLP output-format change requires an intentional codec update.
- The parser and packaged fixture target pinned CLP `1.17.11`; accepting additional status or row grammars is intentionally out of scope.
- Task 218 and later constraint/model changes were not implemented. Existing duplicate-bound and alternative-generation semantics remain unchanged for their assigned tasks.

No task-list status was changed.
