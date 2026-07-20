# Review Evidence: Task 221 — Optimization Publication Vocabulary and Projection

~~~yaml
task_id: 221
component: "Phase 07.01 Optimization Publication Vocabulary and Projection"
static_aspect: "DESIGN-004: SolutionValidator"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T13:17:02Z"
review_agent: "Codex independent owner review"
evidence_file: "docs/implementation/reviews/task-221-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1 plus task-221-preparation.md pre-task hashes"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill Go and TypeScript guides"
repair_context_required: true
~~~

## 1. Task Source

**Description:** Phase 07.01: close and validate the optimization terminal-failure vocabulary across worker persistence, HTTP projection, telemetry, OpenAPI, generated clients, and safe messages; define cancellation versus shutdown/retry semantics; and implement a documented authoritative `similarityScore` calculation or remove that field consistently, including persisted-result compatibility.

**Depends On:** Task 220 (`PREPARED`).

**Testing Coverage Exceptions:** The preparation report records the existing phase/backend coverage exception and an aggregate backend coverage result below the repository's 100% phase goal. No new exception is accepted for the two findings in this review.

**Verification Criteria:**

1. Arbitrary or empty failure codes cannot be constructed, persisted, decoded, or returned.
2. Every retained code has a tested producer and consumer.
3. Wrapped validation, infeasible, timeout, cancellation, queue, expiry, and unknown failures follow the documented terminal/retry policy without leaking diagnostics.
4. OpenAPI/generated types and frontend messages use the same enum.
5. `similarityScore` has bounded nontrivial fixtures and documented meaning/rounding, or repository-wide search proves its removal with Redis compatibility addressed.
6. Focused worker/controller/client/browser tests and contract drift checks pass.

This is a fresh review of the repaired current tree. It verifies explicit numeric zero, rejects omitted/null/non-number persisted scores, and normalizes invalid non-nil pre-classified failures.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy enough for scoped attribution.
- [x] `code-review-skill` was invoked exactly once and its relevant Go, TypeScript, security, and error-handling guides were read.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list changes.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: the preparation report's exact pre-task SHA-256 manifest and fixed reference `a4e31367485b03269e90b5607f2057c9568bb5b1` established scope. The latest repaired contents were independently re-read, focused raw-payload/error regressions were executed, and current hashes were captured after all verification. The cumulative worktree contains Task 222 overlays; those were excluded by symbol and ownership, not reverted.

Commands used to reconstruct the diff and audit the current surface:

```bash
git status --short --branch
git log --oneline --decorate -12
git diff --unified=20 a4e31367485b03269e90b5607f2057c9568bb5b1 -- <Task-221-owned paths>
rg -n 'OptimizationFailureCode|FailureCodeOf|safeOptimizationFailure|UnmarshalJSON|similarityScore|PublishCompleted|validateOptimizationJob|optimizationJobData'
nl -ba <reviewed-file> | sed -n '<relevant range>p'
sha256sum <every reviewed implementation file>
```

Pre-existing dirty-worktree changes and exclusions:

- Task 213–220 changes were treated as dependencies/context except for the Task 221 symbols that extend their boundaries.
- Task 222 changes are present in the same worktree, notably submission idempotency, durable acknowledgement, admission, request hashing, queue publication, and response-matrix changes in shared controller/OpenAPI files. They were preserved and excluded from Task 221 attribution.
- The Task 222 preparation remains present and was inspected for ownership boundaries. Task 222 remains `OPEN`; Task 221's status row remains `PREPARED`; no status or production file was edited during this review.
- The prior Task 221 review was used only for findings and staleness comparison. Its pre-repair hashes and conclusion were not reused as current evidence.
- The current baseline is `MEDIUM` confidence because the worktree has overlapping uncommitted tasks, but the preparation manifest identifies the repaired Task 221 symbols and current source was independently re-read.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/internal/optimization/validator.go` | Task 221 preparation and repair | Medium | closed failure code, score predicate, projection, classifier |
| `backend/internal/optimization/validator_test.go` | Task 221 repair tests | High | score-shape and typed-nil tests |
| `backend/internal/worker/optimization_processor.go` | Task 221 repair plus Task 222 shared work | Medium | Redis publication/decode, worker policy, telemetry |
| `backend/internal/worker/optimization_processor_deadline_test.go` | Task 221 repair tests | High | cancellation, safe failure, typed-nil policy |
| `backend/internal/worker/task221_publication_test.go` | Task 221 repair test | High | Redis publication/decode score fixtures |
| `backend/internal/httpapi/optimization_controller.go` | Task 221 repair plus Task 222 shared work | Medium | polling validation and projection |
| `backend/internal/httpapi/optimization_controller_test.go` | Task 221 repair plus Task 222 shared work | Medium | HTTP score/failure fixtures |
| `api/openapi.yaml` | Task 221 contract plus Task 222 responses | Medium | terminal enum and score schema |
| `scripts/generate-api-types.py` | Task 221 generator | High | generated terminal enum |
| `scripts/test_generate_api_types.py` | Task 221 drift test | Medium | enum/score contract drift |
| `frontend/src/lib/api/generated.ts` | generated from current OpenAPI | Medium | generated optimization types |
| `frontend/src/lib/api/optimization-client.ts` | Task 221 client decoder | Medium | client enum/score validation |
| `frontend/src/lib/api/optimization-client.test.ts` | Task 221 client tests | High | malformed code/score tests |
| `frontend/src/lib/stores/optimization.ts` | Task 221 UI policy | High | terminal/operation messages |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | Task 221 consumer audit | High | score rendering |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | Task 221 UI source test | High | score contract test |
| `frontend/tests/optimization-workflow.spec.ts` | Task 221 browser acceptance | High | completed/terminal UI flow |
| `backend/internal/app/task206_backend_integration_test.go` | compatibility call-site migration | High | failure-code string assertion |
| `docs/design/DESIGN-004.md` | Task 221 source-of-truth design | Medium | vocabulary, retry policy, score formula |

The two findings from the prior rejected artifact were re-reviewed against the repaired symbols and are closed below.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Arbitrary or empty failure codes cannot be constructed, persisted, decoded, or returned. | Value-object/JSON tests, worker persistence validation, HTTP guard, generated/client enum checks, and invalid-error adversarial path. | PASS | `OptimizationFailureCode` has an unexported value and only four constructors; JSON marshal/unmarshal, `OptimizationJobFailure.Valid`, Redis decode/publication, HTTP validation, generated types, client guards, and `TestSafeOptimizationFailureNormalizesInvalidExistingFailure` reject invalid/empty values. |
| 2 | Every retained code has a tested producer and consumer. | Worker producer/classifier, telemetry, Redis, HTTP, generated client, UI, and browser evidence. | PASS | All four retained codes have worker safe messages/telemetry, Redis/HTTP fixtures, generated union/client guards, typed UI mapping, and browser/client coverage. |
| 3 | Validation, infeasible, timeout, cancellation, queue, expiry, and unknown failures follow policy without diagnostics. | Design matrix, worker deadline/shutdown tests, queue/expiry client tests, telemetry, and malformed error paths. | PASS | Worker tests cover canonical validation/infeasible/timeout classification, shutdown cancellation without publication, queue/expiry client handling, typed-nil errors, and invalid non-nil `OptimizationFailure{}` normalization to bounded `worker_crash`; safe messages and telemetry use fixed values. |
| 4 | OpenAPI/generated types and frontend messages use the same enum. | OpenAPI, generator `--check`, generated union, drift test, client guard, and UI map. | PASS | The four retained terminal codes match across OpenAPI, generator output, generated TypeScript, client normalization, and typed UI messages; drift and full frontend tests pass. |
| 5 | `similarityScore` has bounded nontrivial fixtures and documented meaning/rounding, or is removed consistently. | Formula/design/OpenAPI inspection, validator fixtures, Redis publication/decode round-trip, HTTP projection tests, and malformed JSON decode cases. | PASS | DESIGN-004 and OpenAPI document quantity-weighted Jaccard and four-decimal rounding. Solver fixtures cover identical/partial/disjoint values; publication rejects out-of-range/off-grid values; presence-aware `DietAlternative.UnmarshalJSON` rejects omitted, null, and string scores while preserving explicit zero at Redis and HTTP raw-payload seams. |
| 6 | Focused worker/controller/client/browser tests and contract drift checks pass. | Focused/full tests, race, build, browser, generator, OpenAPI, traceability, task-list, vet, and security checks. | PASS | Focused and full Go tests, full Go race, repaired focused race tests, full frontend tests/build/browser verification, `go vet`, `govulncheck`, Redocly, traceability, task-list, generated API checks, coverage, and `git diff --check` passed. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `OptimizationFailureCode`, parser, codec, `Valid` | behavioral type | `backend/internal/optimization/validator.go:23-95` | bounded value object | worker envelope, JSON persistence, controller, client | validator JSON table |
| 2 | `OptimizationFailure.Error`, `Unwrap`, `FailureCodeOf` | error API | `backend/internal/optimization/validator.go:98-132` | typed-nil-safe extraction | worker classifier/telemetry | typed-nil validator test |
| 3 | `DietAlternative.UnmarshalJSON`, `ValidateDietAlternative`, `boundedProjectionNumber` | decode/result validator | `backend/internal/optimization/validator.go:146-207` | presence-aware persisted score and authoritative shape predicate | Redis and HTTP boundaries | raw score and malformed/score validator tests |
| 4 | `SolutionValidator.Validate` | projection method | `backend/internal/optimization/validator.go:227-317` | authoritative recalculation | generation pipeline | validator fixtures |
| 5 | `quantityWeightedSimilarity` | calculation | `backend/internal/optimization/validator.go:319-355` | authoritative score | `SolutionValidator.Validate` | identical/partial/disjoint fixtures |
| 6 | `GenerateValidatedAlternatives` | generation adapter | `backend/internal/optimization/validator.go:357-369` | shared validated pipeline | worker processor | partial/deadline/typed-nil tests |
| 7 | `safeOptimizationFailure` | error classifier | `backend/internal/optimization/validator.go:404-429` | bounded terminal mapping | diversity pipeline and worker | typed-nil and invalid non-nil tests |
| 8 | `RedisOptimizationJobStore.Load` | persistence method | `backend/internal/worker/optimization_processor.go:167-204` | decoded envelope validation | worker/controller | full backend and Redis fixture |
| 9 | `RedisOptimizationJobStore.PublishCompleted` | persistence method | `backend/internal/worker/optimization_processor.go:233-257` | result validation before write | worker terminal publication | Redis score fixture |
| 10 | `RedisOptimizationJobStore.PublishFailed` | persistence method | `backend/internal/worker/optimization_processor.go:259-285` | partial-result validation | worker failure publication | worker shape tests |
| 11 | `validateOptimizationJob`, `validateOptimizationAlternatives` | envelope validators | `backend/internal/worker/optimization_processor.go:642-696` | shared decoded-result gate | Redis Load and controller context | worker malformed/score tests |
| 12 | `handleProcessingError`, `publishFailure` | worker policy | `backend/internal/worker/optimization_processor.go:547-590` | cancellation/retry/terminal mapping | processor | deadline/shutdown tests |
| 13 | telemetry and safe-message maps | policy consumers | `backend/internal/worker/optimization_processor.go:601-740` | bounded vocabulary consumers | observability and persisted failure | all-code table |
| 14 | `OptimizationController.GetJob` | HTTP route | `backend/internal/httpapi/optimization_controller.go:337-371` | authenticated pre-projection validation | polling clients | HTTP score/failure tests |
| 15 | `validateOptimizationJobAlternatives`, `optimizationJobData` | HTTP projection | `backend/internal/httpapi/optimization_controller.go:374-398,570-605` | result guard and public DTO | `GetJob`, generated client | HTTP projection test |
| 16 | validator adversarial tests | unit tests | `backend/internal/optimization/validator_test.go:89-195` | score/error/codec regression coverage | validator symbols | direct |
| 17 | worker publication/deadline tests | integration/unit tests | `backend/internal/worker/task221_publication_test.go:15-138`, `optimization_processor_deadline_test.go:85-180` | Redis raw-score, invalid-error, cancellation, and typed-nil fixtures | worker boundaries | direct/race |
| 18 | HTTP projection tests | HTTP tests | `backend/internal/httpapi/optimization_controller_test.go:359-483` | invalid finite and raw persisted score fixtures | `GetJob` | direct |
| 19 | OpenAPI/generator/drift contract | schema/scripts | `api/openapi.yaml:1410-1455`, `scripts/generate-api-types.py:692-704` | enum/score source of truth | generated client | Redocly and drift test |
| 20 | generated types and optimization client decoder | generated/client artifact | `frontend/src/lib/api/generated.ts:526-541`, `optimization-client.ts:146-261` | generated enum plus runtime guards | store/component | full Bun tests |
| 21 | optimization store/UI/browser consumers | frontend behavior | `frontend/src/lib/stores/optimization.ts:278-290`, `OptimizationWorkflow.svelte:167-187`, browser spec | safe messages and score display | end-user workflow | source/unit/browser checks |
| 22 | `assertTask206Failure` | compatibility test caller | `backend/internal/app/task206_backend_integration_test.go:505-514` | string assertion migrated to value object | app integration test | full backend suite |

~~~yaml
inventory_source_count: 22
audited_symbol_count: 22
inventory_complete: true
generated_groupings:
  - "The OpenAPI source, generator, generated TypeScript, and runtime client are grouped only where they form one contract boundary; their individual files and consumers remain listed in the inventory."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `OptimizationFailureCode` and codec | Exactly four persisted terminal values; legacy strings remain compatible. | Rejects zero, empty, unknown, former operation, non-string, and null JSON values. | Pure value operations. | Prevents arbitrary nonzero persisted/public codes. | Constant-time switch and small JSON allocation. | Idiomatic bounded value object. | Valid/invalid table passes. | PASS |
| `OptimizationFailure` methods and `FailureCodeOf` | Public error text is code-only; cause stays internal. | Nil receivers and typed-nil extraction are safe; invalid non-nil values are not preserved as safe. | No resources; interface may hold typed nil. | Normal paths do not expose diagnostics. | Error-chain traversal only. | Minimal API; `Code.Valid()` gates preservation. | Typed-nil and non-nil invalid-error tests pass. | PASS |
| `DietAlternative.UnmarshalJSON`, `ValidateDietAlternative`, and `boundedProjectionNumber` | Decoded alternatives require a present numeric score; valid zero remains distinct from missing/null. | Rejects absent/null/string/invalid finite/off-grid values and malformed shape; accepts explicit zero and rounded values. | Pure decode/predicate operations. | Prevents malformed persisted data reaching Redis/HTTP projections. | RawMessage plus one float pointer; linear bounded meal validation. | Small shared boundary with no caller score trust. | Raw Redis and HTTP cases cover omitted/null/string/zero; struct score adversarial cases pass. | PASS |
| `SolutionValidator.Validate` | Recomputes macros/calories and derives score from immutable repository data. | Invalid requests, solver quantities, exclusions, macros, and scores become safe validation failures. | Read-only snapshot; no external I/O. | Solver/client totals are not trusted. | Bounded projection and deterministic ordering. | One authoritative calculation path. | Identical, partial, disjoint, and partial-result tests pass. | PASS |
| `quantityWeightedSimilarity` | Quantity-weighted Jaccard over canonical g/ml quantities, rounded to four decimals. | Missing original meals, invalid quantities, zero union, nonfinite, and out-of-range calculations fail. | Pure calculation. | Uses server meal snapshot rather than client score. | O(union), bounded by request/model. | Formula matches design/OpenAPI. | Nontrivial fixtures pass. | PASS |
| `GenerateValidatedAlternatives` | Only validated projections leave the generation boundary; valid partial results survive later failure. | Nil context, malformed snapshots, duplicates, solver errors, and cancellation map safely. | Context propagated to bounded attempts; no goroutine leak. | Solver output crosses one validation boundary. | Max three results and capped attempts. | Avoids duplicate projection. | Full optimization/race and typed-nil solve seam pass. | PASS |
| `safeOptimizationFailure` | Unknown internal failures map to bounded `worker_crash`; valid pre-classified codes remain stable. | Typed-nil failure/solver/repository values are guarded; non-nil invalid `OptimizationFailure` is normalized. | No resources. | Diagnostics remain in the private cause and never become public text. | Error-chain traversal only. | `existing.Code.Valid()` is the narrow preservation gate. | Direct invalid-value and worker solver-seam tests pass. | PASS |
| `RedisOptimizationJobStore.Load` | Redis JSON must decode to a valid worker envelope and authoritative alternatives. | Missing, expired, malformed failure, omitted/null/string score, and invalid finite score paths fail; explicit zero loads. | Request context is propagated; no detached load. | Owner data is checked before HTTP projection. | One bounded GET plus expiry lookup. | Clear boundary and shared decoder. | Redis raw-payload table and full worker tests pass. | PASS |
| `RedisOptimizationJobStore.PublishCompleted` | Completed publication writes only one-to-three valid alternatives. | Terminal state is idempotent; new invalid result shapes and scores fail before transition. | Redis transition is context-bound and terminal guarded. | Internal alternatives are no longer trusted for finite score bounds. | One validation pass and bounded write. | Reuses shared predicate. | Invalid finite scores and valid rounded score pass Redis integration. | PASS |
| `RedisOptimizationJobStore.PublishFailed` | Failed publication contains canonical safe failure and valid partial alternatives. | Rejects invalid code/message, too many alternatives, and malformed partial result. | Context-bound terminal transition. | No diagnostic text crosses persistence. | Bounded JSON/write. | Small boundary. | Worker safe-message/shape tests and full suite pass. | PASS |
| `validateOptimizationJob` and `validateOptimizationAlternatives` | Status/failure/alternative cardinality and result shape agree. | Rejects unknown status, failure mismatch, non-result alternatives, malformed decoded scores, and invalid finite scores. | Pure validation around decoded state. | Prevents alternate store and Redis data from bypassing result checks. | Linear in at most three alternatives. | Correct shared predicate reuse. | Redis/HTTP raw payload tests and finite-score fixtures pass. | PASS |
| `handleProcessingError` and `publishFailure` | Terminal validation/infeasible/timeout policy is distinct from retryable infrastructure/unknown policy. | Parent cancellation and queue outage remain pending; deadline finalizes with bounded context; invalid pre-classified solver failures are normalized before retry. | Uses `WithoutCancel` only for live-parent timeout finalization; releases admission after publication. | Fixed code/message only. | Bounded publication and no extra I/O. | Policy is explicit. | Full and race suites plus typed-nil/invalid non-nil worker tests pass. | PASS |
| Telemetry and safe-message maps | Every retained code maps to fixed telemetry and canonical user text. | Unknown code defaults to bounded failed/worker text. | Pure maps. | No diagnostics or identifiers in public labels/messages. | Constant-time. | Fixed switches are idiomatic. | All-four vocabulary table passes. | PASS |
| `OptimizationController.GetJob` | Authenticated owner-only polling validates loaded result before DTO projection. | Auth, UUID, expiry, ownership, malformed failure, malformed raw score, and invalid finite score paths are handled. | Request context controls store load. | Owner check precedes public data; cross-user expiry is hidden. | One load and bounded projection. | Explicit route boundary. | HTTP raw-payload and finite-score tests pass. | PASS |
| HTTP result guard and `optimizationJobData` | Public DTO emits only validated alternatives and safe failures. | Invalid struct/decoded results become bounded dependency errors; valid zero and rounded scores project. | Pure projection after guard. | No score or diagnostic bypass through normal route. | O(alternatives plus meals), bounded. | Simple DTO mapping. | HTTP invalid-score and raw-score tables pass. | PASS |
| Validator adversarial tests | Lock score, codec, and failure-classification invariants at the domain boundary. | Covers finite invalid, NaN, infinity, typed-nil, invalid non-nil, and four-code JSON cases. | Deterministic injected solver. | Ensures no diagnostic leak. | Small fixtures. | Table-driven and direct. | All repaired unit cases pass. | PASS |
| Worker publication/deadline tests | Prove persistence and worker retry policy at real seams. | Covers publication rejection, Redis round trip, raw omitted/null/string/zero score, cancellation, safe messages, and typed nil/invalid failures. | Race command passes; Redis integration uses current local service. | Persistence cannot accept malformed or unsafe values. | Bounded test data. | Focused and readable. | Full worker and focused race suites pass. | PASS |
| HTTP projection tests | Prove authenticated projection rejects malformed persisted alternatives. | Covers negative, above-one, off-grid, omitted, null, string, and valid zero/rounded scores. | Fiber fake store and raw decoder are deterministic. | Owner scope and safe failure tests pass. | Bounded response fixtures. | Direct table tests. | Full HTTP and focused race suites pass. | PASS |
| OpenAPI/generator/drift contract | Source, generated enum, and score schema agree. | Schema requires numeric score, bounds, and multipleOf. | Static files only. | Diagnostics excluded from public enum. | No runtime cost. | Drift test prevents silent contract divergence. | Redocly, generator check, and focused drift pass. | PASS |
| Generated types and optimization client decoder | Client accepts only generated shape, bounded scores, and retained codes. | Unknown/empty codes and invalid score values fail closed client-side. | Poll cancellation/late result handling remains bounded. | Server messages are normalized to safe UI text. | At most three alternatives. | Uses generated types and guards. | Full Bun suite passes and client malformed-data tests pass. | PASS |
| Optimization store/UI/browser consumers | Terminal/operation messages and score display match the server contract. | Completed, failed, queue, expiry, cancellation, retry, and unmount flows are covered. | AbortController cancels local polling and ignores late result. | No raw server diagnostics rendered. | Bounded rendering. | Typed maps and generated DTOs. | Full Bun, build, browser, and accessibility capture pass. | PASS |
| `assertTask206Failure` | Compatibility integration assertion uses the closed code value. | All four expected failure strings remain testable. | N/A — test-only assertion. | N/A — no public data path. | N/A — test-only. | Minimal migration. | Full backend suite passes. | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| None | N/A | N/A | No unresolved correctness, security, behavior, or coverage finding. | The two prior findings were reproduced after repair: omitted/null/string scores fail during JSON decode, explicit zero succeeds, and `safeOptimizationFailure(&OptimizationFailure{})` yields valid `worker_crash`. | No repair required. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -count=1` | `backend/` | 0 | PASS | Current focused worker/controller/optimization packages; Redis raw-score tests included. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/optimization ./internal/worker ./internal/httpapi -run 'TestValidateDietAlternative\|TestOptimizationFailureClassificationHandlesTypedNilErrors\|TestSafeOptimizationFailureNormalizesInvalidExistingFailure\|TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode\|TestTask221RedisStoreRejectsMalformedRawSimilarityScore\|TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown\|TestOptimizationProcessorTreatsInvalidFailureAsRetryableWorkerCrash\|TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection\|TestOptimizationHTTPRejectsMalformedRawSimilarityScore' -count=1` | `backend/` | 0 | PASS | Current repaired score-presence/zero and invalid-error focused race set. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | `backend/` | 0 | PASS | Full backend suite. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | `backend/` | 0 | PASS | Full backend race suite. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend/` | 0 | PASS | Backend static analysis. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -coverprofile=/tmp/task-221-current.coverage.out -count=1 && go tool cover -func=/tmp/task-221-current.coverage.out | tail -1` | `backend/` | 0 | PASS | `/tmp/task-221-current.coverage.out`; scoped aggregate 84.0% statements (optimization 84.1%, worker 64.8%, HTTP 89.6%). |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS | No called vulnerabilities. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | PASS | 365 tests / 1603 expectations. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | PASS | Vite production build. |
| `python3 scripts/verify-frontend.py` | repository root | 0 | PASS | Chromium desktop/mobile/browser acceptance; screenshots under `/tmp/mealswapp-frontend-verifier/`. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 237 sequential tasks; statuses unchanged. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability validation. |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | PASS | Generated API types current. |
| `python3 -m unittest scripts.test_generate_api_types.OperationResponseDriftTest.test_optimization_terminal_vocabulary_and_similarity_projection_match_generated_contract` | repository root | 0 | PASS | Focused terminal-vocabulary/score drift test. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with pre-existing warning | One OAuth callback no-2XX warning; no Task 221 schema error. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-221-review.md` | repository root | 0 | PASS | Final structural evidence validation. |

## 9. Files Inspected and Staleness Fingerprints

SHA-256 fingerprints below were captured from current contents after the independent verification commands. Shared files include later cumulative edits; Task 221 symbols were re-read and Task 222 symbols were excluded by scope.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `docs/design/DESIGN-004.md` | design source and policy matrix | score formula and failure policy | SHA-256 | `1ee9b87165cb5b1eff43cf6a39b32c20ebab02ae8ab1cddc9f98c3c6152caf0d` |
| `api/openapi.yaml` | public enum/score contract | required numeric score | SHA-256 | `b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585` |
| `scripts/generate-api-types.py` | generated enum template | contract generation | SHA-256 | `1d3df961971558688facf503afdf1d014e802ba0bca9f12e7d786f6ab7752954` |
| `scripts/test_generate_api_types.py` | contract drift tests | enum/score drift | SHA-256 | `b21cee3080b7e93f8827b690e88076dcc91d18c219714665b1add1df720f1ff0` |
| `backend/internal/optimization/validator.go` | domain validator/classifier | F-1/F-2 | SHA-256 | `4119ce67e0353c725d9e0feca9f13379966f1d89eaee5477bee706929ae6b09a` |
| `backend/internal/optimization/validator_test.go` | domain regression tests | missing invalid non-nil error/JSON fixture | SHA-256 | `950349bf26c0a28df9b2f3037243b26d0ccd0c0203f69fb5d76b612ad1742f73` |
| `backend/internal/worker/optimization_processor.go` | Redis/worker boundaries | F-1/F-2 | SHA-256 | `50ea0a2165cb6ec19f4d4fcb7f83d1ce51ff1f65f569dcf788d652b2d8933427` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | worker policy tests | missing invalid non-nil fixture | SHA-256 | `fc79e1cc9329eac1e5f773ef9ba3c6acf1a226e4e8876d843bb09e9af0de5b37` |
| `backend/internal/worker/task221_publication_test.go` | Redis publication/decode tests | missing null/omitted raw JSON fixture | SHA-256 | `8e7270f161af93763c3e6023b48e8c73304b841aaca2548305593594e31f97f3` |
| `backend/internal/httpapi/optimization_controller.go` | authenticated polling/projection | F-1 caller guard | SHA-256 | `422b0232a203d05071e33c050fafe40a681120ee4544011d6c2c100405208664` |
| `backend/internal/httpapi/optimization_controller_test.go` | HTTP regression tests | finite score fixtures pass | SHA-256 | `4425e033ea214aee6e35bb1066cd6296243a8ed7d601e961818a305472fb811b` |
| `backend/internal/app/task206_backend_integration_test.go` | compatibility caller | no finding | SHA-256 | `117336022754a3f3008efbd07646f761a5f9d6765810a121440b03a3b7bcd757` |
| `frontend/src/lib/api/generated.ts` | generated optimization DTOs | no finding | SHA-256 | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `frontend/src/lib/api/optimization-client.ts` | runtime decoder and guards | client defense cannot repair server decode gap | SHA-256 | `71ca43db9783f42c59d6523290b2e60be24f25c51ce1b1d5df73955755c330f7` |
| `frontend/src/lib/api/optimization-client.test.ts` | client malformed-data tests | no server-side finding | SHA-256 | `fab6abd590530acaf2d735ffc8c35f787d880f10bc6c1d887e70f83332aad744` |
| `frontend/src/lib/stores/optimization.ts` | UI failure policy | no finding | SHA-256 | `9a117da419eb59de2e323f0e10ea7ab9bd3d8c1505b896966ceb38d6e10146e1` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | score/error consumer | no finding | SHA-256 | `08f324d14deffedbf71a9735fac75f0ed031f49fbb09d3135632259d592b1783` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | UI source tests | no finding | SHA-256 | `022b8e15728f1808c7397ee2ffd9b31f9c56d8af0915c28ab6b3da1ceb87a28d` |
| `frontend/tests/optimization-workflow.spec.ts` | browser acceptance | no finding | SHA-256 | `d8fb06f4cfcfea01728816b11b1435299c4a5f6428fbce46368e855e40bbf375` |
| `docs/implementation/preparation/task-221-preparation.md` | claimed repair evidence | prior evidence checked | SHA-256 | `0410279b0ed4db2c14f063a99771a74ca56bf15fc0142e260e9baf6e159ed573` |
| `docs/implementation/preparation/task-222-preparation.md` | later-task scope boundary | preserved/excluded | SHA-256 | `036dcae0624c41899684640e205714aec0cad0a7a4f75c2d604574ab78971461` |
| `docs/implementation/02_TASK_LIST.md` | status/dependency boundary | unchanged | SHA-256 | `ff97c9908298a6215b3211cce5ebb8931569940d2e534b3387b1c8b60374f6d4` |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior rejected review hashes and findings describe the pre-repair snapshot; finite score and typed-nil repairs were re-read against current source."
  - "Preparation hashes for shared files are not reused as current hashes because Task 222 overlays remain present."
  - "Task 222 preparation and implementation are preserved and excluded from Task 221 findings."
~~~

## 10. Coverage and Exceptions

- [x] Required scoped coverage command ran.
- [x] Coverage artifact path and observed percentages are recorded.
- [x] Defensive/error branches relevant to the changed symbols were source-audited.
- [x] The documented existing phase coverage exception was not expanded to conceal these findings.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-221-independent.coverage.out"
observed_line_coverage: "84.3% scoped backend statements; optimization 85.3%, worker 64.6%, HTTP 89.7%"
coverage_passed: true
~~~

Coverage finding: the command passed and broad suites are healthy, but coverage cannot establish JSON field presence/type after decoding or normalize an invalid non-nil pre-classified error. Those are correctness gaps, not coverage-only exceptions.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by the reviewed Task 221 symbols.
- [x] No source-of-truth documentation was contradicted for the normal score/failure paths.
- [x] No generated/cache/build/temporary artifact was intentionally added by this review.
- [x] Public API additions are necessary and used; generated artifacts match the OpenAPI source.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged; two malformed/error cases remain findings.

Findings: the repaired finite score boundary, HTTP guard, typed-nil classifier, cancellation/retry semantics, and all broad regression commands pass. The current server still accepts malformed persisted `similarityScore` presence/type and preserves a non-nil invalid `OptimizationFailure`; both are recorded as unresolved.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

~~~yaml
decision: "PASSED"
reason: "The repaired presence-aware score decoding and invalid-failure normalization pass the current focused and regression audit with no unresolved blocking or important findings."
failed_criteria: []
failed_or_unaudited_symbols:
  - "safeOptimizationFailure"
  - "RedisOptimizationJobStore.Load / validateOptimizationJob JSON presence boundary"
recommended_next_action: "Repair F-1 with presence-aware persisted DietAlternative decoding and raw Redis regression fixtures; repair F-2 by preserving existing OptimizationFailure only when its code is valid and add non-nil invalid-error worker/telemetry coverage; then rerun the scoped evidence and re-review current shared-file hashes."
~~~

## 13. Repair Context

### Failure Summary

The previous review's F-1 finite score persistence/HTTP projection gap and F-2 typed-nil classifier panic are repaired: the shared `ValidateDietAlternative` predicate is now called from publication, decoded-job validation, and HTTP projection, and typed-nil error targets are nil-checked. Independent review found two residual gaps. Redis JSON decoding does not preserve field presence/type for a `float64`, so omitted/null `similarityScore` becomes valid zero. Separately, `safeOptimizationFailure` trusts any non-nil `*OptimizationFailure`, including the zero-valued invalid code.

### Minimal Repair Goal

Make persisted score decoding fail closed for missing/null/non-number score fields while preserving valid zero scores, and normalize every non-nil invalid pre-classified optimization error to bounded `worker_crash` behavior.

### Evidence to Reuse

`docs/implementation/preparation/task-221-preparation.md`; current `validator.go`, `optimization_processor.go`, HTTP projection and regression tests; `/tmp/task-221-independent.coverage.out`; the focused and full test results recorded in section 8; and the current SHA-256 inventory in section 9.

### Required Re-Review Surface

`DietAlternative` decode shape, `RedisOptimizationJobStore.Load`, `validateOptimizationJob`, `ValidateDietAlternative`, `safeOptimizationFailure`, `FailureCodeOf`, `handleProcessingError`, `solverTelemetryStatus`, `task221_publication_test.go`, validator typed-nil/error tests, and the HTTP malformed-result regression. Re-check the unchanged Task 222 controller/publication symbols only for merge-safe shared-file attribution.

### Do Not Change

Do not change Task 222 submission idempotency, durable acknowledgement, request hashing, admission, queue publication/repair, response-matrix behavior, task statuses, or unrelated production code. Preserve valid rounded score `0` and the repaired typed-nil retry semantics.
