# Phase 08 UAT — Admin Curation and External Data

## Acceptance status

Phase 08 implementation evidence and aggregate automated verification are available for project-owner acceptance. Tasks 238-262, including Task 258, are recorded as `PASSED`; Task 263 is currently `PREPARED`. This task is limited to preparing acceptance documentation and must not edit or reconcile statuses in `docs/implementation/02_TASK_LIST.md`.

Project-owner checks in this document have **not** been claimed as executed. Record each result in the checklist below. Accept Phase 08 only when every required check passes or the owner explicitly records a deviation.

## Phase recap

Phase 08 implements the restricted administration and external-data boundaries described by ARCH-009 and ARCH-012:

- mandatory owner-scoped private custom-food persistence, authenticated CRUD, retry-stable creation, JSON/CSV export, deletion lockout, erasure, and cache purge without exposing or deleting global curated food;
- backend-owned substitution filter options from active classifications, allergens, dietary presets, and physical-state policy, with cache invalidation and safe frontend degradation;
- typed curation normalization for names, provider identifiers, URLs, units, nutrition data, and provider text;
- bounded USDA and OpenFoodFacts clients, per-provider quota/retry handling, partial success, safe diagnostics, canonical nutrient/unit normalization, explicit warnings, and evidence-based liquid density without a silent `1 ml = 1 g` assumption;
- an authenticated admin gateway with server-derived role checks, CSRF/rate/validation ordering, privacy-safe request correlation, and transactionally atomic mutation-plus-audit behavior;
- read-only external search followed by explicit editable import confirmation, natural-key/idempotency replay protection, conflict confirmation, and immediate local-search visibility;
- manual global item CRUD, global Food Category/Culinary Role management, in-use safeguards, restricted privacy-minimized user lookup, and legal account-deletion retry;
- generated OpenAPI clients and responsive Administration Panel workflows for external import, manual item/classification/user administration, private export/deletion, and dynamic substitution filters;
- privacy-safe metrics/logging and SWE.5 integration obligations across authentication, providers, PostgreSQL, Redis, generated clients, Svelte, browser workflows, export/deletion, search, and audit persistence.

No Phase 08 assumption or action remains open in `docs/implementation/04_OPEN.md`. The accepted backend/frontend coverage exceptions are precise, machine-checked, and do not waive any acceptance behavior.

## Traceability

### Architecture and design sources

| Source | Phase 08 responsibility |
|---|---|
| `ARCH-009` / `DESIGN-009` | AdminController, ExternalSearchProxy, DataImporter, ItemCurator, TagManager, UserAdminPanel, non-admin denial, audit coordination, import/manual/classification/user workflows. |
| `ARCH-012` / `DESIGN-012` | USDAClient, OpenFoodFactsClient, DataNormalizer, RateLimitHandler, bounded provider access, warnings, partial success, outage degradation. |
| `DESIGN-005` | Private/global food-item separation, mandatory owner predicates, macro/micronutrient/unit/density/classification persistence invariants. |
| `DESIGN-008` | Authenticated private-data routes, export bundle, deletion lockout, erasure, and cache-purge completion. |
| `DESIGN-013` | Typed input normalization, CSRF/rate controls, parameterized persistence, metadata-only rejection logging, fail-closed sensitive audit behavior. |
| `DESIGN-001` | SearchView dynamic-filter consumption, selected-item classification merging, stale-request protection, recoverable frontend state. |
| `DESIGN-014` | Privacy-safe, low-cardinality provider/admin/custom-item metrics and structured logs. |

Supporting sources exercised by the task evidence include `DESIGN-002` filter semantics, `DESIGN-010` route-validation ordering, `DESIGN-015` erasure transitions, and `DESIGN-017` sanitized error/retry behavior.

### Requirement sources

| Requirement | Phase 08 acceptance surface |
|---|---|
| `SW-REQ-019` | Persisted classification options propagate into substitution filtering. |
| `SW-REQ-033` | Imported provider data is normalized before local persistence. |
| `SW-REQ-043` | Private custom items remain visible only to their owner. |
| `SW-REQ-054` | Administration Panel and APIs are restricted to verified administrators. |
| `SW-REQ-055` | Administrators search USDA/OpenFoodFacts, edit candidates, classify them, and explicitly import them. |
| `SW-REQ-056` | Administrators create, update, and delete global curated items. |
| `SW-REQ-057` | Administrators manage global Food Categories and Culinary Roles. |
| `SW-REQ-072` | JSON and CSV account exports include owner-scoped private custom items. |
| `SW-REQ-073` | Account deletion removes PII/private custom items and supports only legal retry transitions. |
| `SW-REQ-084` | Admin/provider/custom-item behavior emits privacy-safe operational logging and metrics. |
| `SW-REQ-090` | Imported/manual micronutrient keys are validated against the canonical active vocabulary. |

The authoritative SWE.5 mappings are `docs/testing/integration/ARCH-009-obligations.md` and `docs/testing/integration/ARCH-012-obligations.md`. Together they trace `IT-ARCH-009-001` through `IT-ARCH-009-007` and `IT-ARCH-012-001` through `IT-ARCH-012-003` to the architecture, designs, requirements, and executable tests.

### Task 238-263 matrix

| Task | Delivered surface | Primary traceability | Acceptance evidence |
|---:|---|---|---|
| 238 | Owner-scoped private custom-item persistence and migration | DESIGN-005; SW-REQ-043, SW-REQ-090 | Repository/migration isolation, owner predicate, unit/density/micronutrient tests |
| 239 | Authenticated private CRUD, create idempotency, JSON/CSV export | DESIGN-005, DESIGN-008; SW-REQ-043, SW-REQ-072, SW-REQ-090 | Service/HTTP/export replay and ownership tests |
| 240 | Private-item account-erasure integration | DESIGN-008; SW-REQ-043, SW-REQ-073 | `TestTask240CustomItemErasureIntegration` |
| 241 | Backend-owned substitution filter options | DESIGN-009; SW-REQ-019, SW-REQ-057 | Repository/service/HTTP ordering, empty/degraded, and invalidation tests |
| 242 | Curation input normalization | DESIGN-009, DESIGN-013; SW-REQ-055, SW-REQ-056, SW-REQ-090 | Table-driven normalization and pre-dispatch HTTP validation tests |
| 243 | USDA client | ARCH-012, DESIGN-012; SW-REQ-055 | Fake-server query/key/page/deadline/body/projection tests |
| 244 | OpenFoodFacts client | ARCH-012, DESIGN-012; SW-REQ-033, SW-REQ-055 | Fake-server caller-ID/page/deadline/body/projection tests |
| 245 | Provider quota, retry, and partial-success orchestration | ARCH-012, DESIGN-012; SW-REQ-055 | Deterministic clock/jitter, retry exhaustion, provider-isolation tests |
| 246 | Provider food normalization, warnings, and density | DESIGN-005, DESIGN-012; SW-REQ-033, SW-REQ-055, SW-REQ-090 | Unit/nutrient/density/warning/vocabulary query-count tests |
| 247 | Admin gateway, authorization, CSRF/rate boundary, atomic audit | ARCH-009, DESIGN-009, DESIGN-013; SW-REQ-054 | 401/403/admin, middleware-ordering, rollback, sanitized-envelope tests |
| 248 | Read-only external-search proxy | ARCH-009/012, DESIGN-009/012; SW-REQ-055 | Provider selection, ordering, cancellation, outage, no-mutation tests |
| 249 | Transactional curated import and conflict/idempotency policy | DESIGN-009; SW-REQ-055, SW-REQ-090 | Import/replay/conflict/rollback/search-visibility tests |
| 250 | Manual global item CRUD | DESIGN-005, DESIGN-009; SW-REQ-056, SW-REQ-090 | CRUD/replay/validation/audit/private-isolation/search tests |
| 251 | Global classification CRUD and consumer invalidation | DESIGN-009; SW-REQ-019, SW-REQ-057 | CRUD/cycle/duplicate/in-use/audit/Redis/filter/search tests |
| 252 | Restricted privacy-minimized user administration | DESIGN-009; SW-REQ-054, SW-REQ-073 | Projection, authorization, legal retry, concurrency, audit tests |
| 253 | OpenAPI contract and generated Phase 08 clients | DESIGN-009; SW-REQ-043, SW-REQ-054-057, SW-REQ-072-073, SW-REQ-090 | Redocly lint, route/status review, generated-type drift |
| 254 | Fail-closed Administration Panel shell | DESIGN-009; SW-REQ-054 | Store/component/browser role, direct-route, identity-reset, keyboard tests |
| 255 | External search/import curation UI | DESIGN-009/012; SW-REQ-055, SW-REQ-090 | Component/Playwright provider, warning, conflict, retry, accessibility tests |
| 256 | Manual item/classification/user administration UI | DESIGN-009; SW-REQ-054, SW-REQ-056-057, SW-REQ-073 | Component/Playwright CRUD, confirmation, refresh, audit-failure tests |
| 257 | Dynamic substitution filter UI | DESIGN-001, DESIGN-009; SW-REQ-019, SW-REQ-057 | Source assertion, unit/component/browser ordering/invalidation/degraded tests |
| 258 | Backend security/integration/functional gate | ARCH-009/012; Phase 08 requirement set | Live PostgreSQL/Redis/HTTP/race integration evidence |
| 259 | Frontend functional/E2E/accessibility gate | DESIGN-009; Phase 08 requirement set | Typecheck/build, 526 unit tests, Playwright/axe desktop/mobile suites |
| 260 | Admin/external-data observability gate | DESIGN-014; SW-REQ-043, SW-REQ-054-057, SW-REQ-072-073, SW-REQ-084, SW-REQ-090 | Deterministic metric/log and representative load fixtures |
| 261 | SWE.5 integration verification | ARCH-009/012, DESIGN-009/012; SW-REQ-043, SW-REQ-054-057, SW-REQ-072-073, SW-REQ-090 | Ten passing architecture obligations and real provider/browser integrations |
| 262 | Aggregate quality and exact coverage-exception gate | DESIGN-014; all Phase 08 sources | Passing aggregate report, exact coverage contracts, report/screenshots |
| 263 | Acceptance documentation | DESIGN-009; all Phase 08 sources | This UAT, validator reruns, artifact hashes in `preparations/task-263.md` |

## Automated verification and evidence

### Task 262 aggregate evidence

The following commands and results are recorded by the final Task 262 preparation and embodied in `08_PHASE_REPORT.html`. They are not represented as newly rerun by Task 263.

| Command | Recorded result |
|---|---|
| `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` | PASS, exit 0; contract/traceability/task-list, OpenAPI, coverage-contract regressions, vet, vulnerability scan, local stack/migrations/API, backend test/race/coverage, frontend verification/type drift/typecheck/build/test/coverage, focused/full browser/axe, and report generation passed. |
| `cd backend && go test ./... -p 1 -count=1` | PASS. |
| `cd backend && go test -race ./... -p 1 -count=1` | PASS. |
| `cd backend && MEALSWAPP_REDIS_URL=redis://localhost:6379/12 go test -p 1 -count=1 -coverpkg=./internal/... -coverprofile=phase08-coverage.out ./internal/...` | PASS; changed Phase 08 scope `4,523/4,841` (`93.4%`) after deduplicating cross-package source blocks. |
| `cd backend && go vet ./...` | PASS. |
| `cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; zero called or imported vulnerabilities. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID with one accepted pre-existing warning for the intentional OAuth callback `302`-only response. |
| `cd frontend && bun run check:api-types` | PASS; generated API types current. |
| `cd frontend && bun run typecheck` | PASS. |
| `cd frontend && bun run build` | PASS. |
| `cd frontend && bun test` | PASS; 526 tests, 2,456 expectations, zero failures. |
| `cd frontend && bun test --coverage` | PASS with accepted exact exceptions; aggregate `95.46%` functions and `96.06%` lines. |
| `cd frontend && bun run test:e2e` | PASS; all 294 scheduled desktop/mobile Chromium cases completed without failure, with five intentionally environment-gated real-stack cases skipped in the report run. |
| `bash scripts/verify-task-261-ui.sh` | PASS on immediate rerun and ten-run repetition after one disclosed non-reproducing CSRF-shaped 403; 11/11 successful post-investigation runs. |
| `git diff --check` | PASS. |

Task 262 also records one unrelated pre-existing failure in `python3 scripts/test_generate_api_types.py`: 23/24 tests pass and one Phase 07 wording assertion expects an older optimization phrase. Current OpenAPI lint, generated-output drift, frontend typecheck/build/tests, and all Phase 08 contracts pass. This is not a Phase 08 acceptance waiver.

### Task 263 commands

Task 263 runs only documentation-relevant, non-mutating validation after creating this UAT and preparation record:

| Command | Expected/recorded result |
|---|---|
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS; requirements/design/source traceability and JSON sidecars valid. |
| `git diff --check -- docs/implementation/implemented/08_PHASE_UAT.md docs/implementation/preparations/task-263.md` | PASS; no whitespace errors. |

Exact Task 263 command output and SHA-256 evidence are recorded in `docs/implementation/preparations/task-263.md`.

### Report and screenshots

- Quality report: [`08_PHASE_REPORT.html`](08_PHASE_REPORT.html) — `QUALITY GATE PASSED`; requirements `91/91`, traceability, local stack, frontend verifier, backend/frontend coverage, and exact Phase 08 exceptions are embedded.
- Screenshot directory: [`screenshots/`](screenshots/) — 20 desktop/mobile PNGs generated by the aggregate frontend verifier for authentication, catalog, substitution, subscription, Daily Diet, optimization, and responsive regression states.
- Representative current-shell screenshots: [`08_PHASE_REPORT-desktop.png`](screenshots/08_PHASE_REPORT-desktop.png) and [`08_PHASE_REPORT-mobile.png`](screenshots/08_PHASE_REPORT-mobile.png).
- Phase 08 admin-specific behavior is evidenced by the component/Playwright/axe suites and the real-stack Task 261 flow. The generic aggregate screenshot set is visual regression evidence; it is not mislabeled as proof that the project owner completed the admin checks below.

## Coverage exceptions

The exact authoritative exceptions are under Phase 08 in `docs/implementation/04_OPEN.md` and are machine-checked by `scripts/check.py`.

- Backend changed Phase 08 runtime scope: `4,523/4,841` statements (`93.4%`). The exact 31 below-100 files and statement-block coordinates are categorized as defensive dependency/encoder/claim-corruption branches (`B1`), repeated safe repository/HTTP mappings (`B2`), configuration/cache/wiring fallbacks (`B3`), or instrumentation-only paths (`B4`).
- Backend direct package totals include `deletionworker 100.0%`, `externaldata 99.8%`, `tagmanager 100.0%`, `userdata 97.2%`, `search 96.5%`, and the lower package totals precisely listed in `04_OPEN.md` and the report.
- Frontend aggregate: `95.46%` functions and `96.06%` lines. Phase 08 exceptions are `admin-workflows.ts` (`90.91%` functions, `98.51%` lines), `account-data-client.ts` (`100.00%`, `98.00%`), `admin-client.ts` (`97.22%`, `100.00%`), and generated `generated.ts` fallback line 185 (`100.00%`, `98.98%`).
- Svelte components do not emit Bun runtime rows; component tests and the Playwright/axe suites cover their behavior.
- No authorization, ownership, private/global isolation, CSRF, idempotency/replay, validation, parameterized persistence, mutation-plus-audit rollback, provider bounding/degradation, erasure, invalidation, sanitized observability, accessibility, generated-contract decoding, or immediate search-visibility behavior is waived.

## Project-owner acceptance checks

### Preconditions

1. Start PostgreSQL and Redis with `bash scripts/start-services.sh`.
2. Apply migrations with `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up`.
3. Start the API and frontend using the repository commands. Configure test USDA/OpenFoodFacts credentials or deterministic approved fixtures; do not use production secrets in screenshots or notes.
4. Prepare three identities: one verified administrator, standard user A, and standard user B. Prepare one global curated item and one private custom item for each standard user.
5. Capture request IDs and sanitized server logs for mutation/rollback checks. Never paste tokens, cookies, raw provider payloads, email plaintext, or idempotency keys into this document.

### Acceptance checklist

| ID | Check and steps | Accept criteria | Result |
|---|---|---|---|
| UAT-08-01 | **Non-admin denial.** As anonymous and user A, navigate directly to `/admin` and call representative admin read/mutation routes. Then sign in as admin. | Anonymous API calls return 401; authenticated non-admin calls return 403; no admin navigation/control/data is exposed; spoofed role/identity fields do not help; admin access succeeds. | ☐ |
| UAT-08-02 | **External search/import.** As admin, search USDA, OpenFoodFacts, then both; change provider/page, select a candidate, edit name/macros/classifications, and confirm import. Search locally for the result. | Search is read-only before confirmation; bounded normalized candidates render; edits survive confirmation; one global ownerless item is imported and immediately searchable; no raw provider payload is shown. | ☐ |
| UAT-08-03 | **Warnings and liquid density.** Choose an incomplete/suspicious liquid candidate. Inspect missing-data and suspicious-total warnings; correct physical state/density and import. | Warnings are clear and non-secret; suspicious liquid totals warn rather than automatically reject; import cannot silently assume `1 ml = 1 g`; density provenance is imported/manual/estimated and trusted USDA volume evidence is preferred when present. | ☐ |
| UAT-08-04 | **Manual global CRUD.** Create a solid and liquid global item, view/update them, then soft-delete one. Try invalid macro, image, micronutrient, classification, and missing-density values. | Valid CRUD returns authoritative state; invalid writes fail safely; created/updated item is searchable; deleted item disappears; global items have no owner and private-item routes cannot expose them. | ☐ |
| UAT-08-05 | **Classifications and filter propagation.** Create and rename a Food Category/Culinary Role, attach it to an item, try duplicate/cycle/in-use deletion, then remove use and delete. Open Substitution filters in a fresh and already-open client. | Duplicate/cycle/in-use actions fail without mutation; committed labels propagate after invalidation across clients; selected-item classifications merge once by ID; deletion removes the option; no hardcoded fallback invents policy. | ☐ |
| UAT-08-06 | **Private custom-item isolation/export/erasure.** As users A and B create similarly named private items. Cross-read/update/delete IDs; export A as JSON and CSV; request A deletion, test write lockout, complete/retry erasure, and recheck B/global data. | Each user sees only own items; cross-user IDs disclose nothing; exports include exactly A's private item and no owner/global leakage; pending deletion blocks writes; completion removes A's private item/PII/session/cache while B and global records survive; receipt is pseudonymous. | ☐ |
| UAT-08-07 | **Restricted user administration.** As admin perform bounded lookup and retry one eligible failed account deletion; try an ineligible state and concurrent retry. Inspect response/UI fields. | Only the approved privacy-minimized projection appears; one legal transition is claimed once and audited; illegal/concurrent repeats fail safely; no role mutation, password/token access, impersonation, arbitrary editing, or deletion internals are exposed. | ☐ |
| UAT-08-08 | **Audit rollback.** Force audit persistence failure for import, manual item, and classification mutation fixtures; reload authoritative state and inspect request-correlated diagnostics. | Each mutation rolls back with no item/classification/import/idempotency success residue; UI shows no optimistic success; error/log correlation uses request ID and excludes PII, secrets, raw payloads, and before/after snapshots. | ☐ |
| UAT-08-09 | **Provider degradation.** Exercise one-provider timeout/rate limit/unavailability, complete outage, quota reset, cancellation, and a stale response after a newer query. | Partial success keeps valid candidates plus bounded warnings; complete outage returns empty safe state; retries are bounded and isolated per provider; cancellation/stale responses do not replace newer state; safe retry recovers after reset. | ☐ |
| UAT-08-10 | **Idempotent retry.** Simulate a lost response after private-item create, curated import, and manual global create; retry the same intent/key, then reuse the key with a changed normalized body and issue concurrent retries. | Exact retry returns one stable identity with one mutation/audit effect; changed-body reuse returns conflict; concurrent retries do not duplicate; deliberate new intent uses a new key; keys are not logged or stored in browser persistence. | ☐ |
| UAT-08-11 | **Accessibility and responsive themes.** Complete admin shell, external import, CRUD, classification, user lookup/retry, private export/delete, confirmations, and dynamic filters using keyboard only on desktop/mobile in light/dark themes. Run axe. | Focus order and modal containment are correct; visible labels/errors/warnings are understandable; no clipping, stale unsafe state, or inaccessible destructive action; axe reports zero serious/critical violations in tested views. | ☐ |
| UAT-08-12 | **Search/auth regression.** Verify anonymous Catalog Search, login/register/logout/session expiry, Catalog and Substitution search, dynamic filters, authenticated subscription route, Daily Diet, and optimization baseline views after admin activity and provider outage. | Anonymous catalog remains usable; auth/session state is fail-closed and resets on logout/account change; core search stays responsive and returns imported/updated but not deleted items; established non-admin workflows and saved state remain intact. | ☐ |

## Known notes

- The Redocly `operation-2xx-response` warning for the OAuth callback is accepted because that endpoint intentionally redirects with `302`; it is unrelated to Phase 08 admin contracts.
- Task 262 observed one transient CSRF-shaped 403 in the real-stack Task 261 script. It did not reproduce in the immediate rerun or ten-run repetition (11/11 passes after investigation), and no implementation exception or unsafe retry was added. If UAT reproduces it, capture the request ID and reject acceptance pending diagnosis.
- Five real-stack browser cases are intentionally environment-gated in the aggregate browser run; the dedicated real-stack Task 261 script supplies the cross-component evidence for private deletion and classification/filter publication.
- Phase 09 still owns production infrastructure and cross-cutting hardening actions listed under its own section in `04_OPEN.md`; none is presented as completed by this phase.

## Acceptance decision

Accept Phase 08 when:

1. UAT-08-01 through UAT-08-12 are checked as passing by the project owner;
2. any environment-gated check is rerun in the intended acceptance environment or explicitly accepted with owner/date/reason;
3. no open defect compromises authorization, privacy, audit atomicity, idempotency, provider safety, accessibility, or search/auth regression behavior; and
4. accepted coverage exceptions remain exactly as recorded and validators still pass.

Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations

Project owner: ____________________  Date: ____________________

Notes / defect links: ________________________________________________________________
