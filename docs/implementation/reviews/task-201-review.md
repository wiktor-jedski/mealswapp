# Task 201 Review — CLP Child-Process Solver Wrapper

**Decision: PASSED**

**Scope:** Task 201 only (`DESIGN-004: LPSolverWrapper`). Task 201 is `PREPARED`; dependencies 198 and 199 are `PASSED`. This review changed only this review document.

## Findings

No blocking or non-blocking findings.

The implementation keeps CLP behind an injectable pure-Go child-process boundary owned by the worker. It validates and serializes internal LP data using generated solver identifiers, invokes CLP without a shell, enforces the 30-second maximum deadline, bounds all solver-controlled output, sanitizes retained diagnostics, and unconditionally removes private per-job directories. The production worker performs the pinned executable/version startup check, while the Fiber API has no CLP invocation path.

## Acceptance criteria

| Criterion | Result | Evidence |
|---|---|---|
| Injectable child-process wrapper using `exec.CommandContext` | PASS | `CLPConfig.Runner` supplies the test boundary; the production `runOSCommand` uses `exec.CommandContext` with separate executable and argument values. |
| Validated model serialization to supported LP input | PASS | `serializeLP` validates variables, bounds, finite coefficients, objective coverage, and constraints before producing deterministic CLP LP syntax. The real packaged-CLP fixture solves the generated model successfully. |
| Optimal result mapping | PASS | `TestLPSolverWrapperMapsOptimalOutputAndUsesGeneratedArguments` verifies an optimal solution is parsed and generated variable names are mapped back to the original meal ID. |
| Infeasible and unbounded mapping | PASS | `TestLPSolverWrapperMapsTerminalStatuses` verifies both terminal statuses map to their stable sentinel errors. |
| Cancellation, malformed output, missing executable, and non-zero exit mapping | PASS | `TestLPSolverWrapperMapsCanceledTimeoutMalformedMissingAndNonZero` covers each required failure class and verifies `errors.Is` behavior. |
| Hard 30-second maximum deadline terminates the child and cleans temporary state | PASS | Configuration rejects timeouts above `SolverDeadline`; `TestLPSolverWrapperTerminatesRealChildAndCleansDeadlineDirectory` starts a real sleeping child, verifies prompt timeout termination, and confirms the private job directory is empty afterward. Cleanup is deferred immediately after `MkdirTemp`. |
| Filenames and arguments cannot be influenced by user input | PASS | Job paths and solver names are generated internally; CLP arguments are fixed. The optimal-path test uses `meal;caller-id` and verifies it appears in neither subprocess arguments nor serialized solver identifiers. Executable configuration rejects whitespace, control characters, and option-like values. |
| Bounded and sanitized stdout/stderr/solution handling | PASS | `limitedBuffer` caps each process stream, `readBoundedFile` caps the solution file, and diagnostics remove control characters, redact the job directory, and cap retained text. `TestLPSolverWrapperBoundsAndSanitizesOutput` verifies overflow classification and escape removal. |
| Startup fails clearly when executable/version is unavailable | PASS | Worker startup calls `StartupCheck` after Redis readiness. `CheckVersion` uses bounded `clp -version` execution and exact `1.17.6` matching; tests cover accepted, mismatched, and malformed versions, while missing-executable mapping is covered by the solve boundary. |
| Focused integration fixture exercises packaged CLP | PASS | `TestLPSolverWrapperRunsPackagedExecutableWhenAvailable` checks the real executable/version and solves a fixture. The independent Docker build ran this test with `/usr/bin/clp` and passed. |
| Pinned production worker image and no CGO | PASS | `backend/Dockerfile.worker` pins `coinor-clp=1.17.6-3`, verifies reported version `1.17.6`, builds the worker with `CGO_ENABLED=0`, runs the CLP integration tests in the build stage, and uses a non-root runtime user. The independent image build completed successfully. |
| Solver remains outside the Fiber API process | PASS | CLP construction/readiness appears in `backend/internal/worker/worker.go`; no API application/controller path imports or invokes the optimization wrapper. |
| Architecture, design, and technology documentation | PASS | `DESIGN-004`, `ARCH-004`, consolidated architecture files, and `01_TECH_STACK.md` consistently describe the pinned native CLP child process, pure-Go boundary, worker ownership, deadline, output controls, and absence of CGO/API execution. |
| Traceability and task integrity | PASS | Design comments identify `DESIGN-004 LPSolverWrapper`; task-list and traceability validators pass. Task 202 remains `OPEN` and was not edited by this review. |

## Verification run

- `go test -count=1 ./internal/optimization ./internal/worker ./internal/config` — PASS
- `go test -race -count=1 ./internal/optimization` — PASS
- `go vet ./internal/optimization ./internal/worker ./internal/config` — PASS
- `CGO_ENABLED=0 go build ./cmd/worker` — PASS
- `govulncheck@v1.3.0 ./internal/optimization ./internal/worker ./internal/config` — PASS (`No vulnerabilities found`)
- `docker build --file backend/Dockerfile.worker --tag mealswapp-worker:task-201-review .` — PASS, including pinned CLP installation, `CGO_ENABLED=0` worker build, exact runtime version check, and real CLP integration test
- `python3 scripts/validate-task-list.py` — PASS
- `python3 scripts/validate-traceability.py` — PASS

Task 201 satisfies every reviewed acceptance criterion and is recommended **PASSED**.
