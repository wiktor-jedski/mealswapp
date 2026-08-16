# Phase 08 UAT — Admin Curation and External Data

## Acceptance status

Phase 08 original delivery Tasks 238-263 and the Phase 08.01 post-review remediation Tasks 264-275 are traced below. Task 274 supplies the current passing aggregate report and refreshed screenshots after the remediation implementation. Task 275 refreshes this acceptance document and its preparation evidence; it does not change task-list status.

Project-owner acceptance has **not** been executed or claimed. Every checklist result and the final decision remain unchecked. Accept Phase 08 only after the project owner executes the checks in the intended acceptance environment and records every result.

## Phase recap

### Original Phase 08 delivery

Tasks 238-263 delivered the restricted administration and external-data boundaries described by `ARCH-009` and `ARCH-012`:

- owner-scoped private custom-food persistence, authenticated CRUD, retry-stable creation, JSON/CSV export, deletion lockout, erasure, and cache purge without exposing global curated food;
- backend-owned substitution filters, typed curation normalization, bounded USDA/OpenFoodFacts clients, provider quota/retry handling, safe diagnostics, canonical units/nutrients, explicit warnings, and evidence-based liquid density;
- an authenticated administrator gateway with server-derived role checks, CSRF/rate/validation ordering, privacy-safe correlation, and atomic mutation-plus-audit behavior;
- read-only external search, editable import confirmation, idempotency/conflict handling, manual global item/classification/user administration, and immediate local-search visibility;
- generated clients, responsive Administration Panel workflows, privacy-safe observability, and SWE.5 integration obligations across PostgreSQL, Redis, providers, generated clients, Svelte, and Chromium.

### Phase 08.01 post-review remediation

Tasks 264-275 add the final reviewed behavior and evidence:

- committed manual-item create/update/delete invalidates shared Redis search generations exactly once after commit, while replay and failed/rolled-back mutations do not invalidate;
- private custom-item create/update recursively rejects duplicate JSON members before typed decode or service dispatch;
- each import attempt owns an immutable draft/idempotency snapshot, incompatible controls are disabled during import, stale completions cannot replace current state, and a new search explicitly keeps or discards an unsaved draft;
- Account Export loading/failure clears stale private objects, success feedback, pending deletion state, and destructive controls until an authoritative owner-safe refresh succeeds;
- administration destructive contrast, 200 ms transitions, reduced-motion opt-out, theme focus styling, and heading weights satisfy the approved visual/accessibility rules;
- Phase 08 Go Doc and TSDoc are validated, including OpenAPI-derived generated-client documentation;
- backend, frontend, browser, accessibility, SWE.5, coverage, report, screenshot, and acceptance-document evidence is rerun and refreshed.

The exact accepted backend/frontend coverage dispositions and dated review-action decisions remain authoritative in `docs/implementation/04_OPEN.md`. They waive no acceptance behavior.

## Traceability

### Architecture, design, and requirements

| Source | Phase 08 responsibility |
|---|---|
| `ARCH-009` / `DESIGN-009` | AdminController, ExternalSearchProxy, DataImporter, ItemCurator, TagManager, UserAdminPanel, authorization, audit coordination, import ownership, and manual/classification/user workflows. |
| `ARCH-012` / `DESIGN-012` | USDAClient, OpenFoodFactsClient, DataNormalizer, RateLimitHandler, bounded provider access, warning/partial-success behavior, and provider-to-curation integration. |
| `DESIGN-001`, `DESIGN-002` | Dynamic substitution filters, selected-item classification merging, stale-response protection, and backend-owned filter semantics. |
| `DESIGN-005` | Private/global item separation, owner predicates, persistence invariants, units, density, classifications, and micronutrients. |
| `DESIGN-008` | DataExporter, owner-safe Account Export, private deletion, lockout, erasure, and cache-purge completion. |
| `DESIGN-010` | RequestValidator ordering and recursive duplicate-JSON rejection before dispatch. |
| `DESIGN-011` | Shared Redis generations, CacheInvalidator behavior, cross-instance visibility, and stale-write rejection. |
| `DESIGN-013`, `DESIGN-014` | Typed normalization, safe error/audit behavior, and privacy-safe low-cardinality metrics/logging. |
| `DESIGN-015`, `DESIGN-017` | Legal erasure transitions and sanitized error/retry behavior. |
| `SW-REQ-019`, `SW-REQ-033`, `SW-REQ-043` | Classification filtering, standardized imported storage, and private-item owner isolation. |
| `SW-REQ-054`–`SW-REQ-057` | Administrative access, external curation, manual global items, and classifications. |
| `SW-REQ-072`, `SW-REQ-073` | Owner-scoped data portability and account erasure. |
| `SW-REQ-084`, `SW-REQ-090` | Safe operational logging and canonical micronutrient nomenclature. |

The authoritative SWE.5 mappings are [`ARCH-009-obligations.md`](../../testing/integration/ARCH-009-obligations.md) and [`ARCH-012-obligations.md`](../../testing/integration/ARCH-012-obligations.md). They trace the original obligations plus remediation obligations `IT-ARCH-009-008` through `IT-ARCH-009-011` and `IT-ARCH-012-004` to architecture, designs, requirements, and executable integration tests.

### Original delivery tasks 238-263

| Task | Delivered surface | Primary traceability | Acceptance evidence |
|---:|---|---|---|
| 238 | Owner-scoped private custom-item persistence and migration | DESIGN-005; SW-REQ-043, SW-REQ-090 | Repository/migration isolation and persistence-invariant tests |
| 239 | Authenticated private CRUD, idempotent create, JSON/CSV export | DESIGN-005, DESIGN-008; SW-REQ-043, SW-REQ-072 | Service/HTTP/export replay and ownership tests |
| 240 | Private-item account-erasure integration | DESIGN-008; SW-REQ-043, SW-REQ-073 | Live erasure integration |
| 241 | Backend-owned substitution filter options | DESIGN-009; SW-REQ-019, SW-REQ-057 | Repository/service/HTTP ordering, degradation, and invalidation tests |
| 242 | Curation input normalization | DESIGN-009, DESIGN-013; SW-REQ-055, SW-REQ-056, SW-REQ-090 | Normalization and pre-dispatch validation tests |
| 243 | USDA client | ARCH-012, DESIGN-012; SW-REQ-055 | Bounded fake-provider tests |
| 244 | OpenFoodFacts client | ARCH-012, DESIGN-012; SW-REQ-033, SW-REQ-055 | Bounded fake-provider tests |
| 245 | Provider quota, retry, and partial success | ARCH-012, DESIGN-012; SW-REQ-055 | Deterministic retry/quota/provider-isolation tests |
| 246 | Provider normalization, warnings, and density | DESIGN-005, DESIGN-012; SW-REQ-033, SW-REQ-090 | Unit/nutrient/density/warning tests |
| 247 | Admin gateway, authorization, middleware, atomic audit | ARCH-009, DESIGN-009, DESIGN-013; SW-REQ-054 | 401/403/admin, ordering, rollback, and safe-envelope tests |
| 248 | Read-only external-search proxy | ARCH-009/012, DESIGN-009/012; SW-REQ-055 | Provider ordering, cancellation, outage, and no-mutation tests |
| 249 | Transactional curated import and conflict/idempotency policy | DESIGN-009; SW-REQ-055, SW-REQ-090 | Import/replay/conflict/rollback/search-visibility tests |
| 250 | Manual global item CRUD | DESIGN-005, DESIGN-009; SW-REQ-056, SW-REQ-090 | CRUD/replay/validation/audit/isolation/search tests |
| 251 | Global classification CRUD and consumer invalidation | DESIGN-009; SW-REQ-019, SW-REQ-057 | CRUD/cycle/in-use/audit/Redis/filter/search tests |
| 252 | Restricted privacy-minimized user administration | DESIGN-009; SW-REQ-054, SW-REQ-073 | Projection, authorization, legal retry, concurrency, and audit tests |
| 253 | OpenAPI contract and generated Phase 08 clients | DESIGN-009; Phase 08 API requirements | Redocly lint, route/status review, and generated-type drift |
| 254 | Fail-closed Administration Panel shell | DESIGN-009; SW-REQ-054 | Store/component/browser role, reset, direct-route, and keyboard tests |
| 255 | External search/import curation UI | DESIGN-009/012; SW-REQ-055, SW-REQ-090 | Component/Playwright provider, warning, conflict, retry, and accessibility tests |
| 256 | Manual item/classification/user administration UI | DESIGN-009; SW-REQ-054, SW-REQ-056, SW-REQ-057, SW-REQ-073 | Component/Playwright CRUD, confirmation, refresh, and failure tests |
| 257 | Dynamic substitution filter UI | DESIGN-001, DESIGN-009; SW-REQ-019, SW-REQ-057 | Unit/component/browser ordering, invalidation, and degradation tests |
| 258 | Backend security/integration/functional gate | ARCH-009/012; Phase 08 requirements | Live PostgreSQL/Redis/HTTP/race evidence |
| 259 | Frontend functional/E2E/accessibility gate | DESIGN-009; Phase 08 requirements | Typecheck/build/unit and desktop/mobile Playwright/axe |
| 260 | Admin/external observability gate | DESIGN-014; SW-REQ-084 | Deterministic metrics/logging and load fixtures |
| 261 | SWE.5 integration verification | ARCH-009/012 | Original architecture obligations and real integration paths |
| 262 | Historical aggregate quality gate | DESIGN-014; all Phase 08 sources | Superseded as final evidence by the Task 274 aggregate refresh |
| 263 | Historical acceptance documentation | DESIGN-009; all Phase 08 sources | Superseded as final evidence by this Task 275 refresh |

### Post-review remediation tasks 264-275

| Task | Remediation/evidence surface | Primary traceability | Current evidence |
|---:|---|---|---|
| 264 | Manual-item post-commit search-cache invalidation | DESIGN-009, DESIGN-011; SW-REQ-056 | Exactly-once/replay/failure tests and live peer Catalog/Substitution visibility |
| 265 | Recursive duplicate JSON rejection for private custom items | DESIGN-010; SW-REQ-043, SW-REQ-090 | Create/update top-level/macro/micronutrient pre-dispatch rejection |
| 266 | Import-attempt ownership and explicit draft keep/discard | DESIGN-009, DESIGN-012; SW-REQ-055 | Component/browser supersession, disabled-control, discard, focus, and retry-key tests |
| 267 | Fail-closed authoritative Account Export refresh | DESIGN-008, DESIGN-009; SW-REQ-043, SW-REQ-072 | Loading/failure stale-state clearing, deletion verification, owner-safe retry |
| 268 | Administration visual/accessibility compliance | DESIGN-009; SW-REQ-054 | Contrast, transition, reduced-motion, focus, heading, keyboard, responsive, and axe tests |
| 269 | Exported backend Go Doc gate | DESIGN-009, DESIGN-012, DESIGN-014 | Identifier-led comment validator, formatting, tests, race, and vet |
| 270 | Hand-written/generated frontend TSDoc gate | DESIGN-009 | TSDoc validator, 24 generator tests, type drift, typecheck, unit tests, and build |
| 271 | Backend remediation regression gate | ARCH-009; DESIGN-005, DESIGN-008, DESIGN-010, DESIGN-011 | Production HTTP/PostgreSQL/Redis/race/security integration |
| 272 | Frontend remediation regression gate | DESIGN-008, DESIGN-009 | 533 unit tests, focused component/browser matrix, production build, and frontend verifier |
| 273 | SWE.5 remediation verification | ARCH-009, ARCH-012 | Obligations 008-011/004 and traced production/browser integration tests |
| 274 | Current aggregate quality, coverage, report, and screenshots | DESIGN-014 | Passing aggregate report, 20 refreshed PNGs, exact machine-checked coverage |
| 275 | Current acceptance evidence refresh | DESIGN-009 | This unchecked UAT, report/screenshot links, evidence-integrity checks, and `task-275.md` |

## Automated verification and evidence

### Current post-review aggregate evidence

Task 274 ran the following current gate after the approved Phase 08.01 implementation. Exact output is recorded in [`task-274.md`](../preparations/task-274.md); Task 275 consumes it and does not represent the aggregate as newly rerun.

| Command | Recorded current result |
|---|---|
| `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` | **PASS**, exit 0. Static, backend, frontend, and browser lanes passed; requirements `91/91`; browser `309/314` with five intentional configured skips and no failures. |
| Aggregate backend `go test -race ./... -p 1 -count=1` | **PASS** across commands and packages, including live PostgreSQL/Redis integration. |
| Aggregate `go vet ./...` and `govulncheck@v1.3.0 ./...` | **PASS**; no called-code vulnerability. |
| Aggregate OpenAPI lint and generated-client drift | **PASS** with only the accepted OAuth callback `302`-only warning. |
| `python3 -m unittest scripts/test_generate_api_types.py scripts/test_check_coverage.py` | **PASS**, 42 tests: all 24 generator tests and 18 coverage/report-contract tests. |
| Frontend typecheck/build/unit/coverage | **PASS**; 219-module build; 533 tests and 2,789 expectations; `95.46%` functions and `96.06%` lines. |
| Go Doc, TSDoc, task-list, and source traceability validators | **PASS**. |
| Tracked-content whitespace scan and `git diff --check` | **PASS**. |
| Final `python3 scripts/check.py --quick` | **PASS** after Task 274 evidence/report changes. |

Focused dependency evidence additionally proves live cross-instance cache refresh, duplicate-key rejection before dispatch, import supersession/draft ownership, Account Export failure/recovery, reduced motion, keyboard focus, and zero serious/critical axe violations. See preparations for Tasks 271-273.

### Report and refreshed screenshots

- Current quality report: [`08_PHASE_REPORT.html`](08_PHASE_REPORT.html) — `QUALITY GATE PASSED`, current coverage tables, direct Go/Bun results, traceability, and refreshed visual evidence.
- Screenshot directory: [`screenshots/`](screenshots/) — 20 valid Phase 08 report PNGs refreshed or byte-identically revalidated by Task 274.
- Representative shell captures: [`08_PHASE_REPORT-desktop.png`](screenshots/08_PHASE_REPORT-desktop.png) and [`08_PHASE_REPORT-mobile.png`](screenshots/08_PHASE_REPORT-mobile.png).
- Representative scenario captures: [`08_PHASE_REPORT-substitution-apple-oat-milk-desktop.png`](screenshots/08_PHASE_REPORT-substitution-apple-oat-milk-desktop.png), [`08_PHASE_REPORT-substitution-apple-oat-milk-mobile.png`](screenshots/08_PHASE_REPORT-substitution-apple-oat-milk-mobile.png), [`08_PHASE_REPORT-task-233-daily-diet-light-desktop.png`](screenshots/08_PHASE_REPORT-task-233-daily-diet-light-desktop.png), and [`08_PHASE_REPORT-task-233-daily-diet-light-mobile.png`](screenshots/08_PHASE_REPORT-task-233-daily-diet-light-mobile.png).
- The report screenshots are automated visual-regression evidence. They are not represented as project-owner execution of the administration checks below.

## Current coverage disposition

The authoritative exact exceptions are under Phase 08 in `docs/implementation/04_OPEN.md` and are machine-checked by `scripts/check.py`.

- Backend direct repository aggregate: `87.5%`.
- Machine-defined Phase 08 backend runtime scope: `4,537/4,849` statements (`93.6%`). Every below-100 file and statement coordinate is recorded under reason `B1`–`B4`; Task 274 added no runtime statement or broader exception.
- Frontend aggregate: `95.46%` functions and `96.06%` lines. Current Phase 08 below-100 rows are `admin-workflows.ts`, `account-data-client.ts`, `admin-client.ts`, and generated `generated.ts` line 185 under reasons `F1`–`F3`.
- Svelte components do not emit Bun coverage rows; their disposition is the passing focused component suite plus desktop/mobile Playwright/axe coverage.
- No authorization, ownership, private/global isolation, CSRF, idempotency/replay, duplicate-key rejection, audit rollback, post-commit invalidation, cross-instance visibility, import ownership, Account Export fail-closed state, generated-contract drift, accessibility, or browser behavior is waived.

## Project-owner acceptance checks

### Preconditions

1. Start PostgreSQL and Redis with `bash scripts/start-services.sh`, apply migrations, then start the API and frontend with repository-local caches.
2. Use deterministic approved USDA/OpenFoodFacts fixtures or non-production credentials. Never record secrets, cookies, tokens, raw provider payloads, email plaintext, or idempotency keys.
3. Prepare one verified administrator, standard users A and B, global curated items, and one private custom item per standard user.
4. Use two API/frontend instances sharing PostgreSQL/Redis where a check requires cross-instance cache visibility.
5. Capture only sanitized request IDs and diagnostics for failure checks.

### Acceptance checklist

| ID | Check and steps | Accept criteria | Result |
|---|---|---|---|
| UAT-08-01 | **Non-admin denial.** As anonymous and user A, navigate directly to `/admin` and call representative admin routes; then sign in as admin. | Anonymous calls return 401, non-admin calls return 403, no restricted data/control is exposed, spoofed identity does not help, and verified-admin access succeeds. | ☐ |
| UAT-08-02 | **External search/import.** Search USDA, OpenFoodFacts, then both; select/edit/classify/confirm a candidate and search locally. | Search is read-only before confirmation; normalized candidates/warnings render; exactly one global ownerless item is imported and immediately searchable. | ☐ |
| UAT-08-03 | **Warnings and liquid density.** Curate incomplete/suspicious liquid data and correct physical state/density. | Warnings are safe and clear; no silent `1 ml = 1 g`; trusted volume evidence is preferred and density provenance is explicit. | ☐ |
| UAT-08-04 | **Manual global CRUD.** Create/update/delete solid and liquid items and submit invalid nutrition/image/classification/density data. | Valid writes return authoritative ownerless state; invalid writes fail safely; create/update is searchable and delete removes it without exposing global items through private routes. | ☐ |
| UAT-08-05 | **Classifications and filters.** Create/rename/attach classifications, exercise duplicate/cycle/in-use safeguards, then remove/delete and inspect fresh/already-open clients. | Invalid operations do not mutate; committed labels propagate; selected classifications merge once by ID; deletion removes options without invented frontend policy. | ☐ |
| UAT-08-06 | **Private isolation/export/erasure.** Cross-read/update/delete users A/B items; export A; request deletion, test lockout, complete/retry erasure, and recheck B/global data. | Cross-owner access discloses nothing; exports contain only A's private data; erasure removes A's private/PII/session/cache data while B/global data survives. | ☐ |
| UAT-08-07 | **Restricted user administration.** Perform bounded lookup and one legal deletion retry; try illegal and concurrent retry. | Only the privacy-minimized projection appears; one legal transition is claimed/audited once; no role/password/token/impersonation/arbitrary-edit surface exists. | ☐ |
| UAT-08-08 | **Audit rollback.** Force audit persistence failure for import, manual item, and classification mutation fixtures. | Mutation/idempotency/audit state rolls back, UI claims no success, caches remain unchanged, and sanitized diagnostics contain no PII/secrets/raw payloads. | ☐ |
| UAT-08-09 | **Provider degradation.** Exercise one-provider and complete outage, quota reset, cancellation, and stale response after a newer query. | Partial success is bounded; full outage is safe; retries are isolated/bounded; stale/cancelled results cannot replace current state; reset recovers. | ☐ |
| UAT-08-10 | **Idempotent retry.** Simulate lost responses and concurrent retry for private create, curated import, and manual create; reuse a key with changed input. | Exact retry has one identity/effect; changed-body reuse conflicts; concurrent retries do not duplicate; keys are not logged or browser-persisted. | ☐ |
| UAT-08-11 | **Manual-item cache visibility.** Prewarm Catalog and Substitution Search on instance B; create, rename, and delete through instance A; repeat an exact create and force validation/audit rollback. | Committed mutations become visible across both search modes without stale repopulation; each advances generation once; replay and every failed/rolled-back mutation advance it zero times. | ☐ |
| UAT-08-12 | **Duplicate JSON rejection.** Send duplicate top-level, `macrosPer100`, and micronutrient keys to authenticated private create and update; then send valid bodies. | Each duplicate body returns safe `400 invalid_json` before dispatch/persistence with unchanged state; valid, unknown-field, required-field, CSRF, ownership, and replay behavior remains correct. | ☐ |
| UAT-08-13 | **Import ownership and draft discard.** Delay an import, start a newer workflow, exercise success/conflict/ambiguity/failure completions, try controls while importing, and start a search with an unsaved draft using Keep editing and Discard draft and search. | Stale completion changes nothing; incompatible controls are disabled; Keep preserves draft/key; Discard clears state/ownership before search and ignores late completion; focus returns visibly; completed import resets without a needless warning. | ☐ |
| UAT-08-14 | **Failed Account Export refresh.** Load private data, fail refresh, delete then fail authoritative refresh, and retry with owner-safe data. | Loading/failure hides stale objects and destructive controls; stale success/pending state is cleared; accepted deletion reports verification required; success is claimed only after refresh; retry restores current owner-free data. | ☐ |
| UAT-08-15 | **Contrast, transitions, and reduced motion.** Inspect destructive confirmation in light/dark; activate all administration buttons normally and with reduced-motion preference. | Destructive text contrast is at least `4.5:1` (`4.83:1` light and `6.41:1` dark in automated evidence); buttons use 200 ms transitions normally and no transition under reduced motion. | ☐ |
| UAT-08-16 | **Form focus and headings.** Keyboard through administration inputs/selects/textareas and inspect headings, labels, legends, statuses, and controls in both themes and mobile/desktop. | Controls use theme Surface, 1 px Border, and a visible 2 px Primary focus ring without clipping/hidden focus; only headings use Bold 700; semantic non-heading weights remain appropriate. | ☐ |
| UAT-08-17 | **Accessibility and regression.** Complete admin workflows keyboard-only on desktop/mobile in light/dark, run axe, then verify auth, Catalog/Substitution, subscription, Daily Diet, and optimization baseline views. | Focus/modal behavior is correct; axe has zero serious/critical violations; no stale unsafe state or clipping; established auth/search/non-admin workflows remain intact. | ☐ |

## Known notes

- Redocly retains one accepted warning because the OAuth callback intentionally has a `302` redirect and no `2XX`; it is unrelated to the Phase 08 admin contract.
- Five Playwright cases are intentionally skipped by project/opt-in configuration in the aggregate run; the run has no failed browser case, and focused live/backend/browser dependency evidence covers the remediation paths.
- Coverage remains below the aspirational 100% goal only through the exact machine-checked backend/frontend exceptions above.
- Phase 09 owns production infrastructure and cross-cutting hardening actions listed in `04_OPEN.md`; none is claimed by Phase 08.

## Acceptance decision

Accept Phase 08 only when:

1. UAT-08-01 through UAT-08-17 are checked as passing by the project owner;
2. any environment-gated check is rerun in the intended environment or explicitly accepted with owner/date/reason;
3. no open defect compromises authorization, privacy, audit atomicity, idempotency, cache visibility, hostile-input rejection, import ownership, export safety, accessibility, or core regression behavior; and
4. the current Task 274 aggregate evidence, exact coverage dispositions, report/screenshots, and Task 275 evidence-integrity validators remain valid.

Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations

Project owner: ____________________  Date: ____________________

Notes / defect links: ________________________________________________________________
