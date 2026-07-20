# Task 230 Preparation Evidence

## Scope and authoritative contract

- Task: **230 — Phase 07.01 Strict Optimization Client and Caller-Owned Submission Key**.
- Authoritative row: `docs/implementation/02_TASK_LIST.md:237`, status `OPEN` before and after this work.
- Design owner: `docs/design/DESIGN-001.md`, `SearchView`; wire behavior is constrained by the generated DESIGN-004 optimization contract and the audited response/error policies from Tasks 213, 221, and 227.
- Originating review actions: `docs/implementation/04_OPEN.md:321-322`.
- Scope boundary: only the optimization client, its controller-owned in-memory submission intent, generated optimization contract drift enforcement, focused tests, and this evidence are Task 230. The F-230-01 repair additionally refines only the two optimization-envelope `requestId` OpenAPI schemas. Task 231 retry-policy/lifecycle work, task statuses, generated output, backend code, and unrelated Phase 07.01 changes were not modified.

The worktree already contained cumulative concurrent Phase 07.01 changes and a substantial partial Task 230 implementation. Those changes were preserved and audited in place. The final task attribution is based on the pre-edit hashes captured below, exact symbols, and focused tests rather than the aggregate dirty-worktree diff.

## Sources read

- `docs/implementation/02_TASK_LIST.md:237`: exact Task 230 description, dependencies, and acceptance criteria.
- `docs/design/DESIGN-001.md`: `SearchView`, authenticated action routing, API-error ownership, and browser persistence ownership.
- `docs/architecture/ARCH-001.md`: SPA/API/local-persistence boundaries.
- `docs/implementation/04_OPEN.md:321-322`: strict decoder and caller-owned optimization-key review actions.
- `docs/implementation/preparation/task-213-preparation.md`: exact optimization endpoint status matrix.
- `docs/implementation/preparation/task-221-preparation.md` and `docs/implementation/reviews/task-221-review.md`: terminal failure vocabulary, canonical safe messages, alternative bounds, and similarity semantics.
- `docs/implementation/preparation/task-227-preparation.md`: shared runtime-safe error mapper boundary.
- `api/openapi.yaml`: acknowledgement, five job variants, meal/macro/alternative/failure schemas, statuses, and required idempotency header.
- Current generated types, optimization client/store/component, generator, and all focused tests.

## Implemented runtime contract

### Strict client decoder

`frontend/src/lib/api/optimization-client.ts` now has one fail-closed success boundary:

- `submitOptimization` requires a caller option containing an optimization-prefixed canonical UUIDv4 key before CSRF or fetch I/O and accepts only HTTP `202`.
- `getOptimizationJob` accepts only HTTP `200`.
- `decodeEnvelope` requires exactly `status`, `requestId`, and `data`; acknowledgement status is exactly `accepted`, polling status is exactly `ok`, and request IDs match the bounded printable token policy.
- `decodeAcknowledgement` requires exactly `jobId`, `status`, and `pollUrl`, a UUID job ID, `queued`, and the exact relative `/api/v1/optimization/jobs/{jobId}` poll URL.
- `decodeJobEnvelope` discriminates and reconstructs queued, processing, completed, failed, and cancelled variants without spreading untrusted input. Every variant has an exact required/optional property set, common UUID/date/poll validation, and forbidden cross-variant fields are rejected.
- `decodeFailedJob` accepts only the four terminal codes and their fixed canonical safe messages; arbitrary diagnostics cannot enter client state.
- `decodeAlternatives` / `decodeAlternative` enforce completed cardinality `1..3`, failed cardinality `0..3`, meal cardinality `1..100`, exact nested fields, UUID meal IDs, finite positive quantities up to `1,000,000` on the `0.001` grid, canonical units, integer positions `0..99`, finite macros/calories `0..1,000,000,000`, and similarity `0..1` on the four-decimal grid.
- Malformed JSON, wrong successful statuses, additional fields, wrong types, unsafe URLs/messages, malformed dates/UUIDs, non-finite JSON projections, and legacy calorie placement all become the fixed `malformed_optimization_response` error.

### Caller-owned memory-only key

- `OptimizationSubmissionOptions.idempotencyKey` is required, so direct TypeScript submission without caller ownership does not compile; runtime JavaScript omission or a weak/malformed key fails before I/O.
- `generateOptimizationIdempotencyKey` uses only `globalThis.crypto.randomUUID()`, validates the raw provider result as canonical lowercase UUIDv4 with RFC 4122 variant bits, prefixes it with `optimization-`, and has no clock, pseudo-random, or persistence fallback.
- Missing, throwing, null, object, malformed, non-v4, wrong-variant, uppercase, and otherwise non-canonical provider output returns the fixed `secure_random_unavailable` failure.
- `createOptimizationController` owns the key in its closure. Ambiguous submission failure retains and reuses it; acknowledgement clears submission-key reuse and a deliberate new submission allocates a new key; busy phases suppress concurrent submission; diet changes and disposal clear private pending state.
- Neither the key nor pending submission is exposed through `OptimizationState`, rendered by `OptimizationWorkflow`, or read/written through localStorage/sessionStorage.

### Generated-contract drift guard

`scripts/generate-api-types.py` adds `OPTIMIZATION_REQUEST_ID_RULE`, `OPTIMIZATION_SCHEMA_RULES`, `OPTIMIZATION_PROPERTY_NAMES`, and `optimization_contract_mismatches` and executes the guard before generated output comparison or writing. It locks the decoder assumptions for:

- `MealQuantity`, `MacroProjection`, status/failure enums, alternatives, and failures;
- acknowledgement data/envelope and exact polling envelope;
- the exact `requestId` policy on both optimization envelopes: `1..120` characters from `[A-Za-z0-9._:-]`; the polling envelope covers queued, processing, completed, failed, and cancelled jobs;
- the five discriminated job variants, exact properties, discriminators, dates, UUIDs, poll URL references, nullable failed timestamps, and alternative bounds;
- exactly one optimization-submission `IdempotencyKey` parameter reference.

`scripts/test_generate_api_types.py` deliberately mutates units/quantity grids, macro bounds, terminal vocabulary, similarity precision, failure bounds, acknowledgement shape/status, union discriminator, variant status/additional/required fields, nullable timestamps, cardinality, nested references, idempotency wiring, and both optimization-envelope request-ID policies. Empty-permitting, over-120-permitting, reviewer-reproduced `maxLength: 10`, and unsafe-character mutations are rejected for acknowledgement and the shared five-variant polling envelope. Checked-in generated output remains byte-for-byte current.

### F-230-01 repair

- `api/openapi.yaml` makes the existing runtime `safeRequestId` policy machine-readable on `OptimizationJobAcknowledgementEnvelope.requestId` and `OptimizationJobStatusEnvelope.requestId` with `minLength: 1`, `maxLength: 120`, and `pattern: '^[A-Za-z0-9._:-]+$'`.
- `optimization_contract_mismatches` compares each complete request-ID property rule with `OPTIMIZATION_REQUEST_ID_RULE`, so both relaxed and stricter decoder-relevant drift fail before generated-output comparison. This closes the review repro where acknowledgement `maxLength: 10` previously returned no mismatch.
- `test_optimization_request_id_bounds_and_safe_characters_cannot_drift` runs eight deliberate subtests across both envelopes: too-short, too-long, reviewer-stricter-maximum, and unsafe-character mutations.
- Runtime decoder, controller, component, generated TypeScript, key lifecycle, and browser-persistence behavior are unchanged.

## Exact symbol and test inventory

| File | Symbols / tests | Task evidence |
|---|---|---|
| `frontend/src/lib/api/optimization-client.ts` | `OptimizationSubmissionOptions`, `submitOptimization`, `getOptimizationJob`, `generateOptimizationIdempotencyKey`, `decodeAcknowledgement`, `decodeJobEnvelope`, `decodeJobCommon`, `decodeFailedJob`, `decodeAlternatives`, `decodeAlternative`, `decodeEnvelope`, scalar validators, `secureRandomUnavailable` | Exact statuses, envelopes, variants, nested values, canonical URL, safe failure, and key generation/runtime enforcement. |
| `frontend/src/lib/api/optimization-client.test.ts` | 17 tests at lines 65-423 | Valid acknowledgement/all five variants; status/envelope/UUID/date/property/URL/failure/meal/macro/similarity/cardinality rejection; random-provider table; compile/runtime key ownership. |
| `frontend/src/lib/stores/optimization.ts` | `createOptimizationController`, closure-local `pending`, `submit`, `retry`, `runSubmission`, `setDiet`, `dispose` | Ambiguity reuse, acknowledgement handoff, rotation, suppression, and memory-only lifecycle. |
| `frontend/src/lib/stores/optimization.test.ts` | 11 tests at lines 78-370 | Same-key ambiguity retry, deliberate rotation, concurrent suppression, random failure, storage prohibition, scope/disposal clear, and strict-client-to-store malformed-payload integration. |
| `api/openapi.yaml` | `OptimizationJobAcknowledgementEnvelope.requestId`, `OptimizationJobStatusEnvelope.requestId` | The runtime `1..120` safe-token policy is explicit for acknowledgement and all five polling variants. |
| `scripts/generate-api-types.py` | `OPTIMIZATION_REQUEST_ID_RULE`, `OPTIMIZATION_SCHEMA_RULES`, `OPTIMIZATION_PROPERTY_NAMES`, `optimization_contract_mismatches`, `main` gate | Generated/OpenAPI assumptions, including exact request-ID bounds and safe characters, cannot drift silently. |
| `scripts/test_generate_api_types.py` | `test_optimization_terminal_vocabulary_and_similarity_projection_match_generated_contract`, `test_deliberate_optimization_decoder_contract_drift_is_rejected`, `test_optimization_submission_must_retain_caller_key_parameter`, `test_optimization_request_id_bounds_and_safe_characters_cannot_drift`, generated-output/status tests | Current contract plus deliberate too-short, too-long, stricter-maximum, and unsafe request-ID drift verification. |

The production component `frontend/src/lib/components/OptimizationWorkflow.svelte` remains unchanged: it calls the controller rather than the raw API and renders only controller state. `frontend/src/lib/api/generated.ts` remains unchanged and current; the OpenAPI change is contract-only and does not alter generated TypeScript or runtime behavior.

## Acceptance-criteria evidence

| Criterion | Result | Evidence |
|---|---|---|
| Exact `202` acknowledgement and exact `200` queued/processing/completed/failed/cancelled | PASS | Valid-variant table plus wrong-2xx tests. |
| Reject wrong envelope/status, unsafe request ID/poll URL, malformed UUID/date, missing/cross/additional fields | PASS | Acknowledgement and job adversarial tables. |
| Reject unsafe failure and invalid meals/macros/similarity/alternative cardinality | PASS | Fixed failure map and nested adversarial tables, including completed zero and over-three rejection and valid failed zero. |
| No malformed payload reaches store/rendering or crashes polling | PASS | Real strict client + controller integration leaves `job: null`, `alternatives: []`, and a fixed malformed-response failure. |
| Direct submission requires caller key | PASS | Required options compile check and runtime omission/weak-key no-I/O tests. |
| Ambiguity reuse, deliberate rotation, concurrent suppression | PASS | Store controller tests assert exact key sequences and one in-flight submit. |
| Secure-random failure | PASS | Missing/throwing/malformed provider table and controller no-I/O state test. |
| No browser persistence | PASS | Storage getters throw if touched; controller completes, changes diet, and disposes with zero accesses. Source/state audit finds no key field or storage call. |
| Generated drift and frontend tests | PASS | 16 Python drift tests, including eight request-ID mutation subtests; generated check, focused/full Bun tests, typecheck, build, and coverage all pass. |

## Commands run

All commands exited `0`.

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts` | PASS — 28 tests, 136 expectations. |
| `cd frontend && ... bun test` | PASS — 414 tests, 1,877 expectations across 37 files. |
| `cd frontend && ... bun test --coverage src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts` | PASS — client 97.78% functions / 95.00% lines; store 81.48% functions / 79.69% lines. |
| `cd frontend && ... bun test --coverage` | PASS — 414 tests; 93.32% functions / 94.01% lines overall; strict client 95.00% lines. Existing accepted Phase 07 coverage exceptions remain in `docs/implementation/04_OPEN.md`; Task 230 adds no exception. |
| `cd frontend && ... bun run typecheck` | PASS. |
| `cd frontend && ... bun run build` | PASS — Vite production build, 205 modules transformed. |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS — 16 tests, including eight deliberate optimization request-ID mutations across both envelopes. |
| `python3 scripts/generate-api-types.py --check` | PASS — generated API types current. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies; Task 230 remains `OPEN`. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS — document valid; one pre-existing explicitly ignored OAuth callback 2XX warning. |
| `git diff --check -- <Task 230 files>` | PASS. |

The root aggregate `scripts/check.py` was not run because it invokes Docker, browser, backend, and later Phase 07.01 gates outside this single frontend task. All Task 230 frontend, generated contract, OpenAPI, traceability, task-list, build, and coverage checks were run directly.

## Staleness and content fingerprints

### Pre-edit task-surface hashes

| File | SHA-256 before this completion pass |
|---|---|
| `frontend/src/lib/api/optimization-client.ts` | `76569c172a04a9970fcb3dfbc5092faa99555cfa8b43271c74f3231c603592a7` |
| `frontend/src/lib/api/optimization-client.test.ts` | `113ceb6699ab8ce2e4931156985df909f73ecbd899b5ad0df79097aa79f4e4d7` |
| `frontend/src/lib/stores/optimization.ts` | `a2e959c819daa0a0a1d1cf685e13c36926bcf24d9e55786205a3b46c3301019e` |
| `frontend/src/lib/stores/optimization.test.ts` | `abc554a72528b4524ced29c7d3ca5416b81b2b9219fba40eb28b51ef96d87d5f` |
| `scripts/generate-api-types.py` | `c6900a5a16e9e9a7504c1b54e9b2239e445a157651eb7f3a6c17eea549e75228` |
| `scripts/test_generate_api_types.py` | `e6b0036d19012b56126f2c2cf0659b1453b3d53af3dad8a4a32de191415c0d3f` |
| `api/openapi.yaml` | `b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585` |
| `frontend/src/lib/api/generated.ts` | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |

### F-230-01 repair baseline hashes

| File | SHA-256 before repair |
|---|---|
| `api/openapi.yaml` | `b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585` |
| `scripts/generate-api-types.py` | `5f58d4c1528f82415ce13c27948bddb391bccc332ea3f89ce02ce82ec2619a46` |
| `scripts/test_generate_api_types.py` | `fe2cee17abcc8f198a8f9de3fac3168afd8c8a8a1c6f481152681ab69d2a3649` |

### Final audited hashes

| File | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `a44ed4b1ed8bdaebba1510b1b18c5214c43051e77ba307ae1ddab2d1fa3dc6f4` |
| `docs/design/DESIGN-001.md` | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/architecture/ARCH-001.md` | `03fcbae9676ecf278e72a621c7fd9911d0c3dc6ee15eaebcec308d010cc76833` |
| `docs/design/01_TECH_STACK.md` | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/implementation/04_OPEN.md` | `c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527` |
| `api/openapi.yaml` | `392a3d531301a937b001bc7561b6e5cdef76a6a786d2073d739ab81cd1161c4a` |
| `frontend/src/lib/api/generated.ts` | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `frontend/src/lib/api/error-message-mapper.ts` | `7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e` |
| `frontend/src/lib/api/optimization-client.ts` | `c047e9ab5bd97ac381b8efa72d6d99fa362e4973c3b60d785348715bac2b4c09` |
| `frontend/src/lib/api/optimization-client.test.ts` | `e67cf00595ab34c40510f76a4a1b256cb570c4ec4c371c493d4ff8eedb79d280` |
| `frontend/src/lib/stores/optimization.ts` | `a2e959c819daa0a0a1d1cf685e13c36926bcf24d9e55786205a3b46c3301019e` |
| `frontend/src/lib/stores/optimization.test.ts` | `d1c25017a9a48f1fb576b549f20a1b8c6e46d5a9854a7c4d8758004a9f9a8efb` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `620e825cd23e258fee69ccb42899e00c01f2dc7a53df5d5b8e3d9cc3c6f00b33` |
| `scripts/generate-api-types.py` | `c2fdf54b8280eedf91b149ae9f94fd8d1f9a01d22095b57bb53309f792313acc` |
| `scripts/test_generate_api_types.py` | `3a1116c9165f67386e315ef380b52083c805f962e406e9a1282300d533b2813a` |

The unchanged generated/client/store hashes demonstrate preserved runtime behavior and concurrent input; only the OpenAPI request-ID contract, generator guard, mutation tests, and this preparation evidence changed for F-230-01. No task status or unrelated task row was altered.
