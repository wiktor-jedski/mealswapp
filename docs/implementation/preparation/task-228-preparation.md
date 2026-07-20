# Task 228 Preparation — Strict Daily Diet Client and Retry-Stable Create

## Scope and worktree safety

- Authoritative task: Phase 07.01 Task 228, `DESIGN-001: SearchView`, status `OPEN`.
- This repair is limited to findings F-228-01, F-228-02, and F-228-03 from `docs/implementation/reviews/task-228-review.md`: UUID-provider output validation, complete Daily Diet contract drift enforcement, bounded response acquisition/decoding, focused tests, and this evidence file.
- The worktree already contained cumulative concurrent Phase 07.01 changes. No unrelated path was cleaned, reverted, staged, or rewritten. Task statuses and unrelated task rows were not modified.
- The previous preparation file described Task 230 optimization work under the wrong task number. It has been replaced with evidence for the current authoritative Task 228 row.

## Sources read

| Source | Relevant contract | SHA-256 |
|---|---|---|
| `docs/implementation/02_TASK_LIST.md` | Task 228 description, dependencies 216/227, exact acceptance criteria, and preserved `OPEN` status | `4657500ac6ef4628e9aa1c11fe0db5504f8607e35e084999dfdf50f8e9e53957` |
| `docs/design/DESIGN-001.md` | `SearchView` orchestration, authenticated-action routing, runtime error projection, and memory/storage responsibilities | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/implementation/04_OPEN.md` | strict Daily Diet decoding/status action, retry-stable operation-key action, and shared mapper action | `c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527` |
| `api/openapi.yaml` | canonical list/get/create/replace/delete statuses; envelope, UUID, date-time, name, entry, quantity, unit, position, macro, and idempotency bounds | `b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585` |
| `frontend/src/lib/api/generated.ts` | generated Daily Diet DTOs and canonical request builders | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `frontend/src/lib/api/error-message-mapper.ts` | inherited Task 227 shared runtime-safe error mapping boundary | `7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e` |
| `docs/implementation/reviews/task-228-review.md` | F-228-01 malformed secure-random output, F-228-02 incomplete type/reference/idempotency drift guards, and F-228-03 post-buffer body limit | `35c0aacc72a5c34df4ca6a8ff785ff9a9a090a1292007973a812ae6b9de8efdf` |
| `docs/implementation/preparation/task-227-preparation.md` and `docs/implementation/reviews/task-227-review.md` | shared mapper ownership and Task 228 residual boundary | read; unchanged |

## Implemented surface

| File | Exact Task 228 evidence | SHA-256 |
|---|---|---|
| `frontend/src/lib/api/daily-diet-client.ts` | canonical `listDailyDiets`, `getDailyDiet`, `createDailyDiet`, `replaceDailyDiet`, and `deleteDailyDiet`; exact 200/201/204 policy; exact object reconstruction; `generateDailyDietIdempotencyKey` validates raw output through canonical lowercase RFC 4122 UUIDv4 `randomUuidV4` before interpolation; `boundedText` streams through `ReadableStreamDefaultReader`, counts bytes before buffering, cancels above 5 MiB, and performs fatal UTF-8 decode before JSON parsing | `35d60162f1f5e9a3db350b95d93e6b2c894e9926be5305b406a2815e9ad03db6` |
| `frontend/src/lib/api/daily-diet-client.test.ts` | all existing exact response/decoder behavior plus canonical valid UUIDv4 acceptance, null/undefined/string/object/nil/non-canonical provider rejection before I/O, and no-header chunked oversize cancellation on the first over-limit chunk | `72ae560716e8abf580cc173e9f603f238de45029f7ce7170cda659d6960cd941` |
| `frontend/src/lib/stores/daily-diet.ts` | closure-local create intent and key; request fingerprint ownership; lost-response replay; active-promise suppression; key rotation on changed/discarded intent; clearing after success and on `clear`; no key in Svelte state or browser storage | `3e5c77197b8bd5c2c6911d821c3ea07f89254a29dc468f51894f1ffcff031b23` |
| `frontend/src/lib/stores/daily-diet.test.ts` | same-key lost-response replay; pending-click suppression; edit/success/clear rotation; secure-random failure before API I/O; explicit zero-access browser-storage fixture; identity lifecycle key rotation | `f59880d99e076097c1304068badaa09984765e6041ffc858646a65828c24bae5` |
| `frontend/src/lib/components/DailyDietCollection.svelte` | every editable create-intent transition discards retry ownership; save is suppressed while creating; logout and account change reset component draft and call store `clear` | `1428689f367cd04f32e562f132c39b79f609fa0ae7fa9fd104b69e9b20d8ca04` |
| `frontend/src/lib/components/DailyDietCollection.test.ts` | edit hooks, pending suppression, and complete identity-owned draft reset assertions | `c0869a7ec40af0806231e72bf900d320d60ca39a3742f302fb5e43a48ab6cf65` |
| `scripts/generate-api-types.py` | exact audited Daily Diet response matrix; `DAILY_DIET_SCHEMA_RULES` exact scalar/ref/bound rules; `DAILY_DIET_PROPERTY_NAMES` exact decoded property sets; `daily_diet_contract_mismatches` fail-closed checks; and `operation_block` enforcement that Daily Diet create retains exactly one `IdempotencyKey` parameter reference before generated output is compared or written | `c6900a5a16e9e9a7504c1b54e9b2239e445a157651eb7f3a6c17eea549e75228` |
| `scripts/test_generate_api_types.py` | checked-in generated output equality plus deliberate status, operation, idempotency type/reference, canonical-unit type, every decoded Daily Diet scalar/ref/shape family, unknown property, collection item, and envelope request/data drift rejection | `e6b0036d19012b56126f2c2cf0659b1453b3d53af3dad8a4a32de191415c0d3f` |

## Review repair evidence

| Finding | Exact repaired symbols | Regression evidence |
|---|---|---|
| F-228-01 | `generateDailyDietIdempotencyKey`, `randomUuidV4` in `daily-diet-client.ts`; malformed-provider test in `daily-diet-client.test.ts` | A canonical lowercase UUIDv4 remains accepted. Missing/throwing providers and null, undefined, non-UUID string, object, nil UUID, and uppercase/non-canonical output all throw `secure_random_unavailable`; fetch call count remains zero. |
| F-228-02 | `DAILY_DIET_SCHEMA_RULES`, `DAILY_DIET_PROPERTY_NAMES`, `daily_diet_contract_mismatches`, `operation_block` in `generate-api-types.py`; deliberate mutation tests in `test_generate_api_types.py` | The guard rejects Daily Diet success-status drift; `IdempotencyKey` type/bounds/header/reference drift; envelope/request-ID/data drift; entry/unit/collection/macro references; scalar types, formats, bounds, and unknown decoded properties. The guard runs before check/write output. |
| F-228-03 | `MAX_RESPONSE_BYTES`, `boundedText`, and all success/error/delete callers in `daily-diet-client.ts`; chunked-stream test in `daily-diet-client.test.ts` | Declared oversize is rejected before reads. Undeclared chunked input is counted while reading, canceled immediately when cumulative bytes exceed 5 MiB, never passed to `JSON.parse`, and bounded bytes are decoded with fatal UTF-8 handling. |

## Runtime decisions

- Every successful JSON response is treated as hostile input. The client reconstructs a fresh `DailyDiet` only after exact-key and nested-field validation; no cast or shallow normalization can expose malformed server data.
- List/get/replace accept only HTTP 200, create only HTTP 201, and delete only an empty HTTP 204. Any other successful status is a bounded `malformed_daily_diet_response`.
- Non-success HTTP responses use the Task 227 `ErrorMessageMapper`; malformed, empty, or oversized error documents use its fixed status policy. CSRF acquisition failures are projected through the same mapper policy.
- Create requires a caller-supplied 8–255 visible-ASCII key before CSRF or Daily Diet network I/O. The API client never generates a replacement key implicitly.
- The controller owns one key in closure memory for one request fingerprint. An ambiguous failure retains it, retry reuses it, an active request promise suppresses parallel clicks, edits discard it, success clears it, and logout/account `clear` removes it. No idempotency key enters Svelte state, local storage, or session storage.
- Key generation uses only browser `crypto.randomUUID()`. Its raw return must be a canonical lowercase RFC 4122 UUIDv4, including version and variant bits, before the `daily-diet-` prefix is formed. Missing, throwing, or malformed providers fail closed with fixed safe error data and perform no create request; there is no clock or pseudo-random fallback.
- JSON and mapped-error bodies are acquired through one 5 MiB streaming byte limit. A trustworthy oversized declaration is rejected before reading; chunked or falsely small declarations remain bounded by cumulative bytes, trigger cancellation at the first over-limit chunk, and only bounded bytes reach fatal UTF-8 decoding and `JSON.parse`.

## Verification evidence

| Command | Result |
|---|---|
| `cd frontend && bun test src/lib/api/daily-diet-client.test.ts src/lib/stores/daily-diet.test.ts src/lib/components/DailyDietCollection.test.ts` | PASS — 27 tests, 140 expectations |
| `cd frontend && bun run typecheck` | PASS |
| `cd frontend && bun test` | PASS — 390 tests |
| `cd frontend && bun test --coverage` | PASS — 390 tests, 1,741 expectations; aggregate 93.54% lines; Daily Diet client 95.22%; Daily Diet store 94.95% |
| `cd frontend && bun run build` | PASS |
| `cd frontend && bun run check:api-types` | PASS — generated API types current |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS — 13 tests, including exhaustive deliberate Daily Diet decoder/idempotency-contract drift |
| `npx --no-install redocly lint api/openapi.yaml` | PASS — valid contract; one pre-existing ignored OAuth 302 warning |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies |
| `git diff --check -- <Task 228 repair files>` | PASS |
| search for `fetchDailyDiets`, `Math.random`, `Date.now`, `localStorage`, `sessionStorage`, and bypassed fallback parameters in Task 228 production files | PASS — none present; only the intentional safe-error fallback comment matches `fallback` |

## Status and residuals

- No task status was changed, per assignment. Task 228 remains `OPEN` in the authoritative table.
- No OpenAPI or generated TypeScript contract was changed by this repair; the implementation consumes the current authoritative contract and adds checks that fail if its decoder assumptions drift.
- Repository policy targets 100% phase coverage. The accepted Phase 07 frontend coverage exception remains recorded in `docs/implementation/04_OPEN.md`; this task improves the touched client/store coverage but does not alter that project-owner disposition.
- The aggregate `scripts/check.py` was not run because Task 228 is a frontend client/store retry and its relevant frontend, contract, OpenAPI, traceability, build, and coverage checks all ran directly. Docker/browser/backend phase gates and Phase 07.01 status completion remain outside this single-task retry.
