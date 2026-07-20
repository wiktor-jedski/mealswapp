# Review Evidence: Task 227 — Shared Runtime-Safe Client Error Mapping

~~~yaml
task_id: 227
phase: "07.01"
component: "DESIGN-017: ErrorMessageMapper"
static_aspect: "ErrorMessageMapper"
input_status: "OPEN (preserved because the user prohibited task-status edits)"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T15:54:00Z"
review_agent: "fresh independent owner review"
evidence_file: "docs/implementation/reviews/task-227-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus refreshed task-227-preparation.md manifest"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
code_review_skill_invocation_count: 1
relevant_language_guide: "TypeScript guide; unknown-boundary narrowing, strict typing, async error propagation, fixed safe output, and adversarial tests applied"
review_template_path: "docs/implementation/reviews/REVIEW_TEMPLATE.md"
review_template_available: false
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 07.01: centralize Daily Diet and optimization unknown-envelope parsing and approved code/status mapping in one runtime-safe client `ErrorMessageMapper`, including strict boolean retryability and bounded request IDs, while preserving the deliberate ownership-safe 403/404 Daily Diet projection.

**Design source:** `docs/design/DESIGN-017.md`, static aspect `ErrorMessageMapper`.

**Architecture source:** `docs/architecture/ARCH-017.md`, `ErrorMessageMapper`.

**Depends On:** 213 (PASSED in the current task list).

**Verification criteria:** Both clients import one mapper; malformed category, code, message, retryability, and request-ID values fall back safely; technical diagnostics never render; approved validation/auth/entitlement/dependency/rate-limit/timeout cases preserve the documented fixed policy; table-driven frontend tests and generated-contract drift checks pass.

**Template note:** The requested `docs/implementation/reviews/REVIEW_TEMPLATE.md` does not exist in this checkout (`sed` returned “No such file or directory”). The review uses the established evidence schema in `docs/implementation/reviews/task-223-review.md`, `task-224-review.md`, and `docs/implementation/reviewer-prompt.md`; the absence is recorded rather than creating or modifying an unrelated template.

## 2. Pre-Review Gates

- [x] Task 227 row was read from `docs/implementation/02_TASK_LIST.md`; it remains `OPEN`.
- [x] Task 213 dependency is `PASSED`.
- [x] `docs/implementation/preparation/task-227-preparation.md` was read in full and its current manifest was checked against the worktree.
- [x] `docs/design/01_TECH_STACK.md`, `docs/architecture/ARCH-017.md`, and `docs/design/DESIGN-017.md` were read.
- [x] The actual `HEAD` diff and all Task 227 untracked implementation/test files were inspected.
- [x] `code-review-skill` was invoked exactly once; its complete guidance and the TypeScript/security/error-handling references were read.
- [x] No production file, unrelated task, or task-list status was changed by this review.
- [x] The dirty worktree contains concurrent Phase 07 work; attribution uses the preparation manifest, current symbols, exact hashes, and the HEAD diff rather than assuming the entire worktree diff belongs to Task 227.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "Task status remains OPEN only because the user prohibited status edits; the independent review decision is recorded here."
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD` is `a4e31367485b03269e90b5607f2057c9568bb5b1`. The worktree is cumulatively dirty with Tasks 213–224 and other Phase 07 changes. Task 227 ownership was reconstructed from the refreshed preparation manifest, current hashes, the HEAD diff, current source, and the Task 227-specific untracked files.

Commands used to reconstruct the scoped surface:

~~~bash
git status --short
git rev-parse HEAD && git log -8 --oneline --decorate
git diff HEAD -- frontend/src/lib/api/daily-diet-client.ts frontend/src/lib/api/optimization-client.ts scripts/generate-api-types.py
rg -n 'mapErrorMessage|safeErrorFromSource|safeErrorForStatus|responseError|resolveCsrfToken|app_error_contract_mismatches' frontend scripts
sha256sum <current Task 227 files and source-of-truth files>
~~~

| Changed file | Task 227 attribution | Audited symbols/units |
|---|---|---|
| `frontend/src/lib/api/error-message-mapper.ts` | HIGH; Task 227 untracked production module in preparation manifest | `ErrorMessageScope`, `ErrorRule`, `DAILY_DIET_FALLBACKS`, `OPTIMIZATION_FALLBACKS`, `APPROVED_RULES`, `mapErrorMessage`, `approvedSourceError`, `fallbackFor`, `approved`, `rule`, `isObject`, `safeRequestId` |
| `frontend/src/lib/api/error-message-mapper.test.ts` | HIGH; Task 227 untracked focused suite | five mapper tests and all table fixtures |
| `frontend/src/lib/api/daily-diet-client.ts` | HIGH for mapper integration; prior strict-client changes excluded | `deleteDailyDiet`, `resolveCsrfToken`, `responseError`, removed local error-policy helpers |
| `frontend/src/lib/api/daily-diet-client.test.ts` | HIGH for Task 227 additions; prior client-contract tests retained | shared 500/503/504 mapping and fixed-message/request-ID regression |
| `frontend/src/lib/api/optimization-client.ts` | HIGH for mapper integration; concurrent Task 221 normalization changes excluded | `resolveCsrfToken`, `responseError`, removed local error-policy helpers |
| `frontend/src/lib/api/optimization-client.test.ts` | HIGH for Task 227 additions; concurrent terminal-normalization tests retained | audited submission-error table and malformed terminal/similarity regression |
| `scripts/generate-api-types.py` | HIGH for `AppError` drift additions; pre-existing response/unit generation changes excluded | `APP_ERROR_CATEGORIES`, `app_error_contract_mismatches`, `main` drift gate |
| `scripts/test_generate_api_types.py` | HIGH for Task 227 drift tests; prior response/contract tests retained | `test_runtime_error_contract_matches_generated_type_policy`, `test_malformed_retryability_and_category_contracts_are_rejected` |

The HEAD diff reports the tracked Task 227-related client/generator surface as 280 insertions and 115 deletions across five tracked files. The two mapper files and the generator test file are untracked, so their full contents were inspected directly and hashed. `frontend/src/lib/api/generated.ts` and `api/openapi.yaml` contain concurrent Phase 07 contract work; they were audited as dependencies but not attributed as Task 227 implementation changes.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Daily Diet and optimization use one shared mapper. | Imports and repository-wide search for removed local policy helpers. | PASS | `daily-diet-client.ts:2` and `optimization-client.ts:2` import `mapErrorMessage`; `safeErrorFromSource`, `safeErrorForStatus`, and duplicated validators are absent from both clients. |
| 2 | Unknown error envelopes are runtime-parsed. | Public input type and structural guards. | PASS | `mapErrorMessage` accepts `envelope: unknown`; `isObject` rejects null/arrays/primitives; `approvedSourceError` requires object fields of exact runtime types before rule selection. |
| 3 | Malformed category, code, message, and retryability values fall back safely. | Adversarial table and source audit. | PASS | `approvedSourceError:73-85` requires string category/code/message and boolean retryability; unapproved values use `fallbackFor`; server message text is never copied to output. |
| 4 | Request IDs are bounded and safe. | Boundary tests and regex audit. | PASS | `safeRequestId:107-109` permits only 1–120 ASCII correlation-token characters with an alphanumeric first character; tests cover exact length, oversize, empty, whitespace, newline, NUL, and precedence of error-level IDs. |
| 5 | Technical diagnostics never render. | Hostile text fixtures and fixed output audit. | PASS | Mapper tests pass stack, SQL/PostgreSQL, Redis, provider, URL, credential, oversized, and control-character text; every output message is a mapper-owned constant. |
| 6 | Approved category/code/status policy remains stable. | Fallback and approved-rule tables, client tests, OpenAPI source, and generated contract. | PASS | `DAILY_DIET_FALLBACKS:10-21`, `OPTIMIZATION_FALLBACKS:23-35`, and `APPROVED_RULES:37-56` cover current Daily Diet/optimization status policy; representative table-driven mapper/client cases pass. Valid booleans are deliberately retained; malformed/missing booleans select the status fallback. |
| 7 | Daily Diet 403 and 404 remain ownership-safe and indistinguishable. | Direct mapper and client regression. | PASS | `mapErrorMessage:64-67` bypasses source classification for Daily Diet 403/404 and returns the same security-safe projection; `daily-diet-client.test.ts:139-153` and mapper test lines 128-140 pass. Valid request IDs remain correlation metadata only. |
| 8 | Generated contract drift cannot silently weaken the runtime boundary. | Generator checks and deliberate mutation tests. | PASS | `app_error_contract_mismatches:109-126` enforces required fields, exact category enum, boolean `retryable`, and string `requestId`; the two Task 227 Python regressions reject deliberate retryability/category mutations. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified/removed | Callers or consumers | Tests/evidence |
|---:|---|---|---|---|---|---|
| 1 | `ErrorMessageScope` | exported union | `frontend/src/lib/api/error-message-mapper.ts:6` | added | Both scoped clients | Typecheck; client imports |
| 2 | `ErrorRule` | internal type | `error-message-mapper.ts:8` | added | Fallback and approved policy tables | Mapper coverage |
| 3 | `DAILY_DIET_FALLBACKS` | fixed policy table | `error-message-mapper.ts:10-21` | added | `fallbackFor` | Unknown/status and client tests |
| 4 | `OPTIMIZATION_FALLBACKS` | fixed policy table | `error-message-mapper.ts:23-35` | added | `fallbackFor` | Unknown/status and client tests |
| 5 | `APPROVED_RULES` | approved code/status table | `error-message-mapper.ts:37-56` | added | `approvedSourceError` | Table-driven mapper/client tests |
| 6 | `mapErrorMessage` | exported boundary function | `error-message-mapper.ts:59-71` | added | Daily Diet/optimization response and CSRF paths | Five focused mapper tests; 100% mapper line/function coverage |
| 7 | `approvedSourceError` | runtime validator/rule lookup | `error-message-mapper.ts:73-85` | added | `mapErrorMessage` | Malformed/approved mapper cases |
| 8 | `fallbackFor` | scoped status fallback | `error-message-mapper.ts:87-93` | added | `mapErrorMessage` | Known/unknown status cases |
| 9 | `approved` | table constructor | `error-message-mapper.ts:95-97` | added | `APPROVED_RULES` | Mapper coverage |
| 10 | `rule` | fixed-rule constructor | `error-message-mapper.ts:99-101` | added | Both fallback tables | Mapper coverage |
| 11 | `isObject` | runtime shape guard | `error-message-mapper.ts:103-105` | added | Mapper parsing | Primitive/array/null cases |
| 12 | `safeRequestId` | bounded token validator | `error-message-mapper.ts:107-109` | added | `mapErrorMessage` | Boundary cases and mapper coverage |
| 13 | `deleteDailyDiet` | client status fallback call site | `frontend/src/lib/api/daily-diet-client.ts:100-109` | modified | Daily Diet API consumers | Build/typecheck; client tests |
| 14 | `resolveCsrfToken` | propagated-error mapping call site | `daily-diet-client.ts:166-182` | modified | Daily Diet mutations | Build/typecheck; mapper integration audit |
| 15 | `responseError` | raw HTTP error boundary | `daily-diet-client.ts:184-192` | modified | `requestJson` | Fixed-message/request-ID client tests |
| 16 | `safeErrorFromSource`, `safeErrorForStatus`, `isErrorCategory`, `isSafeMessage`, `isSafeCode`, `isSafeRequestId` | duplicated local policy | `daily-diet-client.ts` prior `254-306` | removed | None after migration | Repository search confirms removal |
| 17 | `resolveCsrfToken` | propagated-error mapping call site | `frontend/src/lib/api/optimization-client.ts:105-117` | modified | Optimization submission | Build/typecheck; client integration audit |
| 18 | `responseError` | raw HTTP error boundary | `optimization-client.ts:132-140` | modified | `requestJson` | Submission error table |
| 19 | duplicated optimization error-policy helpers | local policy | `optimization-client.ts` prior `207-252` | removed | None after migration | Repository search confirms removal |
| 20 | `APP_ERROR_CATEGORIES` | generator source contract | `scripts/generate-api-types.py:103-106` | added/modified in concurrent generator surface | `app_error_contract_mismatches` | Python drift tests |
| 21 | `app_error_contract_mismatches` | generator validator | `scripts/generate-api-types.py:109-126` | added | `main`, drift tests | 9 Python tests; deliberate mutations fail |
| 22 | `main` AppError drift gate | generator entry point | `scripts/generate-api-types.py:1383-1410` | modified | `bun run check:api-types`, full frontend check | Current generated output check passes |
| 23 | `test_runtime_error_contract_matches_generated_type_policy` | Python test | `scripts/test_generate_api_types.py:19-22` | added | Generator contract | Pass |
| 24 | `test_malformed_retryability_and_category_contracts_are_rejected` | Python test | `scripts/test_generate_api_types.py:24-32` | added | Generator contract | Pass |
| 25 | `unknown Daily Diet envelopes use a fixed status fallback` | mapper test | `error-message-mapper.test.ts:7-26` | added | `mapErrorMessage` | Pass |
| 26 | `approved error codes retain fixed policy through one table-driven mapper` | mapper test | `error-message-mapper.test.ts:28-75` | added | `APPROVED_RULES` | Pass |
| 27 | `malformed fields and hostile text never cross the mapper boundary` | mapper test | `error-message-mapper.test.ts:77-118` | added | Runtime safety | Pass |
| 28 | `request IDs accept only bounded printable correlation tokens` | mapper test | `error-message-mapper.test.ts:120-126` | added | `safeRequestId` | Pass |
| 29 | `Daily Diet 403 and 404 remain ownership-safe and indistinguishable` | mapper test | `error-message-mapper.test.ts:128-141` | added | Ownership projection | Pass |
| 30 | shared Daily Diet 500/503/504 regression | frontend test | `daily-diet-client.test.ts:155-183` | added | `responseError` | Pass |
| 31 | fixed Daily Diet message and bounded request-ID regression | frontend test | `daily-diet-client.test.ts:185-205` | added | `responseError` | Pass |
| 32 | optimization submission 429/500/503/504 table | frontend test | `optimization-client.test.ts:159-187` | added | `responseError` | Pass |

`optimization-client.ts` also contains Task 221 terminal failure normalization and similarity validation changes in the current diff. Those symbols were audited only as dependency context and are excluded from this Task 227 decision.

~~~yaml
inventory_source_count: 32
audited_symbol_count: 32
inventory_complete: true
generated_groupings:
  - "Only the removed duplicated validator set is grouped because it is one intentionally deleted local policy surface; every active Task 227 production function and regression test is listed separately."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | Security boundary | Performance/API | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|
| `ErrorMessageScope` | Limits the public mapper boundary to the two Task 227 client domains. | Type-level scope selection is used at every client call site; an invalid runtime value still falls back to fixed optimization-safe output. | Does not carry user data. | Small union type. | Typecheck and client imports pass. | PASS |
| `ErrorRule` | Represents fixed output fields without a request ID. | Readonly rule values are built only from mapper-owned literals. | Prevents source message/cause fields from entering policy values. | Static type only. | Typecheck and mapper coverage. | PASS |
| `DAILY_DIET_FALLBACKS` | Defines fixed status policy for Daily Diet errors. | Covers current 400/401/403/404/409/422/429/500/503/504 statuses. | Ownership-safe 403/404 values are fixed. | Constant-size lookup table. | Unknown/status and client tests. | PASS |
| `OPTIMIZATION_FALLBACKS` | Defines fixed status policy for optimization errors. | Covers current 400/401/403/404/409/410/422/429/500/503/504 statuses. | Fixed category/code/message/retryability values only. | Constant-size lookup table. | Unknown/status and client tests. | PASS |
| `APPROVED_RULES` | Defines approved scope/status/code/category combinations. | Unlisted or category-mismatched codes select the status fallback. | Server code becomes output only after exact table lookup. | Constant-size keyed lookup. | Table-driven mapper/client tests; exhaustive-code coverage is optional gap. | PASS |
| `mapErrorMessage` | Returns an `AppError` from a scope, HTTP status, and unknown raw envelope. | Handles primitive/null/array bodies, missing error objects, malformed fields, approved rules, unknown statuses, and Daily Diet 403/404. | Never returns source message/category/code unless the code/category pair is approved; request IDs are separately bounded. | Constant-size table lookup and fixed output; no I/O. | Full mapper function/line coverage; only representative approved-code cases are direct tests. | PASS |
| `approvedSourceError` | Requires exact runtime field types and exact scope/status/code/category agreement. | Missing source or malformed message/retryability/category/code selects fallback; valid boolean retryability is retained by documented policy. | Source text is not copied. | One keyed lookup. | Hostile and malformed field cases pass. | PASS |
| `fallbackFor` | Selects the fixed status policy for the client scope. | Known statuses use fixed category/code/message/retryability; unknown statuses use fixed generic scope fallback. | No source data enters output. | Constant-time lookup. | 418 and audited status fallbacks pass. | PASS |
| `approved` | Expands one fixed base rule into approved code keys for a scope/status. | Only static code lists are used to construct entries. | No raw envelope data enters the rule table. | Initialization-only linear expansion. | Covered through mapper table. | PASS |
| `rule` | Constructs one fixed category/code/message/retryability rule. | No dynamic server values enter rule construction. | Fixed safe messages and codes. | Initialization only. | Covered through mapper table. | PASS |
| `isObject` | Distinguishes JSON object-shaped envelopes/errors from null, arrays, and primitives. | Rejects all non-object shapes before property access. | Prevents primitive/property confusion at the raw JSON boundary. | Constant-time shape check. | Null/array/primitive cases pass. | PASS |
| `safeRequestId` | Accepts only 1–120 allowed ASCII correlation-token characters with an alphanumeric first character. | Rejects empty, oversized, whitespace, newline, NUL, and unsafe punctuation/control values. Error-level ID takes precedence over top-level ID only after validation. | Prevents control/diagnostic text from being propagated as correlation metadata. | Bounded regex work. | Boundary cases pass; direct exhaustive allowed-character table is optional. | PASS |
| `deleteDailyDiet` | Uses the shared status fallback for the unreachable/non-success status branch retained by the current client surface. | Non-OK responses are mapped by `requestJson`; the explicit fallback remains safe for unexpected successful status handling. Exact success-status cleanup belongs to Task 228. | Fixed mapper output. | No extra I/O. | Full frontend check and client tests pass. | PASS |
| `daily-diet-client.resolveCsrfToken` | Propagated CSRF errors enter the same scoped mapper boundary. | Existing `DailyDietClientError` is preserved; other errors are converted using source status or 503 fallback. | Wrapped source AppError is revalidated as unknown data by mapper. | One mapping call. | Typecheck/full tests; no Task 227 regression. | PASS |
| `daily-diet-client.responseError` | Passes raw `response.json()` output to the mapper without generated casts. | Empty/malformed JSON leaves envelope unknown and selects status fallback; object/primitive/hostile bodies remain safe. | Removes the old top-level request-ID bypass and local regex policy. | One JSON parse and one mapper call. | Fixed message and bounded request-ID client regression passes. | PASS |
| `optimization-client.resolveCsrfToken` | Propagated CSRF errors enter the optimization-scoped mapper. | Existing client errors are preserved; unknown propagated errors map to safe optimization status fallback. | Source AppError is structurally revalidated by mapper. | One mapping call. | Typecheck/full tests; no Task 227 regression. | PASS |
| `optimization-client.responseError` | Passes raw HTTP error JSON to the shared mapper. | Empty/malformed JSON and unknown codes use safe status policy; approved 429/500/503/504 cases preserve fixed output. | Removes local message/code/request-ID trust. | One JSON parse and one mapper call. | Submission table passes. | PASS |
| Removed Daily Diet validators | No duplicated Daily Diet mapping policy remains in the client. | All old local paths are replaced by the shared boundary; unrelated network/success decoders remain outside this task’s scope. | Eliminates Daily Diet policy drift. | Removes six local helper implementations. | Repository search confirms no old helper declarations/callers. | PASS |
| Removed optimization validators | No duplicated optimization mapping policy remains in the client. | All old local paths are replaced by the shared boundary; unrelated network/success decoders remain outside this task’s scope. | Eliminates optimization policy drift. | Removes six local helper implementations. | Repository search confirms no old helper declarations/callers. | PASS |
| `APP_ERROR_CATEGORIES` | Pins the generated source category vocabulary used by the mapper. | Category additions/removals/reordering in OpenAPI are treated as contract drift. | Prevents generated/runtime category disagreement. | Static tuple comparison. | Current and deliberate category mutation tests. | PASS |
| `app_error_contract_mismatches` | OpenAPI AppError remains compatible with generated runtime-safe policy assumptions. | Missing schema, required-field drift, category drift, non-boolean retryability, and non-string request ID are rejected. | Prevents contract changes that weaken the mapper’s generated type boundary. | Static source scan, no runtime I/O. | Current and deliberate category/retryability mutations pass. | PASS |
| generator `main` AppError gate | Drift is rejected before generated output is checked or written. | Contract mismatch returns nonzero; current source then proceeds to generated-contract rendering/check. | Fails closed on source drift. | Small bounded source scan. | `bun run check:api-types`, Python suite, and full frontend check pass. | PASS |
| `unknown Daily Diet envelopes use a fixed status fallback` | Proves primitive/missing envelopes select fixed scope/status policy. | Covers null, array, primitive error body, known dependency status, and unknown status. | No raw body text is returned. | Deterministic table assertions. | Pass. | PASS |
| `approved error codes retain fixed policy through one table-driven mapper` | Proves representative approved category/code/status mappings and correlation ID retention. | Covers validation, auth, entitlement, dependency, rate-limit, and timeout representatives. | Hostile source message is ignored. | Deterministic table assertions. | Pass; exhaustive-code coverage is optional gap. | PASS |
| `malformed fields and hostile text never cross the mapper boundary` | Proves wrong category/code/message/retryability shapes and diagnostic strings fail closed. | Covers null, unapproved, wrong-type, stack, URL, credential, oversized, and control text. | Strong fixed-output assertion. | Deterministic adversarial table. | Pass. | PASS |
| `request IDs accept only bounded printable correlation tokens` | Proves empty, over-limit, whitespace, newline, and NUL IDs are discarded. | Covers exact 120-character accepted boundary and invalid values. | Prevents unsafe correlation metadata. | Constant-size assertions. | Pass; punctuation-boundary expansion optional. | PASS |
| `Daily Diet 403 and 404 remain ownership-safe and indistinguishable` | Proves source diagnostics cannot distinguish missing from forbidden resources. | Both statuses receive the same fixed projection and safe request ID. | Protects IDOR/cross-user confidentiality. | Deterministic pair assertion. | Pass. | PASS |
| shared Daily Diet 500/503/504 regression | Proves the client uses the shared fixed status policies. | Covers server, dependency, and timeout statuses with request IDs. | Server message is not propagated. | Deterministic fetch mock. | Pass. | PASS |
| fixed Daily Diet message and bounded request-ID regression | Proves oversized correlation ID and secret-bearing message do not cross the client boundary. | 500 response with hostile text and 121-character ID. | No diagnostic or oversized metadata reaches `DailyDietClientError`. | One mocked response. | Pass. | PASS |
| optimization submission 429/500/503/504 table | Proves optimization submission response errors use shared fixed policies. | Covers rate-limit, server, dependency, and timeout statuses with IDs. | Server message is not propagated. | Deterministic fetch mock. | Pass. | PASS |
| `test_runtime_error_contract_matches_generated_type_policy` | Proves current OpenAPI AppError contract is accepted. | Reads source and expects no mismatch. | Source contract remains aligned with runtime assumptions. | Deterministic Python test. | Pass. | PASS |
| `test_malformed_retryability_and_category_contracts_are_rejected` | Proves deliberate OpenAPI boolean/category weakening is rejected. | Mutates each declaration and checks the relevant mismatch. | Fails closed before generated output check/write. | Deterministic Python mutation test. | Pass. | PASS |

Mandatory audit conclusion: the shared mapper is the only Task 227 API-error parser for Daily Diet and optimization response/propagated-CSRF paths. Its fixed tables, runtime guards, safe-message policy, request-ID bound, and ownership projection satisfy the task contract. Task 228/230 success-envelope decoding and operation/key ownership remain explicit residual boundaries from the preparation document and are not reclassified as Task 227 findings.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| 🟢 [nit] | `frontend/src/lib/api/error-message-mapper.test.ts:28-75`; `error-message-mapper.ts:37-56` | `APPROVED_RULES` and approved mapping test | The production table contains 27 approved scope/status/code entries, while the direct mapper table exercises representative entries rather than every code variant. Client tests add more status coverage, but an accidental code-table deletion or a valid-boolean policy regression for an unrepresented code would not be caught directly. | Focused mapper coverage is 100% by lines/functions and all 20 focused tests pass, but the test fixture has six direct approved mappings and does not enumerate every approved code/status pair. | Optional hardening: expand the table-driven cases to cover every approved entry, including a valid boolean retryability value for each policy class. No Task 227 acceptance criterion fails and this does not block PASSED. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/error-message-mapper.test.ts src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts --coverage` | `frontend/` | 0 | PASS | 20 tests; `error-message-mapper.ts` 100% functions and 100% lines |
| `python3 -m unittest scripts/test_generate_api_types.py` | repository root | 0 | PASS | 9 generator drift/contract tests |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | `frontend/` | 0 | PASS | Generated API types current |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | `frontend/` | 0 | PASS | Typecheck, production build, and all 371 frontend tests; 371 pass, 0 fail |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 237 sequential tasks; Task 227 remains OPEN |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS | OpenAPI valid; one pre-existing warning for OAuth callback declaring only 302 and no 2XX response |
| `python3 scripts/validate-traceability.py` | repository root | 1 | KNOWN PRE-EXISTING FAILURE | Only concurrent Task 224 declarations in `backend/internal/queue/job_queue.go` at lines 74, 461, 586, 597, 757, 837, and 841; no Task 227 frontend/generator finding |
| `sha256sum <reviewed files>` | repository root | 0 | PASS | Current hashes recorded in Section 9 |

The repository-wide aggregate check was not run because its Docker/Chromium/local-stack gates exercise unrelated dirty-worktree Phase 07 work. The full frontend check, focused tests/coverage, generated drift suite, OpenAPI lint, task-list validator, traceability validator, and diff check provide the scoped evidence required here.

## 9. Files Inspected and Staleness Fingerprints

Current contents were hashed after the review commands. The preparation manifest’s current Task 227 hashes matched the implementation/test files below. The task list remains an existing shared dirty-worktree file and was inspected only for source/status; it was not edited.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `frontend/src/lib/api/error-message-mapper.ts` | shared runtime-safe mapper | optional exhaustive-policy test gap only | SHA-256 | `7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e` |
| `frontend/src/lib/api/error-message-mapper.test.ts` | mapper safety and policy regressions | optional exhaustive-policy test gap | SHA-256 | `aff0fd048b0034916a63774c225eaa609edc79585a31c55570819ddeb34c7df1` |
| `frontend/src/lib/api/daily-diet-client.ts` | Daily Diet mapper integration | no blocking/important finding | SHA-256 | `2e1bbad5ce856b0beb64f859bac99d462d970313a28e13f1615fa1b5daa3554c` |
| `frontend/src/lib/api/daily-diet-client.test.ts` | Daily Diet boundary regressions | no blocking/important finding | SHA-256 | `ebe875f56aa09aff558965de72285dda79feac677a4d7b15ba0c2e51634783d7` |
| `frontend/src/lib/api/optimization-client.ts` | optimization mapper integration | no blocking/important finding; concurrent Task 221 changes excluded | SHA-256 | `d1bc3b4944c6dc3ff5dc7bc2bd7fd31b5d3d948d5d0f1654c5f777f800569666` |
| `frontend/src/lib/api/optimization-client.test.ts` | optimization boundary regressions | no blocking/important finding; concurrent Task 221 tests excluded | SHA-256 | `fab6abd590530acaf2d735ffc8c35f787d880f10bc6c1d887e70f83332aad744` |
| `scripts/generate-api-types.py` | source/generated AppError drift gate | no blocking/important finding; concurrent generator changes excluded | SHA-256 | `17145445e24ccb0b1a807c251f771a52cac1cd659fced51ab7f5c31eb3e962c1` |
| `scripts/test_generate_api_types.py` | generator drift regressions | no blocking/important finding | SHA-256 | `a0a96fff54ac95d23aa56cd8ccedc4fbb8293d97a9e0f5253263aadd43417b21` |
| `api/openapi.yaml` | current AppError and endpoint source contract | dependency context; concurrent changes excluded | SHA-256 | `b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585` |
| `frontend/src/lib/api/generated.ts` | generated AppError consumer contract | dependency context; concurrent changes excluded | SHA-256 | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `docs/design/DESIGN-017.md` | ErrorMessageMapper design source | no contradiction | SHA-256 | `5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c` |
| `docs/architecture/ARCH-017.md` | error-handling architecture source | no contradiction | SHA-256 | `28cb8218c2abfc170565a674870f251cc1cbf17dd8e2726cdaf19cb5d806dd55` |
| `docs/design/01_TECH_STACK.md` | stack and test source | no contradiction | SHA-256 | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/implementation/04_OPEN.md` | originating frontend review action | Task 227 action matches implementation | SHA-256 | `c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527` |
| `docs/implementation/preparation/task-227-preparation.md` | scoped implementation evidence | current preparation checked | SHA-256 | `dd4b4e05d2c665fcc461ebbe964a8b127dafb55e1813ec9e240235697109944d` |
| `docs/implementation/02_TASK_LIST.md` | task source and status | status remains OPEN; not edited by review | SHA-256 | `3641b4740cc3c5e40b23740e2a090ff26d75ca9f6b19663d3c15b476c793b779` |
| `docs/implementation/reviewer-prompt.md` | fallback review instructions because requested template is absent | no finding | SHA-256 | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The requested docs/implementation/reviews/REVIEW_TEMPLATE.md was absent; neighboring task-223/task-224 review evidence and reviewer-prompt.md were used as the established schema."
  - "Concurrent Phase 07 modifications in the client, generator, OpenAPI, and generated-type files were separated from Task 227 using the preparation manifest and symbol-level diff audit."
~~~

## 10. Coverage and Exceptions

- [x] Focused mapper coverage command ran.
- [x] Mapper reports 100% function and line coverage.
- [x] Focused client and generator tests ran.
- [x] Full frontend typecheck, build, and test suite ran.
- [x] Uncovered or underrepresented policy cases were manually audited and are recorded as the optional Section 7 finding.
- [x] The repository traceability failure is pre-existing concurrent Task 224 queue work and is not a Task 227 exception.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "Bun focused --coverage output (stdout; no persistent artifact)"
observed_line_coverage: "100% for error-message-mapper.ts; 20 focused tests and 371 full frontend tests passed"
coverage_passed: true
~~~

Coverage finding: the new mapper itself is fully covered by lines/functions, but the approved table has more code/status combinations than the direct mapper fixture enumerates. This is a non-blocking test-hardening opportunity, not a failure of the task’s required focused coverage gate.

## 11. Negative and Regression Checks

- [x] Both clients consume the same exported mapper and no duplicate local error-policy helper remains.
- [x] Raw server messages, stack-like text, URLs, SQL/Redis/provider text, credentials, oversized text, and controls were challenged and never rendered.
- [x] Malformed primitive/object envelopes and wrong runtime field types fail closed to fixed status policy.
- [x] Strict boolean retryability behavior was tested for string/non-boolean values and source-audited for missing/null values.
- [x] Request-ID boundary, precedence, and ownership-safe Daily Diet 403/404 behavior were tested.
- [x] Generated AppError category/retryability drift mutations fail the generator contract check.
- [x] Full frontend typecheck/build and all 371 frontend tests pass.
- [x] No task-list status or unrelated production code was changed by this review.
- [x] The known traceability failure was isolated to concurrent Task 224 queue declarations; no Task 227 traceability failure was reported.

Findings: no blocking or important correctness, security, behavior, API, or lifecycle issue remains. One optional test-coverage note remains visible in Section 7.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those conditions are satisfied.

~~~yaml
decision: "PASSED"
reason: "The shared mapper and both client integrations satisfy Task 227’s runtime-safety, fixed-message, request-ID, approved-policy, ownership-projection, and generated-contract drift criteria. Current focused/full frontend and Python evidence passes; the only remaining note is optional exhaustive policy-table coverage."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Keep Task 227 OPEN as instructed. Optionally expand the approved-rule table tests before the Phase 07.01 aggregate gate; do not treat the concurrent Task 224 traceability failure as Task 227 scope."
~~~

## 13. Repair Context

Not applicable to the current PASSED decision. No prior Task 227 review or blocking repair was present in this checkout. The originating `docs/implementation/04_OPEN.md:319` action was rechecked against the current shared mapper, both client integrations, adversarial tests, and generated AppError drift tests.
