# Review Evidence: Task 230 — Strict Optimization Client and Caller-Owned Submission Key

~~~yaml
task_id: 230
phase: "07.01"
component: "DESIGN-001: SearchView"
static_aspect: "SearchView strict optimization client and caller-owned submission key"
input_status: "OPEN (preserved; task-status edits were prohibited)"
review_decision: "PASSED"
decision: "PASSED"
reviewed_at_utc: "2026-07-18T11:55:48Z"
review_agent: "Codex independent owner re-review"
evidence_file: "docs/implementation/reviews/task-230-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus cumulative dirty Phase 07.01 worktree"
baseline_confidence: "HIGH"
inventory_symbol_count: 14
audited_symbol_count: 14
inventory_source_count: 14
code_review_skill_invoked: true
code_review_skill_invocation_count: 1
relevant_language_guides: "TypeScript, Svelte, security, async/concurrency, common-bugs, architecture, performance, and universal-quality guidance applied"
review_template_path: "docs/implementation/reviews/REVIEW_TEMPLATE.md"
review_template_available: false
fallback_review_template_path: "review.txt"
fallback_review_template_sha256: "f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20"
prior_rejected_review_sha256_before_rewrite: "1b8d05ea6b7d42cdc617d1e4f23b5d129cce24f1fb223854baaa794d6d47c448"
blocking_findings: 0
important_findings: 0
optional_findings: 0
repair_context_required: true
~~~

## 1. Task Source

**Description:** Phase 07.01: replace optimization response assertions and partial normalization with exact runtime decoding for acknowledgement and all job variants, enforce exact endpoint statuses and bounded canonical poll URLs, and require the controller to supply a collision-resistant memory-only submission key.

**Task row:** `docs/implementation/02_TASK_LIST.md:237`; status is `OPEN` and was not changed.

**Dependencies:** Tasks 213, 221, and 227 are `PASSED` in the current task table. Task 231 remains `OPEN` and owns retry-policy, identity-lifecycle, and polling-configuration work beyond this review.

**Design sources:** `docs/design/DESIGN-001.md` (`SearchView`), `docs/architecture/ARCH-001.md` (SPA and local-persistence boundaries), `docs/design/01_TECH_STACK.md`, and the generated optimization wire contract in `docs/design/DESIGN-004.md`.

**Preparation source:** `docs/implementation/preparation/task-230-preparation.md` was read in full. Its F-230-01 repair claims were independently checked against the current OpenAPI source, generator, tests, runtime decoder, controller, and current command results.

**Prior rejected review:** The prior `task-230-review.md` was read in full before replacement. It rejected only F-230-01: the generator guard did not detect a stricter optimization-envelope `requestId` bound. The pre-rewrite evidence hash is recorded above.

**Template note:** `docs/implementation/reviews/REVIEW_TEMPLATE.md` is absent both from this checkout and the `HEAD` tree. The complete `review.txt` fallback, `docs/implementation/reviewer-prompt.md`, and the established review-evidence structure were read and used. No unrelated template was created.

**Decision at a glance:** F-230-01 is repaired and passes direct mutation probes. Exact `202`/`200` status handling, all five job variants, strict nested decoding, bounded canonical poll URLs, safe failures, caller-owned canonical UUIDv4 keys, ambiguity reuse, deliberate rotation, concurrency suppression, memory-only lifecycle, generated drift checks, and frontend gates all pass.

## 2. Pre-Review Gates

- [x] The exact Task 230 row was read; it remains `OPEN`.
- [x] Full `docs/design/DESIGN-001.md`, `docs/architecture/ARCH-001.md`, `docs/design/01_TECH_STACK.md`, and the relevant `DESIGN-004` contract were read.
- [x] Full `docs/implementation/preparation/task-230-preparation.md` and the prior rejected review were read.
- [x] Full `review.txt` fallback and `docs/implementation/reviewer-prompt.md` were read; the requested `REVIEW_TEMPLATE.md` path was checked and is absent.
- [x] `code-review-skill` was invoked exactly once and its applicable TypeScript, Svelte, security, async/concurrency, common-bug, architecture, performance, and universal-quality guidance was applied.
- [x] The full Task 230 implementation surface was re-audited: client decoder, all variants, nested values, safe failures, controller key lifecycle, caller/component path, OpenAPI/generated contract, drift guard, and focused tests.
- [x] F-230-01 was reproduced independently: both envelopes accept exactly the documented request-ID rule, and all eight deliberate mutations are rejected.
- [x] Focused and full frontend tests, focused and aggregate coverage, typecheck, build, generated-output, OpenAPI, task-list, traceability, and diff checks passed.
- [x] No merge, reset, checkout, staging, cleanup, code edit, task-row edit, or task-status edit was performed. Only this review evidence file was rewritten.
- [x] The cumulative dirty worktree was preserved; unrelated Phase 07.01 files were not attributed to Task 230.

~~~yaml
pre_review_gates_passed: true
~~~

## 3. Review Baseline and Change Surface

The baseline is commit `a4e31367485b03269e90b5607f2057c9568bb5b1` on branch `multistep-phase-07`, plus the cumulative dirty Phase 07.01 worktree. `git status --short` reports 126 pre-existing changed or untracked paths. Attribution was reconstructed from the Task 230 preparation manifest, exact task row, current symbols, direct callers, tests, and content hashes rather than treating the aggregate dirty-worktree diff as Task 230.

Task 230 owns these boundaries:

1. `submitOptimization` requires a caller-provided optimization-prefixed canonical UUIDv4 key, validates it before CSRF or network I/O, and accepts only HTTP `202`.
2. `getOptimizationJob` accepts only HTTP `200`; both success paths require exact envelopes and reconstruct only validated fields.
3. Acknowledgement and polling payloads use bounded printable request IDs, UUIDs, strict timestamps, exact relative poll URLs, exact variant property sets, fixed safe terminal failures, bounded alternatives, and validated nested meals/macros/similarity.
4. The controller keeps one submission intent in a closure-local pending object, reuses its key for ambiguous pre-acknowledgement failures, clears the reusable copy after acknowledgement, rotates it for deliberate new intent, suppresses concurrent submissions, and clears it on diet scope change or disposal.
5. The API client/controller path contains malformed polling payloads as safe state failures; no raw response fields are spread into store state or rendered.
6. The generator checks the current optimization schema assumptions before generated-output comparison, including exact request-ID bounds and safe characters.

Task 231's broader poll configuration, remount/global-store, and authenticated identity lifecycle actions remain outside this review. No Task 230 regression or unresolved Task 230 finding was identified.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Evidence and conclusion |
|---:|---|---|---|
| 1 | Exact `202` acknowledgement and exact `200` queued/processing/completed/failed/cancelled polling responses are accepted. | PASS | `submitOptimization` checks `response.status === 202`; `getOptimizationJob` checks `=== 200`; valid acknowledgement and all five variant tests pass. |
| 2 | Wrong successful statuses, wrong envelope statuses, unsafe request IDs, malformed UUIDs/dates, and missing/additional/cross-variant fields are rejected. | PASS | Exact envelope/property checks and adversarial tables reject malformed data before it reaches the returned DTO or store. |
| 3 | Acknowledgement and job poll URLs are bounded, relative, canonical, and tied to the validated job ID. | PASS | `canonicalPollUrl` requires the exact `/api/v1/optimization/jobs/{jobId}` path and a length bound; absolute, host-relative, mismatched, suffixed, and oversized hostile values fail. |
| 4 | Nested alternatives, meals, units, quantities, positions, macros, calories, similarity, terminal codes, and safe messages are strict. | PASS | Exact nested objects, `1..100` meals, completed `1..3` alternatives, failed `0..3` alternatives, canonical units, finite bounds, four-decimal similarity, and fixed failure code/message pairs pass valid and hostile fixtures. |
| 5 | Malformed success data cannot reach store/rendering state or crash polling. | PASS | The strict client-to-controller integration test leaves no job or alternatives and exposes only the fixed `malformed_optimization_response` failure. |
| 6 | Direct submission cannot compile or execute without a caller-owned key. | PASS | `OptimizationSubmissionOptions.idempotencyKey` is required; compile-time omission, runtime omission, weak-key, and pre-I/O no-network tests pass. |
| 7 | Ambiguity reuses one key; deliberate new intent rotates it; concurrent submissions are suppressed. | PASS | Controller tests assert `key-1`, `key-1` for ambiguity and `key-2` for deliberate intent, while concurrent submits produce one in-flight submission. |
| 8 | Secure randomness fails closed and keys remain memory-only, private, and cleared on diet scope/disposal. | PASS | Only `crypto.randomUUID()` is used; missing/throwing/non-v4/wrong-variant/noncanonical results fail safely; storage access is zero and pending state is closure-local. |
| 9 | Generated contract drift is guarded for the repaired request-ID policy. | PASS | OpenAPI declares the exact rule on both envelopes; the generator compares the complete rule; 16 Python tests include eight request-ID mutations, and the independent 8/8 probe rejects all of them. |
| 10 | Frontend tests and build/type/generated/OpenAPI checks pass. | PASS | 33 focused tests, 414 full tests, typecheck, build, coverage, generated check, task-list, traceability, OpenAPI lint, and diff checks pass. |

## 5. Changed-Symbol Inventory

| # | Grouped symbol/unit | File:line | Task 230 surface audited | Result |
|---:|---|---|---|---|
| 1 | `submitOptimization`, `getOptimizationJob`, request/error helpers, `OptimizationClientError` | `frontend/src/lib/api/optimization-client.ts:23-142` | Exact endpoint statuses, CSRF/network/error paths, generated request helpers, and malformed-success containment | PASS |
| 2 | `decodeEnvelope`, `decodeAcknowledgement`, `safeRequestId`, `uuid`, `dateTime`, `canonicalPollUrl` | `frontend/src/lib/api/optimization-client.ts:144-150,271-306` | Exact envelopes, request IDs, UUID/date validation, and bounded canonical URL policy | PASS |
| 3 | `decodeJobEnvelope`, `decodeJobCommon`, `decodeFailedJob`, `assertKeys`, `exactObject` | `frontend/src/lib/api/optimization-client.ts:152-202,256-269` | All five discriminated job variants and required/optional/forbidden fields | PASS |
| 4 | `decodeAlternatives`, `decodeAlternative`, scalar validators | `frontend/src/lib/api/optimization-client.ts:204-222,308-338` | Strict nested meals, macros, units, cardinality, quantities, positions, and similarity | PASS |
| 5 | `generateOptimizationIdempotencyKey`, `validIdempotencyKey`, `secureRandomUnavailable` | `frontend/src/lib/api/optimization-client.ts:82-96,249-292` | Collision-resistant UUIDv4 generation, format enforcement, and fail-closed secure-random behavior | PASS |
| 6 | Optimization client fixtures and 17 client tests | `frontend/src/lib/api/optimization-client.test.ts:65-421` | Status/envelope/variant/nested adversarial tables, safe failures, key ownership, URL safety, and randomness | PASS |
| 7 | `PendingSubmission`, `createOptimizationController`, `submit`, `runSubmission` | `frontend/src/lib/stores/optimization.ts:99-189` | Caller-owned key creation, private pending state, ambiguity retention, acknowledgement handoff, and suppression | PASS |
| 8 | `retry`, `pollExistingJob`, `pollJob`, `beginOperation`, `setDiet`, `dispose` | `frontend/src/lib/stores/optimization.ts:123-280` | Key reuse/rotation boundary, polling terminal handling, abort invalidation, scope clearing, and late-result suppression | PASS |
| 9 | Optimization controller fixtures and 11 controller tests | `frontend/src/lib/stores/optimization.test.ts:78-370` | Key reuse/rotation, concurrency, secure-random failure, memory-only lifecycle, and strict-client containment | PASS |
| 10 | Optimization routes, response statuses, schemas, variants, and nested bounds | `api/openapi.yaml:627-704,1406-1671` | Source-of-truth `202`/`200` matrix and optimization response contract, including both request-ID envelopes | PASS |
| 11 | Generated optimization DTOs, discriminated union, status helpers, and request builders | `frontend/src/lib/api/generated.ts:519-700` | Generated type/request surface, five variants, exact request header, and current output | PASS |
| 12 | `OPTIMIZATION_REQUEST_ID_RULE`, schema rules, property rules, mismatch guard, generator gate | `scripts/generate-api-types.py:185-369,1658-1697` | Contract drift enforcement before generated-output comparison, including complete request-ID rules | PASS |
| 13 | Optimization generator/status/mutation tests | `scripts/test_generate_api_types.py:34-125` | Current output, response matrix, nested/variant/schema mutations, idempotency wiring, and eight request-ID mutations | PASS |
| 14 | `OptimizationWorkflow` caller/rendering surface and source-contract tests | `frontend/src/lib/components/OptimizationWorkflow.svelte:1-200`; `frontend/src/lib/components/OptimizationWorkflow.test.ts:1-64` | Controller-only submission path, terminal-state rendering, bounded alternatives, and no direct key/storage exposure | PASS |

~~~yaml
inventory_symbol_count: 14
inventory_complete: true
inventory_grouping_note: "Rows group only tightly coupled symbols with one contract/evidence boundary; decoder stages, controller key stages, and generator guard/tests remain separate because they have distinct acceptance risks."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal, edge, and error paths | Security, concurrency, and resource audit | Tests and result |
|---|---|---|---|---|
| `submitOptimization` / `getOptimizationJob` | Generated request helpers are used; POST is exactly `202` and GET exactly `200`. | CSRF, network, non-2xx, malformed JSON, and unexpected successful statuses use safe client errors. | Caller key is validated before CSRF/network; polling requests use encoded generated paths and response URLs are independently canonicalized. Focused status/error tests pass. |
| `decodeEnvelope` / `decodeAcknowledgement` | Success envelopes have exactly `status`, `requestId`, and `data`; endpoint discriminators are exact; acknowledgement data is exactly `jobId/status/pollUrl`. | Null, array, missing, extra, wrong status/request ID, wrong job status, malformed UUID, and mismatched URL reject with a fixed error. | No untrusted envelope fields are spread into state; request IDs are bounded safe metadata. Acknowledgement adversarial tests pass. |
| `decodeJobEnvelope` / `decodeJobCommon` | Status selects queued, processing, completed, failed, or cancelled; common UUID/date/path fields are required. | Each variant has an exact property set; optional failed timestamps are nullable only when present; cross-variant fields reject. | Reconstruction returns only validated fields and blocks backend diagnostics. All five valid variants and hostile variants pass. |
| `decodeFailedJob` | Failure code and message must be one of the four fixed canonical pairs; partial alternatives are optional and bounded. | Unknown/empty codes, unsafe messages, malformed optional dates, additional failure fields, and invalid partial alternatives reject. | Failure diagnostics never render from the response. Valid partial and zero-alternative failed fixtures pass. |
| `decodeAlternatives` / `decodeAlternative` | Completed alternatives are `1..3`; failed alternatives are `0..3`; each nested object is exact. | Empty/over-limit arrays, wrong types, non-finite/out-of-range values, unsupported units, grid violations, and legacy calorie placement reject. | Arrays are bounded and DTOs are reconstructed; raw objects do not cross the boundary. Nested adversarial tables pass. |
| `generateOptimizationIdempotencyKey` / `validIdempotencyKey` | Only `optimization-` plus a canonical lowercase UUIDv4 with RFC 4122 variant bits is accepted. | Missing crypto, provider throw, null/object, uppercase, non-v4, wrong variant, and malformed output fail closed. | No clock, pseudo-random, storage, or weak fallback exists. Provider and no-I/O failure tests pass. |
| Optimization client test suite | The client tests exercise the public boundary rather than trusting private decoder implementation details. | Valid acknowledgement and all five variants pass; wrong statuses, envelopes, fields, URLs, dates, nested values, failures, and cardinalities fail safely. | Fetch/CSRF calls, malformed-response containment, key omission, weak keys, and secure-random failures are covered. The 17 client tests pass. |
| `createOptimizationController.submit` / `runSubmission` | One closure-local key belongs to one intentional request; acknowledgement clears the reusable key copy. | Ambiguous pre-ack errors retain the key; accepted acknowledgement transitions to polling; malformed acknowledgement becomes safe failure. | Busy phases suppress duplicate work; request exclusions are cloned; late operations require the current token. Key-sequence and integration tests pass. |
| `retry` / `pollExistingJob` / `pollJob` | Pre-ack retry reuses the same key; post-ack retry polls the known job; deliberate new submission allocates a fresh key. | Terminal completed/failed/cancelled states stop polling; retryable failures retain the appropriate safe action; expiry starts a new submission. | Abort/token invalidation prevents late commits. Task 231's broader policy remains outside scope. Controller tests pass. |
| `setDiet` / `dispose` / `beginOperation` | Diet scope changes and disposal clear pending key state and invalidate active work. | Active submit/poll aborts; late results are ignored; state is reset for a new diet. | Key is absent from `OptimizationState`; controller state is memory-only and controller-local. Scope/disposal tests pass. |
| OpenAPI optimization schemas and operations | POST/GET success statuses, five variants, nested bounds, and both exact request-ID policies match the decoder. | Response-matrix, Redocly, and deliberate schema mutation checks reject drift; the only Redocly warning is the pre-existing OAuth callback 2XX warning. | Server-owned IDs and safe terminal vocabulary remain outside client-submitted authoritative data. Current source passes. |
| Generated optimization types and request builders | The generated union represents all five variants, completed cardinality, failed partial alternatives, endpoint paths, and required idempotency header. | Generated output is byte-for-byte current; generated request tests pass. Runtime decoding supplies constraints ordinary interfaces cannot express. | Exact generated headers contain one caller key and credentialed requests. Current output and typecheck pass. |
| `optimization_contract_mismatches` and generator `main` gate | The complete request-ID block is compared separately for acknowledgement and polling envelopes before output comparison. | Empty/over-limit/stricter-maximum/unsafe-pattern mutations all report schema drift; current source reports no mismatch. | A future OpenAPI change cannot silently weaken the exact request-ID decoder assumption. Sixteen Python tests and independent 8/8 probe pass. |
| `OptimizationWorkflow` and its source-contract tests | UI submits only through the controller and displays validated controller state and at most three alternatives. | Busy, failed, expired, completed, empty, retry, and malformed-result states remain bounded and user-safe. | No key field or browser storage access exists in the component path. Five component tests and build pass. |

## 7. Findings

### F-230-01 — Closed: optimization request-ID drift guard now covers the exact runtime policy

The prior review reproduced a valid stricter OpenAPI mutation (`maxLength: 10`) that returned no generator mismatch. The repair is present and independently verified:

- `api/openapi.yaml:1483-1496` and `:1658-1671` declare `requestId` on both optimization envelopes as `type: string`, `minLength: 1`, `maxLength: 120`, and pattern `^[A-Za-z0-9._:-]+$`.
- `frontend/src/lib/api/optimization-client.ts:278-280` enforces the same `{1,120}` safe-token rule at runtime. An independent runtime probe accepted exactly 120 safe characters and rejected 121 characters and unsafe characters.
- `scripts/generate-api-types.py:186-191,361-365` compares the complete request-ID property block for both envelopes before generated-output comparison.
- `scripts/test_generate_api_types.py:95-117` runs eight deliberate subtests: too-short, too-long, reviewer-stricter maximum, and unsafe pattern for each envelope.
- `python3 -m unittest scripts/test_generate_api_types.py` passed all 16 tests. The independent mutation probe rejected all 8 of 8 mutations.

No blocking, important, or optional Task 230 finding remains. The prior rejection is resolved.

## 8. Commands Run

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts` | PASS — 33 tests, 176 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS — 414 tests, 1,877 expectations across 37 files. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts` | PASS — 33 tests; optimization client 97.78% functions / 95.00% lines; optimization store 81.48% functions / 79.69% lines. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | PASS — 414 tests; 93.32% functions / 94.01% lines overall; optimization client 97.78% functions / 95.00% lines; optimization store 81.48% functions / 79.69% lines. Existing Phase 07 frontend coverage exceptions remain recorded in `docs/implementation/04_OPEN.md`; Task 230 adds no exception. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS — Vite transformed 205 modules. |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS — 16 tests. |
| `python3 scripts/generate-api-types.py --check` | PASS — generated API types are current. |
| Independent Python `optimization_contract_mismatches` mutation probe for both request-ID envelopes | PASS — 8 of 8 deliberate mutations rejected: two too-short, two too-long, two stricter maxima, and two unsafe patterns. |
| Independent Bun runtime request-ID boundary probe | PASS — exactly 120 safe characters accepted; 121 characters and unsafe characters rejected. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies; Task 230 remains `OPEN`. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS — valid document with one pre-existing explicitly ignored OAuth callback 2XX warning. |
| `git diff --check -- <Task 230 implementation and evidence paths>` | PASS. |

The root aggregate `scripts/check.py` was not run because it invokes Docker, browser, backend, vulnerability, local-stack, and phase-wide gates outside this frontend/API task. The scoped frontend, OpenAPI, generator, traceability, task-list, build, and coverage gates were run directly.

## 9. Files Inspected and Staleness Fingerprints

The following current SHA-256 values were recomputed after the audit. The review file itself is excluded because this re-review rewrites it. The prior rejected review hash before rewrite is recorded in the front matter.

| File | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `a9f42df2aa3ef1090406bfbc5fc2f3e51c2ff0aca63c977359ba5349c3487264` |
| `docs/design/DESIGN-001.md` | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/architecture/ARCH-001.md` | `03fcbae9676ecf278e72a621c7fd9911d0c3dc6ee15eaebcec308d010cc76833` |
| `docs/design/01_TECH_STACK.md` | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/design/DESIGN-004.md` | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/implementation/04_OPEN.md` | `c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527` |
| `docs/implementation/reviewer-prompt.md` | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |
| `review.txt` | `f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20` |
| `docs/implementation/preparation/task-213-preparation.md` | `83bb5de5f8c4138c9e15d1f8a64725cc0fe5b7ee31f2ae3ed3f68dafa28ccea0` |
| `docs/implementation/preparation/task-221-preparation.md` | `ecf646e5b92139608ac4b74326f7d921064a24d420deb22b764a2a3e6657a632` |
| `docs/implementation/preparation/task-227-preparation.md` | `dd4b4e05d2c665fcc461ebbe964a8b127dafb55e1813ec9e240235697109944d` |
| `docs/implementation/preparation/task-230-preparation.md` | `11849b62ffacb742e7bf6a0269729bfa8a36d9e4b9aa4a353f05e3fc9f2b9964` |
| `docs/implementation/reviews/task-221-review.md` | `5d371e74117f5d68ed8189eeae53a3155fedb561177f2f254eac1b3d7ba28e72` |
| `api/openapi.yaml` | `392a3d531301a937b001bc7561b6e5cdef76a6a786d2073d739ab81cd1161c4a` |
| `frontend/src/lib/api/generated.ts` | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `frontend/src/lib/api/error-message-mapper.ts` | `7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e` |
| `frontend/src/lib/api/optimization-client.ts` | `c047e9ab5bd97ac381b8efa72d6d99fa362e4973c3b60d785348715bac2b4c09` |
| `frontend/src/lib/api/optimization-client.test.ts` | `e67cf00595ab34c40510f76a4a1b256cb570c4ec4c371c493d4ff8eedb79d280` |
| `frontend/src/lib/stores/optimization.ts` | `a2e959c819daa0a0a1d1cf685e13c36926bcf24d9e55786205a3b46c3301019e` |
| `frontend/src/lib/stores/optimization.test.ts` | `d1c25017a9a48f1fb576b549f20a1b8c6e46d5a9854a7c4d8758004a9f9a8efb` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `620e825cd23e258fee69ccb42899e00c01f2dc7a53df5d5b8e3d9cc3c6f00b33` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | `022b8e15728f1808c7397ee2ffd9b31f9c56d8af0915c28ab6b3da1ceb87a28d` |
| `scripts/generate-api-types.py` | `c2fdf54b8280eedf91b149ae9f94fd8d1f9a01d22095b57bb53309f792313acc` |
| `scripts/test_generate_api_types.py` | `3a1116c9165f67386e315ef380b52083c805f962e406e9a1282300d533b2813a` |

~~~yaml
all_reviewed_files_hashed: true
hash_scope: "All Task 230 implementation, contract, design, planning, dependency, and evidence inputs listed above; procedural skill and validator files were not implementation-baseline inputs."
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "REVIEW_TEMPLATE.md is absent; review.txt was used as the complete fallback."
  - "The prior rejected review hash was captured before this evidence rewrite; its F-230-01 finding was independently reproduced and then closed."
  - "The cumulative dirty worktree contains unrelated Phase 07.01 work; attribution was kept at symbol and boundary level."
  - "The task-list and preparation hashes embedded in earlier evidence predate later concurrent Phase 07.01 task-row additions; the current hashes above are authoritative for this re-review."
~~~

## 10. Coverage and Exceptions

- [x] Focused client/controller/component tests pass: 33 tests and 176 expectations.
- [x] Full frontend tests pass: 414 tests and 1,877 expectations.
- [x] Focused coverage reports 97.78% optimization-client functions / 95.00% lines and 81.48% optimization-store functions / 79.69% lines.
- [x] Full coverage reports 93.32% functions / 94.01% lines overall; optimization client remains 97.78% functions / 95.00% lines and optimization store 81.48% functions / 79.69% lines.
- [x] Typecheck, build, generated output, task-list, traceability, OpenAPI lint, and diff checks pass.
- [x] Task-specific F-230-01 drift coverage is complete: both envelopes have exact request-ID rules and eight deliberate mutations are rejected.
- [x] Existing Phase 07 frontend coverage exceptions remain recorded in `docs/implementation/04_OPEN.md`; this re-review does not add or alter an exception.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "Bun test --coverage stdout; no persistent coverage artifact committed"
observed_line_coverage: "94.01% aggregate; 95.00% optimization client; 79.69% optimization controller/store"
coverage_passed: true
coverage_reason: "All Task 230 gates pass; the lower aggregate and controller/store figures are within the previously accepted Phase 07 frontend coverage exception and no new exception is added."
~~~

## 11. Negative and Regression Checks

- [x] No weak `Date.now()` or `Math.random()` optimization-key fallback exists in the scoped production surface.
- [x] `submitOptimization` rejects omitted, weak, malformed, non-v4, wrong-variant, uppercase, and noncanonical caller keys before CSRF/network I/O.
- [x] Secure-random unavailability fails closed with a fixed safe error.
- [x] No `localStorage` or `sessionStorage` access exists in the optimization client/controller/component key path; the storage-access test observes zero accesses.
- [x] Ambiguous submission retries reuse the exact key; deliberate new submission allocates a new key; concurrent submits do not fork the intent.
- [x] Diet scope change and disposal clear private pending key state and abort/invalidate active work.
- [x] Exact successful statuses are enforced: POST `202`, poll `200`; unexpected successful `2xx` responses are rejected as malformed.
- [x] All five variants reject forbidden/additional fields; failed partial alternatives may be empty, while completed alternatives must be nonempty and bounded to three.
- [x] Nested meals/macros/units/quantity/position/similarity and fixed safe failure messages are reconstructed only after validation.
- [x] Poll URLs reject absolute, host-relative, mismatched-ID, suffixed, and over-policy values; accepted URLs are the exact canonical relative path.
- [x] Request IDs accept exactly one to 120 safe characters and reject empty, unsafe, and over-limit values at runtime.
- [x] OpenAPI request-ID constraints are exact on both optimization envelopes; stricter, looser, and unsafe mutation tests fail the generator guard.
- [x] Current generated output is byte-for-byte current and all current deliberate optimization schema mutations fail as expected.

## 12. Decision

**PASSED.**

F-230-01 is repaired and closed. The current implementation meets the Task 230 contract: exact endpoint statuses and all five variants pass; envelopes, nested data, terminal codes, messages, UUIDs, dates, cardinalities, quantities, macros, units, similarity, and canonical bounded URLs are fail-closed; malformed payloads cannot enter state or rendering; callers must supply a canonical UUIDv4 optimization key; ambiguity reuses the caller-owned memory-only key; deliberate intent rotates it; concurrent submissions are suppressed; scope/disposal clears it; browser storage is untouched; and the generated request-ID contract cannot drift silently.

Task 230 remains `OPEN` exactly as requested. Task 231's retry-policy, remount/identity lifecycle, and polling-configuration work was not reclassified into this decision.

~~~yaml
decision: "PASSED"
review_decision: "PASSED"
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 13. Repair Context

This is a re-review after the prior rejected review. The repair was limited to the F-230-01 OpenAPI request-ID declarations, generator guard, mutation tests, and refreshed preparation evidence. The re-review changed only this evidence file; it did not edit application code, generated output, unrelated files, the task row, or any task status.

The F-230-01 reproduction (`maxLength: 10` on the acknowledgement request ID) now returns a mismatch, as do the seven other deliberate boundary/pattern mutations. The repair is accepted with no residual finding. Future work owned by Task 231 remains explicitly outside this review.
