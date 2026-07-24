# Review Evidence: Task 253 — Admin and External-Data OpenAPI Contract

```yaml
task_id: 253
component: "Admin and External-Data OpenAPI Contract"
static_aspect: "DESIGN-009: AdminController"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T18:05:00Z"
review_agent: "Codex fresh independent final review"
evidence_file: "docs/implementation/reviews/task-253-review.md"
baseline_ref: "HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69 plus current task-253 preparation/review fingerprints"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "OpenAPI 3.1/Redocly, TypeScript generated contracts, and HTTP/API security guidance"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 08 extends `api/openapi.yaml` with custom-item, filter-option, admin authorization, external search, curated import, manual item, classification, user administration, idempotency, warning, audit-safe error, and pagination contracts, then regenerates the frontend API types.

**Depends On:** 239, 241, 248, 249, 250, 251, 252; all are currently `PASSED`.

**Testing Coverage Exceptions:** None.

**Verification Criteria:** OpenAPI lint and the frontend generated-type drift check pass; route/status inspection covers 200/201/202/204 plus applicable 400/401/403/404/409/422/429/500/503/504 responses, cookie auth, CSRF, `Idempotency-Key`, retry metadata, bounded warning enums, exact DTO ownership boundaries, and no raw external payload or audit snapshot is client-visible.

The current task row is still `PREPARED` at `docs/implementation/02_TASK_LIST.md:260`. This review changed only this evidence file; it did not edit production code or the task list.

The review applied the requested OpenAPI, TypeScript, and API-security guidance once. Source/runtime context was checked against `ARCH-009`, `DESIGN-009`, the Phase 08 requirements, and the existing admin gateway/controllers.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`; all seven dependencies are currently `PASSED`.
- [x] The preparation report claims completion and records the repaired F-253-003 surface.
- [x] A task-specific baseline/diff is available: fixed `HEAD`, current preparation fingerprints, previous rejected review, and current worktree diff.
- [x] `code-review-skill` was invoked exactly once and its TypeScript and security guidance was read and applied.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code changes and did not edit the task list.

```yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD 81ca40ce` was confirmed. The shared worktree contains concurrent Phase 08 changes, including unrelated task-list status transitions and prior API work. Task-owned scope was reconstructed from the task row, preparation report, previous F-253-003 review, current `git diff`, current untracked typecheck fixture, and current fingerprints. The task-owned contract surface is the Phase 08 portion of `api/openapi.yaml`, the Phase 08 generator checks/templates, the generator tests, `frontend/src/lib/api/generated.ts`, and `frontend/src/lib/api/generated.phase08-typecheck.ts`. Runtime Go files were inspected as unchanged consumers and parity evidence, not attributed as Task 253 modifications.

Commands used to reconstruct the diff:

```bash
git status --short
git rev-parse --short HEAD
git diff --stat HEAD -- api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py frontend/src/lib/api/generated.ts
git diff --unified=0 HEAD -- api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py frontend/src/lib/api/generated.ts
git ls-tree HEAD -- frontend/src/lib/api/generated.phase08-typecheck.ts
sha256sum api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py frontend/src/lib/api/generated.ts frontend/src/lib/api/generated.phase08-typecheck.ts docs/implementation/preparations/task-253.md docs/implementation/02_TASK_LIST.md
```

Pre-existing dirty-worktree changes and exclusions:

The worktree already contains Tasks 238–252 backend, database, frontend, API, preparation, and task-list changes. The current task-list hash differs from the preparation hash because earlier task statuses were changed from `OPEN` to `PASSED`; the Task 253 row text/status and dependency status were re-read and are current. Concurrent API changes outside the Phase 08 Admin and External-Data contract, such as existing billing paths, were excluded from Task 253 attribution. No unrelated file was reverted, staged, reformatted, or overwritten.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `api/openapi.yaml` | Current Phase 08 contract diff plus repaired `CustomItemEnvelope` strictness | MEDIUM | Phase 08 paths, parameters, responses, strict envelopes, safe DTOs, warning/pagination/ownership bounds |
| `scripts/generate-api-types.py` | Current generator diff plus F-253-003 repair | HIGH | Phase 08 response map, nine-envelope inventory, classification rule, source drift guard, generated template, main gate |
| `scripts/test_generate_api_types.py` | Current Phase 08 contract and source-mutation tests | HIGH | Eight Phase 08-focused test methods, including the parameterized nine-envelope mutation test |
| `frontend/src/lib/api/generated.ts` | Generated artifact from current OpenAPI source | HIGH | `ErrorEnvelope`, `OkEnvelope`, nine Phase 08 aliases, safe DTO projections |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | Untracked compile-time regression fixture recorded by preparation evidence | HIGH | Nine strict-envelope assertions and missing-data/non-`ok` negative assertions |

The task-owned implementation scope is distinguishable with MEDIUM confidence because the API file is shared by concurrent phase work; all reviewed Task 253 symbols and acceptance boundaries are individually attributable and fingerprinted.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | OpenAPI source is valid. | Redocly lint | PASS | `npx --no-install redocly lint api/openapi.yaml` exited 0; only the existing intentional OAuth callback 302-only warning remains. |
| 2 | Generated frontend API output is current. | Generator `--check` and exact generated-output test | PASS | `python3 scripts/generate-api-types.py --check` exited 0; `test_generated_output_drift_is_detected` passes. |
| 3 | Required Phase 08 operations and success statuses exist. | Source operation matrix and generator response guard | PASS | All 17 operations are present with exact 200, 201, 202, or 204 success statuses as applicable. |
| 4 | Applicable client-error and dependency statuses are complete. | Per-operation status matrix and runtime status inspection | PASS | Exact 400/401/403/404/409/429/500/503/504 sets match; 422 exists elsewhere in the API for search, while Phase 08 validation is intentionally 400. Global required-status coverage includes 422. |
| 5 | Cookie authentication and admin authorization boundaries are explicit. | OpenAPI security plus runtime gateway/controller tests | PASS | Admin and private custom-item routes declare `cookieAuth`; runtime `RequireAdmin` and `requireAdminRole` enforce verified admin role; public filter options remain unauthenticated by design. |
| 6 | CSRF protects every Phase 08 state-changing route. | OpenAPI mutation security and middleware-order tests | PASS | Every Phase 08 POST/PUT/DELETE mutation declares `csrfHeader`; router middleware is auth, role, CSRF, validation, rate limiting, audit, then handler; focused HTTP tests pass. |
| 7 | Idempotency and retry behavior is represented and bounded. | Header/component inspection and route/service tests | PASS | Custom-item and admin-item creates require `Idempotency-Key`; curated imports use the conditional optional key; exact replay and changed-body conflict behavior is covered; `429` responses reference `TooManyRequests` with positive `Retry-After`. |
| 8 | Warning enums and pagination are bounded. | Schema and parameter bounds plus mutation tests | PASS | Provider warning vocabulary is closed and bounded; candidate/warning arrays, filter options, classification collections, user pages, cursors, query pages, and limits have finite bounds. |
| 9 | DTO ownership and global/private boundaries are exact. | OpenAPI schema inspection, generated types, adversarial runtime tests | PASS | Private custom items omit owner fields and are session-scoped; admin items/classifications are global projections; private/global isolation tests pass. |
| 10 | Raw provider payloads and audit snapshots are not client-visible. | Closed DTO/error schema inspection and generated forbidden-field scan | PASS | `ExternalCandidate`, import/admin/user/error projections contain no raw provider, audit snapshot, before/after, owner, password, or token fields; generated scan finds none of `rawPayload`, `auditSnapshot`, `ownerId`, `passwordHash`, or `accessToken`. |
| 11 | Classification name schema matches runtime max length and normalization. | Source rule check plus Go normalization tests | PASS | Both classification schemas specify max length 120 and the exact NFC/whitespace/Unicode policy; runtime validates normalized rune count at 120 and rejects 121/control/invalid UTF-8 cases. |
| 12 | All nine Phase 08 success envelopes remain strict at the source boundary. | Source mutation regression and TypeScript fixture | PASS | `CustomItemEnvelope`, `FilterOptionsEnvelope`, `ExternalSearchEnvelope`, `CuratedImportEnvelope`, `AdminItemEnvelope`, `AdminClassificationEnvelope`, `AdminClassificationCollectionEnvelope`, `AdminUserPageEnvelope`, and `AdminDeletionRetryEnvelope` each require exactly `status`, `requestId`, and `data`, are closed objects, and use `status.const: ok`. The independent probe ran 54 mutations: 9 envelopes × 6 mutations, with 0 misses. |

### Phase 08 path/status/auth matrix

| Method and path | Required statuses | Auth/CSRF/retry boundary |
|---|---|---|
| POST `/api/v1/custom-items` | 201, 400, 401, 403, 409, 500, 503, 504 | cookie + CSRF + required idempotency |
| GET `/api/v1/custom-items/{itemId}` | 200, 400, 401, 404, 500, 503, 504 | cookie + owner-safe lookup |
| PUT `/api/v1/custom-items/{itemId}` | 200, 400, 401, 403, 404, 409, 500, 503, 504 | cookie + CSRF + owner-safe mutation |
| DELETE `/api/v1/custom-items/{itemId}` | 204, 400, 401, 403, 404, 500, 503, 504 | cookie + CSRF + no-content response |
| GET `/api/v1/search/filter-options` | 200, 400, 429, 500, 503, 504 | public read + bounded retry metadata |
| GET `/api/v1/admin/external-search` | 200, 400, 401, 403, 429, 500, 503, 504 | cookie + runtime admin role + bounded retry |
| POST `/api/v1/admin/imports` | 201, 400, 401, 403, 409, 429, 500, 503, 504 | cookie + admin + CSRF + conditional idempotency |
| POST `/api/v1/admin/items` | 201, 400, 401, 403, 409, 429, 500, 503, 504 | cookie + admin + CSRF + required idempotency |
| GET `/api/v1/admin/items/{itemId}` | 200, 400, 401, 403, 404, 429, 500, 503, 504 | cookie + admin + global projection |
| PUT `/api/v1/admin/items/{itemId}` | 200, 400, 401, 403, 404, 409, 429, 500, 503, 504 | cookie + admin + CSRF + retry |
| DELETE `/api/v1/admin/items/{itemId}` | 204, 400, 401, 403, 404, 409, 429, 500, 503, 504 | cookie + admin + CSRF + no-content |
| GET `/api/v1/admin/classifications` | 200, 400, 401, 403, 429, 500, 503, 504 | cookie + admin + bounded collection |
| POST `/api/v1/admin/classifications/{classification}` | 201, 400, 401, 403, 409, 429, 500, 503, 504 | cookie + admin + CSRF + kind enum |
| PUT `/api/v1/admin/classifications/{classification}` | 200, 400, 401, 403, 404, 409, 429, 500, 503, 504 | cookie + admin + CSRF + UUID boundary |
| DELETE `/api/v1/admin/classifications/{classification}` | 204, 400, 401, 403, 404, 409, 429, 500, 503, 504 | cookie + admin + CSRF + in-use conflict |
| GET `/api/v1/admin/users` | 200, 400, 401, 403, 404, 429, 500, 503, 504 | cookie + admin + bounded exact/page lookup |
| POST `/api/v1/admin/users/{userId}/deletion-requests/{requestId}/retry` | 200, 400, 401, 403, 404, 409, 429, 500, 503, 504 | cookie + admin + CSRF + legal retry state |

## 5. Changed-Symbol Inventory

The declarative OpenAPI units are grouped by boundary. Generated artifacts are grouped only where their output is produced as one exact artifact and checked byte-for-byte against the generator template. Each added test method is listed separately.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `PHASE08_OPERATION_RESPONSES` | response policy constant | `scripts/generate-api-types.py:119-137` | added | `phase08_contract_mismatches`, generator main | route/status mutation tests |
| 2 | `PHASE08_SUCCESS_ENVELOPES` | envelope inventory constant | `scripts/generate-api-types.py:139-149` | added/repaired | source guard, generated alias tests | nine-envelope parameterized mutation test |
| 3 | `ADMIN_CLASSIFICATION_NAME_RULE` | runtime-parity contract constant | `scripts/generate-api-types.py:151-158` | added | Phase 08 source guard | classification parity/mutation tests |
| 4 | `custom_item_contract_mismatches` | source contract function | `scripts/generate-api-types.py:387-401` | added | generator main | custom-item projection tests |
| 5 | `phase08_contract_mismatches` | source contract function | `scripts/generate-api-types.py:504-566` | added/repaired | generator main and tests | route, security, warning, envelope, privacy mutations |
| 6 | `main` Phase 08/custom-item gate calls | generator entrypoint | `scripts/generate-api-types.py:2042-2054` | modified | CLI drift check/generation | generator check |
| 7 | Phase 08 route operations | OpenAPI declarative route units | `api/openapi.yaml:627-1377` | added | Fiber route consumers and generated API consumers | response/security matrix and HTTP tests |
| 8 | Phase 08 parameters, security, responses | OpenAPI declarative boundary units | `api/openapi.yaml:1398-1663` | added/modified | route validation, auth, CSRF, retry consumers | source mutation tests and runtime tests |
| 9 | Nine strict success envelope schemas | OpenAPI behavioral schemas | `api/openapi.yaml:1716-2120,2596-2610` | added/repaired | response writers and generated aliases | 54 source mutations and type fixture |
| 10 | Phase 08 safe DTO schemas and bounds | OpenAPI behavioral schemas | `api/openapi.yaml:1716-2120,2504-2570,2967-2981` | added | admin/custom/import/filter consumers | warning, pagination, ownership/privacy tests |
| 11 | `GENERATED` `ErrorEnvelope`/`OkEnvelope` and nine aliases | generator output template | `scripts/generate-api-types.py:647-1892` | added | generated frontend API contract | generated output drift and typecheck |
| 12 | Phase 08 generated artifact | generated TypeScript types | `frontend/src/lib/api/generated.ts:23-1266` | added | frontend API clients and future admin UI | generator check, 459 Bun tests |
| 13 | `Phase08SuccessEnvelopeTypeChecks` | TypeScript compile-time fixture | `frontend/src/lib/api/generated.phase08-typecheck.ts:1-40` | added | frontend typecheck | strict positive and negative assignments |
| 14 | `test_phase08_routes_security_statuses_and_safe_dtos_match_generated_types` | Python regression test | `scripts/test_generate_api_types.py:21-31` | added | generator contract gate | focused unittest |
| 15 | `test_phase08_security_or_warning_drift_is_rejected` | Python mutation test | `scripts/test_generate_api_types.py:34-40` | added | generator contract gate | CSRF and warning mutation cases |
| 16 | `test_phase08_classification_names_match_runtime_normalization` | Python parity test | `scripts/test_generate_api_types.py:42-46` | added | generator contract gate | focused unittest |
| 17 | `test_phase08_classification_name_drift_is_rejected` | Python mutation test | `scripts/test_generate_api_types.py:48-59` | added | generator contract gate | max-length/description mutation cases |
| 18 | `test_phase08_generated_success_envelopes_are_strict` | Python generated-contract test | `scripts/test_generate_api_types.py:61-67` | added | generated type consumers | focused unittest |
| 19 | `test_phase08_source_success_envelopes_cannot_be_weakened` | Python source mutation test | `scripts/test_generate_api_types.py:69-90` | added/repaired | source drift gate | 54 independent mutations |
| 20 | `test_custom_item_name_and_classification_contracts_match_generated_types` | Python DTO projection test | `scripts/test_generate_api_types.py:92-105` | added | custom-item generated consumers | focused unittest |
| 21 | `test_custom_item_name_or_parent_projection_drift_is_rejected` | Python DTO mutation test | `scripts/test_generate_api_types.py:107-114` | added | custom-item source gate | focused unittest |
| 22 | `OperationResponseDriftTest` Phase 08 test unit | Python test class | `scripts/test_generate_api_types.py:18-114` | modified | generator contract suite | 5 focused methods pass |

```yaml
inventory_source_count: 22
audited_symbol_count: 22
inventory_complete: true
generated_groupings:
  - "The Phase 08 OpenAPI route/parameter/schema units are declarative boundary groups; each group has its own audit row and direct source matrix evidence."
  - "The generated TypeScript Phase 08 artifact is one generated output unit; its source template, byte-for-byte drift check, compile-time fixture, and frontend typecheck are audited together."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `PHASE08_OPERATION_RESPONSES` | Enumerates all 17 required operations and exact status sets. | Missing route, extra status, and wrong status are reported by comparison. | N/A — immutable in-process policy data. | Does not authorize by itself; paired with runtime gateway checks. | Constant-size set comparisons. | Clear route/method map; no duplicate operation keys. | Route matrix, focused tests, and direct probe cover every entry. | PASS |
| `PHASE08_SUCCESS_ENVELOPES` | Names exactly the nine strict success schemas, including `CustomItemEnvelope` and `AdminClassificationEnvelope`. | Missing schema and malformed schema are detected. | N/A — immutable inventory. | Defines source-to-consumer success boundary. | Nine bounded schema checks. | Tuple is explicit and easy to audit. | 54/54 source mutations detected. | PASS |
| `ADMIN_CLASSIFICATION_NAME_RULE` | Encodes the exact 120-code-point post-NFC runtime rule. | Length, description, whitespace, Unicode, and punctuation drift are rejected. | N/A — source comparison only. | Prevents schema/runtime validation disagreement. | Constant string match; no I/O. | Single shared rule avoids duplicated literals in tests. | Two schema variants and max-length/description mutations pass. | PASS |
| `custom_item_contract_mismatches` | Requires safe custom-item name pattern and hierarchy-free classification summaries. | Missing pattern, extra classification property, or wrong projection count is reported. | N/A — pure source inspection. | Prevents owner/audit hierarchy leakage into frontend DTOs. | Linear bounded text scans. | Small pure helper with typed list output. | Current and deliberate name/parent mutations pass. | PASS |
| `phase08_contract_mismatches` | Enforces strict envelopes, routes/statuses, auth/CSRF, idempotency, retry, warnings, status coverage, privacy, and classification parity. | Missing blocks, malformed properties, extra statuses, missing headers, and forbidden fields fail closed with actionable messages. | N/A — pure deterministic source inspection; no shared mutable state. | Checks source declarations while runtime role/CSRF enforcement remains in gateway consumers. | Linear scans over bounded source text; no subprocess or network. | Readable staged checks; no unnecessary public API. | Focused tests plus independent 54-mutation and matrix probes pass. | PASS |
| `main` Phase 08/custom-item gate calls | CLI must reject drift before generated output comparison or write. | Any mismatch returns exit 1; current valid source reaches generated check. | No file write under `--check`; generation write is existing intended behavior. | Source contract is checked before frontend artifact acceptance. | One source read and bounded scans. | Correct fail-fast CLI integration. | `--check` and focused suite pass. | PASS |
| Phase 08 OpenAPI route operations | Each route has the documented success/error statuses and response shape. | 400 validation, auth/role, not-found/conflict, rate, and dependency states are represented; deletes are 204. | N/A — declarative contract; runtime cancellation is covered by caller tests. | Cookie auth, runtime admin role, and CSRF are explicit where applicable. | Query/page/header/body bounds prevent unbounded input. | Operation IDs and path parameters are consistent with runtime routes. | Exact 17-operation matrix and HTTP packages pass. | PASS |
| Phase 08 OpenAPI parameters/security/responses | Header and security components encode cookie, CSRF, idempotency, and positive retry metadata. | Required versus conditional idempotency is explicit; missing keys map to 400 and conflicts to 409. | N/A — declarative definitions. | Cookies and synchronizer header are separated; no client token payload is modeled. | Header/string/UUID bounds are finite. | Reused `$ref` components avoid copy/paste drift. | CSRF/key mutation tests, runtime auth tests, and Redocly pass. | PASS |
| Nine strict success envelope schemas | Every envelope is a closed object with exactly top-level `status`, `requestId`, `data`; status is literal `ok`; data is required. | Object-type, additional-property, required-data, status-const, data-name, and schema-removal mutations all fail. | N/A — wire schema only. | Prevents success/error confusion and extra-field leakage. | Bounded nested arrays/objects in payload schemas. | Consistent schema shape across all nine aliases. | 54/54 mutations pass, including the two named repaired envelopes. | PASS |
| Phase 08 safe DTO schemas and bounds | DTOs expose only normalized editable/projected fields, closed warning vocabularies, bounded pages, and safe errors. | Missing/extra fields, invalid enum values, over-limit arrays, and unsafe projections are rejected by schema/runtime tests. | N/A — wire schema only; no audit snapshot crosses response. | No raw provider, owner, password, token, or audit fields. | Candidate, warning, filter, classification, user, and micronutrient collections are bounded. | DTOs are minimal and align with generated aliases. | Source scans, generated scan, focused HTTP/service tests pass. | PASS |
| `GENERATED` template and nine aliases | `OkEnvelope<TData>` requires literal `ok`, request ID, and data; Phase 08 aliases use it, not legacy `Envelope`. | Generated output detects intentional drift; legacy compatibility envelope remains separate. | N/A — static generated source. | Strict TypeScript shapes prevent consumer success/error weakening. | No runtime allocation; generation is deterministic. | Static template is simple and checked against output. | Generator check, strict alias assertions, and typecheck fixture pass. | PASS |
| Generated TypeScript Phase 08 artifact | Must be byte-identical to generator output and contain no forbidden client-visible fields. | TypeScript compile rejects missing data and non-`ok` classification envelope assignments. | N/A — type declarations only. | Safe DTOs omit raw provider/audit/ownership secrets. | No runtime code added by declarations. | Generated file remains source-controlled and traceable. | `--check`, 459 Bun tests, and `tsc` pass. | PASS |
| `Phase08SuccessEnvelopeTypeChecks` | All nine aliases satisfy strict success shape; negative assignments remain non-assignable. | Missing data and error-status assignments fail at compile time. | N/A — compile-time only. | Reinforces client-side boundary without replacing server authorization. | No runtime work. | Type-level assertions use `Assert` and `AssertFalse` idiomatically. | Frontend typecheck passes. | PASS |
| `test_phase08_routes_security_statuses_and_safe_dtos_match_generated_types` | Current source must satisfy the Phase 08 guard and expose all required generated symbols. | Forbidden DTO field names are rejected from generated output. | N/A — isolated source fixture. | Tests no raw/audit/owner/password/token fields. | Bounded string assertions. | Directly exercises public generator contract. | Focused test passes. | PASS |
| `test_phase08_security_or_warning_drift_is_rejected` | CSRF and warning cardinality are required source invariants. | Removing CSRF or warning bound must produce a mismatch. | N/A — in-memory mutated source. | Protects state-changing and provider-warning boundaries. | Two small source mutations. | Clear targeted test. | Focused test passes. | PASS |
| `test_phase08_classification_names_match_runtime_normalization` | Both request and response classification schemas carry the exact shared rule. | Missing exact rule fails. | N/A — in-memory source. | Prevents alternate validation policy at API boundary. | Constant-time block checks. | Table-driven schema loop. | Focused test passes. | PASS |
| `test_phase08_classification_name_drift_is_rejected` | Classification max length and normalization description cannot silently diverge. | 121 maximum and removed description are both rejected for both schemas. | N/A — in-memory source. | Keeps Unicode normalization and safe character policy synchronized. | Four small mutations. | Uses shared rule constant rather than duplicate expected strings. | Focused test passes. | PASS |
| `test_phase08_generated_success_envelopes_are_strict` | Generated template and every Phase 08 alias must be strict `OkEnvelope`. | A permissive `Envelope` alias or missing strict template fails. | N/A — generated text inspection. | Ensures consumer types do not accept error-shaped success data. | Linear text checks. | Explicit alias loop includes all nine names. | Focused test passes. | PASS |
| `test_phase08_source_success_envelopes_cannot_be_weakened` | Source guard must protect every strict envelope invariant. | Six mutations per envelope cover type, closure, required data, status const, data property, and schema deletion. | N/A — in-memory source; no external state. | Detects source weakening before generation. | 54 bounded mutations. | Parameterized loop is complete and deterministic. | Independent probe confirms 9 envelopes and 54/54 detections. | PASS |
| `test_custom_item_name_and_classification_contracts_match_generated_types` | Custom item name and hierarchy-free classification output match source policy. | Whitespace-only, NUL, and valid padded name cases are challenged. | N/A — generated/source inspection. | Prevents parent/owner projection leakage. | Small regex checks. | Focused assertions are readable. | Focused test passes. | PASS |
| `test_custom_item_name_or_parent_projection_drift_is_rejected` | Name pattern and classification property set are drift-protected. | Removing pattern or adding `parentId` must fail. | N/A — in-memory source. | Blocks unsafe input and hierarchy leakage. | Two bounded mutations. | Pure mutation test. | Focused test passes. | PASS |
| `OperationResponseDriftTest` Phase 08 unit | Groups the eight added Phase 08 test methods under one isolated unittest class. | Individual test failures remain visible; full suite has one unrelated optimization assertion only. | N/A — unittest lifecycle is isolated. | Covers source security/privacy and strict envelope boundaries. | No production I/O. | Standard library `unittest`, table-driven subtests. | Five selected Phase 08 methods pass; full module is 23/24 with unrelated pre-existing failure. | PASS |

Mandatory audit questions were answered for every row: declarative and type-only units have explicit N/A reasons for resources/cancellation; pure source checks have no process, lock, transaction, or network state; runtime consumers were challenged for auth, ownership, malformed inputs, cancellation, error mapping, and audit boundaries through focused HTTP/service/race tests; all loops and payload collections are bounded by the inspected schemas.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| None | — | — | No blocking, important, or optional finding. | Current source, generated output, runtime consumers, focused tests, mutation probe, hashes, and validators agree. | None. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `git status --short; git rev-parse --short HEAD; scoped git diff` | repository root | 0 | PASS; fixed `HEAD` is `81ca40ce`; shared worktree changes were scoped and preserved. | Current worktree, not persisted. |
| `python3 -m unittest scripts/test_generate_api_types.py` | repository root | 1 | 23/24 tests pass; one existing optimization test fails because the unrelated source lacks `Quantity-weighted Jaccard similarity`. | Test output; not a Task 253 failure. |
| Focused five Phase 08 generator tests via `python3 -m unittest scripts.test_generate_api_types.OperationResponseDriftTest...` | repository root | 0 | PASS; all selected route/security/classification/generated-envelope/source-mutation tests pass. | Test output. |
| Independent in-memory nine-envelope mutation probe | repository root | 0 | PASS; 9 envelopes, 54 mutations, 0 misses. | Probe output: `envelopes=9 mutations_checked=54 failures=0`. |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | PASS; generated API types are current. | Generator output. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS; one known intentional OAuth callback 302-only warning. | Redocly output. |
| `cd frontend && ... bun run typecheck` | `frontend/` | 0 | PASS; includes `generated.phase08-typecheck.ts`. | TypeScript output. |
| `cd frontend && ... bun run build` | `frontend/` | 0 | PASS; 208 modules transformed. | Build output; ignored `frontend/dist/`. |
| `cd frontend && ... bun test` | `frontend/` | 0 | PASS; 459 tests, 0 failures, 2099 expectations across 41 files. | Bun output. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS. | Validator output. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS; 263 ordered tasks; Task 253 remains PREPARED. | Validator output. |
| `git diff --check` | repository root | 0 | PASS. | No whitespace errors. |
| Focused Go tests for security, curation, customitem, dataimporter, externaldata, itemcurator, search, tagmanager, useradmin, and httpapi | `backend/` | 0 | PASS. | Go test output. |
| Focused Go coverage with `/tmp/task253-focused.cover`, then `go tool cover -func` | `backend/` | 0 | PASS supporting evidence; aggregate inspected Phase 08 context is 91.9%; `normalizeCurationName` and `normalizeVisibleText` are each 100.0%. No Task 253 runtime implementation was changed. | `/tmp/task253-focused.cover`; documented pre-existing Phase 08 denominator context. |
| Focused Go race tests for the same packages | `backend/` | 0 | PASS; no race report. | Go race output. |
| `go vet ./...` | `backend/` | 0 | PASS. | Vet output. |
| `python3 scripts/check.py` | repository root | 1 | Aggregate stopped at local-stack migration because an existing PostgreSQL type name collision occurs in `000004_micronutrient_vocabulary.up.sql`; preceding traceability, task-list, OpenAPI, vulnerability, vet, and focused checks passed. | Aggregate output; unrelated environment/concurrent migration state. |
| `sha256sum` over every file in Section 9 | repository root | 0 | PASS; hashes captured after review. | Section 9. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-253-review.md` | repository root | 0 | PASS; final evidence is structurally valid. | This review evidence file. |

## 9. Files Inspected and Staleness Fingerprints

The current task-253 preparation hashes for the repaired surface match the files below. The earlier rejected review hashes for the three repaired files intentionally do not match; those files were re-reviewed. The current task-list hash differs from the preparation hash only because prior task statuses were updated concurrently; the Task 253 row was re-read and remains unchanged. The review file is excluded from its own fingerprint.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `api/openapi.yaml` | Phase 08 source-of-truth routes, schemas, and security contracts | None | SHA256 | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `scripts/generate-api-types.py` | Source drift guard and generated template | None | SHA256 | `a2d4b6ab3b41862233c531f2738f861c256cdedcab5ed7963915e89cb6384721` |
| `scripts/test_generate_api_types.py` | Generator and mutation regression tests | None | SHA256 | `f8317f543a0eb730d837d2350d131ba431534c0bd5eb08747dec34631536e3ef` |
| `frontend/src/lib/api/generated.ts` | Generated frontend DTO/envelope artifact | None | SHA256 | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | Strict generated-envelope compile-time fixture | None | SHA256 | `5a8d415198b975e12dcc003ed537f3007843583e3c3dbf80ca81c0b4d015aee3` |
| `backend/internal/security/normalizer.go` | Runtime classification normalization parity consumer | None | SHA256 | `f87732321090d144229227b4573cf5ff1155d80f95c4e68da44a513c55802607` |
| `backend/internal/security/curation_normalizer_test.go` | Runtime 120-boundary adversarial tests | None | SHA256 | `28ea79df82b789d677cd5a4f1649afb51311e757cd936782fe3b7b3e1191b749` |
| `backend/internal/httpapi/admin_controller.go` | Runtime admin/auth/CSRF/audit gateway consumer | None | SHA256 | `cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/httpapi/router.go` | Runtime route middleware ordering and request boundary | None | SHA256 | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/httpapi/curation_validation.go` | Runtime normalized request handoff | None | SHA256 | `14cd4a46838d84fb643fd944e0ba1327ee1af71f22db97ce733d0df5dd4483de1` |
| `backend/internal/httpapi/custom_item_controller.go` | Private ownership/idempotency response consumer | None | SHA256 | `4ea8018aa044b3ab34ee54d8391e9dd4cd3a08dc911a8008888dd9daec791d4d0a` |
| `backend/internal/httpapi/external_search_controller.go` | External normalized candidate response consumer | None | SHA256 | `32086389a2a6ac1d27162ec17cac197f5a103d62cb0d591423ff0344534ac864` |
| `backend/internal/httpapi/filter_option_controller.go` | Public bounded filter-option response consumer | None | SHA256 | `380673327db3acc6043c284109ba66a37b89ef9dd40a0c1dd500ab5092b0e78a` |
| `backend/internal/httpapi/import_controller.go` | Curated import/idempotency/audit response consumer | None | SHA256 | `04e0e65035302d15501dd44e0ba1327ee1af71f22db97ce733d0df5dd4483de1` |
| `backend/internal/httpapi/manual_item_controller.go` | Global item ownership/audit response consumer | None | SHA256 | `b7ec1af1f64a48461922915ff5d2aa012ee870f42e0983fb407d00e6f1496b4c` |
| `backend/internal/httpapi/classification_admin_controller.go` | Classification DTO/audit response consumer | None | SHA256 | `1584656419a549fe7e3975304a7e30feca623e50599043110117a33ff428df08` |
| `backend/internal/httpapi/user_admin_controller.go` | Privacy-minimized user/retry response consumer | None | SHA256 | `ffc6606c3599956aa7877a400a9c1f4f9bdf37976e91e31a921e0c7aefbc4e3f` |
| `docs/design/DESIGN-009.md` | Detailed admin component source | None | SHA256 | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/architecture/ARCH-009.md` | Administration architecture source | None | SHA256 | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/requirements/01_SOFT_REQ_SPEC.md` | SW-REQ-054–057 and CSRF requirement source | None | SHA256 | `80b2f57a8c1caebd8b37cdb949cc7e928f3a128a2b8ed81313637b919cafba8b` |
| `docs/implementation/04_OPEN.md` | Current Phase 08 assumptions and coverage context | None | SHA256 | `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d` |
| `docs/implementation/preparations/task-253.md` | Preparation evidence and repair fingerprints | None | SHA256 | `81c741a726a36d3c568a995562b72d1eed7d3158223799608d215053e0ff8f30` |
| `docs/implementation/02_TASK_LIST.md` | Status/dependency scope control | None; row revalidated | SHA256 | `9925d9bafd1726484ac9dcb170712976296d1c1c6109dbe39105f0fd896cc5a1` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-253-review.md — prior F-253-003 rejection superseded after re-review"
  - "docs/implementation/preparations/task-253.md — prior task-list fingerprint is stale only because earlier task statuses changed; Task 253 row was revalidated"
```

## 10. Coverage and Exceptions

- [x] Required task-local contract and generated-type commands ran.
- [x] Supporting focused runtime coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected; source-check branches and all six envelope mutations per schema were exercised.
- [x] Exceptions exactly match the task row: no Task 253 coverage exception is claimed.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task253-focused.cover"
observed_line_coverage: "91.9% aggregate Phase 08 runtime context; 100.0% normalizeCurationName and normalizeVisibleText"
coverage_passed: true
```

Coverage finding: Task 253 adds OpenAPI, Python generator, generated TypeScript, and compile-time fixture surfaces rather than a new runtime package. The 91.9% aggregate is a supporting denominator across pre-existing Phase 08 runtime packages; the directly relevant classification parity functions are 100.0%, and their adversarial tests pass. Existing Phase 08 lower-coverage deviations are recorded in `docs/implementation/04_OPEN.md`; none is newly attributed to Task 253.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by the reviewed Task 253 contract surface.
- [x] No source-of-truth documentation was contradicted; `ARCH-009`, `DESIGN-009`, and SW-REQ-054–057 align with the contract.
- [x] No generated/cache/build/temporary artifact was unintentionally added; build output is ignored and the only untracked reviewed fixture is the documented typecheck assertion file.
- [x] Public API additions are necessary and used by Phase 08 routes/consumers.
- [x] Duplicate helpers and obsolete aliases were searched for; Phase 08 has one `OkEnvelope` family and does not use legacy `Envelope` aliases.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged through runtime controller/service tests, race tests, and source mutations. Declarative units have no resources or cancellation paths.

Findings: None. The full generator module's single failing optimization assertion and the aggregate local PostgreSQL migration collision are pre-existing/out-of-scope and do not affect Task 253 evidence; the focused Task 253 tests and all required row commands pass.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those conditions hold.

Before accepting the decision, run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-253-review.md
```

```yaml
decision: "PASSED"
reason: "The repaired OpenAPI source, generated TypeScript, runtime parity consumers, route/security matrix, and nine-envelope source mutation gate all pass with current hashes and no blocking or important finding."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None; Task 253 is ready for a PASSED status recommendation. The task-list row remains PREPARED because this review was instructed not to edit it."
```

## 13. Repair Context

N/A — `review_decision` is `PASSED`; no repair context is required.

### Failure Summary

N/A.

### Minimal Repair Goal

N/A.

### Evidence to Reuse

Current preparation evidence, current source mutation output, and this review's fingerprints.

### Required Re-Review Surface

N/A unless any hashed implementation file changes after this review.

### Do Not Change

Do not edit production code or the task-list row as part of this review.
