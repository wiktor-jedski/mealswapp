# Task 262 preparation — Phase 08 coverage and aggregate quality gate

## Outcome and scope

- Task: **262 — Phase 08 Coverage and Aggregate Quality Gate**.
- Task-list row is `PREPARED`; the finding repair did not edit `docs/implementation/02_TASK_LIST.md`.
- The original aggregate gate passed and produced `docs/implementation/implemented/08_PHASE_REPORT.html` plus 20 report screenshots. The Task 262 review then rejected the preparation because the executable gate did not bind those accepted exceptions to current Phase 08 measurements. The repair below closes that finding; final rerun evidence supersedes the original path-only acceptance evidence.
- Scope was limited to aggregate verification, Phase 08 action/assumption/coverage disposition, report evidence, and this preparation document. Extensive pre-existing/concurrent Phase 08 source, test, review, preparation, task-list, and design changes were preserved.
- `tools.md`, named by the phase-completion workflow, does not exist in this repository. Repository commands were taken from `AGENTS.md`, task 262, package manifests, and `scripts/check.py`; this was not a verification blocker.
- Task 263 owns `08_PHASE_UAT.md`; no acceptance document or project-owner acceptance claim was created here.

## Sources and contract

Read before verification:

- `docs/implementation/01_PLAN.md`, task row 262 in `02_TASK_LIST.md`, and the complete Phase 08 section of `04_OPEN.md`;
- `docs/architecture/ARCH-009.md`, `ARCH-012.md`, and the Phase 08 task design sources `DESIGN-001`, `DESIGN-005`, `DESIGN-008`, `DESIGN-009`, `DESIGN-012`, `DESIGN-013`, and `DESIGN-014`;
- Tasks 258-261 preparation/review/obligation evidence and `docs/testing/integration/ARCH-009-obligations.md` / `ARCH-012-obligations.md`;
- `scripts/check.py`, frontend package scripts, the `phase-completion`, `golang-security`, and failure-diagnosis skill instructions.

Task 262 requires contract and generated-type drift, traceability, task-list integrity, backend formatting/tests/coverage/vet/race/vulnerability checks, frontend typecheck/build/tests/coverage, local PostgreSQL/Redis integration, browser/axe verification, `git diff --check`, precise coverage disposition, and disposition of every Phase 08 assumption/action.

## Finding repair and executable exception contract

- `scripts/check.py` now creates `backend/phase08-coverage.out` with `-coverpkg=./internal/...`, deduplicates identical source blocks across package test binaries, and measures the fixed Phase 08 runtime manifest directly. Every below-100 file must exactly match the covered/total count and `line.column-line.column` uncovered statement blocks under the Phase 08 section of `04_OPEN.md`; missing, malformed, stale, additional, or unjustified entries fail.
- The measured backend contract is `4,523/4,841` statements (`93.4%`). The 31 accepted files, exact uncovered blocks, and evidence categories `B1`-`B4` are recorded in the machine-checked `phase08-backend-coverage-contract` in `04_OPEN.md`; all other files in the 47-file runtime manifest are measured at 100%. No behavior named in the original non-waiver statement is waived.
- The generic frontend exception gate no longer accepts path presence. It requires the complete set of current below-100 Bun rows to match exact function percentage, line percentage, uncovered-line output, owning phase, and a defined evidence category. The Phase 08 validator separately requires all nine Phase 08 runtime rows and binds its four current exceptions to Phase 08.
- Deterministic `scripts/test_check_coverage.py` tests cover valid contracts and reject missing, malformed, over-broad, unjustified, stale-metric, and wrong-phase exceptions. This suite is now part of `scripts/check.py`.
- `scripts/generate_report.py` now includes the exact Phase 08 backend/frontend accepted-exception tables and states that aggregate success depends on semantic validation against current measurements.

## Final command evidence

Commands ran from the repository root unless a directory prefix is shown. Backend and frontend cache variables were supplied by `scripts/check.py` or set to the repository-local values documented in `AGENTS.md`.

| Command | Result |
|---|---|
| `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` (first run) | **TASK-OWNED FAIL:** every preceding gate passed, then frontend coverage policy stopped on three not-yet-documented Phase 08 rows: `admin-workflows.ts`, `account-data-client.ts`, and `admin-client.ts`. This directly caused the required coverage disposition work; it was not hidden or called unrelated. |
| `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` (first finding-repair run) | **TASK-OWNED FAIL, resolved:** the new exact backend gate detected `deletionworker/account_deletion.go:68.21-69.14` was nondeterministically uncovered because the cancellation and ticker cases could become ready together. A deterministic scheduler-cancellation regression now covers the branch; ten focused package runs report `100.0%`. No exception was added for test nondeterminism. |
| `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` (final repaired run) | **PASS, exit 0.** Contract/traceability/task list, OpenAPI, 11 coverage-contract regressions, vet, vulnerability scan, local stack/migrations/API, backend tests/race/package and exact Phase 08 coverage, frontend verifier/type drift/typecheck/build/tests/semantic coverage, focused browser suites, full Playwright/axe (`289` passed, `5` intentionally environment-gated skips), and exception-aware report generation all passed. |
| `python3 scripts/validate-task-list.py` | **PASS:** `263` sequential tasks with ordered dependencies. Task list was not edited. |
| `python3 scripts/validate-traceability.py` | **PASS.** Source/design traceability and JSON sidecars are valid. |
| `npx --no-install redocly lint api/openapi.yaml` | **VALID:** one pre-existing accepted `operation-2xx-response` warning for the intentional OAuth callback `302`-only redirect. |
| `cd frontend && bun run check:api-types` | **PASS:** `Generated API types are current.` |
| Aggregate `gofmt -l` check | **PASS:** no backend formatting paths reported. |
| `cd backend && go test ./... -p 1 -count=1` | **PASS:** all command and backend packages. |
| `cd backend && go test -race ./... -p 1 -count=1` | **PASS:** all command and backend packages, including queue/repository live-dependency tests. |
| `cd backend && go test ./internal/... -p 1 -count=1 -coverprofile=coverage.out` | **PASS:** repository aggregate statement coverage `87.4%`; exact package values are below. |
| `cd backend && MEALSWAPP_REDIS_URL=redis://localhost:6379/12 go test -p 1 -count=1 -coverpkg=./internal/... -coverprofile=phase08-coverage.out ./internal/...` | **PASS:** cross-package profile total `90.0%`; deduplicated changed Phase 08 Go scope `4,523/4,841 = 93.4%`. The aggregate now parses this profile and rejects any mismatch from the exact Phase 08 contract. |
| `python3 -m unittest scripts/test_check_coverage.py` | **PASS:** 11 deterministic tests; missing, malformed, over-broad, unjustified, stale-metric, and wrong-phase exception cases are rejected. |
| `cd backend && go vet ./...` | **PASS:** no diagnostics. |
| `cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | **PASS:** zero called vulnerabilities and zero vulnerabilities in imported packages; 18 required-module advisories are not called. |
| `cd frontend && bun run typecheck` | **PASS:** no TypeScript diagnostics. |
| `cd frontend && bun run build` | **PASS:** production Vite build completed. |
| `cd frontend && bun test` | **PASS:** `526` tests, `0` failures, `2,456` expectations. |
| `cd frontend && bun test --coverage` | **PASS with accepted exact exceptions:** `All files | 95.46% funcs | 96.06% lines`; all below-100 runtime rows are documented in `04_OPEN.md`. |
| `python3 scripts/verify-frontend.py --screenshot-stem 08_PHASE_REPORT` | **PASS:** rendered DOM, scripted scenarios, and desktop/mobile screenshots. Controlled 401/500 fixture diagnostics were expected and did not surface unsafe UI state. |
| Focused Phase 06 browser command from `scripts/check.py` | **PASS:** `72/72` desktop/mobile tests. |
| Focused Phase 07 browser command from `scripts/check.py` | **PASS:** `30/30` desktop/mobile tests. |
| `cd frontend && bun run test:e2e` | **PASS:** all `294` scheduled desktop/mobile Chromium cases completed without a failure; explicitly environment-gated real-stack cases remained skipped. The suite includes Phase 08 admin routing, external import, manual/classification/user workflows, private export/deletion, dynamic filters, modal focus containment, responsive themes, and axe scans. |
| `bash scripts/verify-task-261-ui.sh` (first task-262 invocation) | **TRANSIENT TASK-OWNED FAIL:** valid verified-admin claims reached `POST /api/v1/custom-items`, which returned `403`; API logs showed no stale listener or authorization-role mismatch. The result was retained and investigated. |
| `bash scripts/verify-task-261-ui.sh` (immediate rerun plus ten-run loop) | **PASS 11/11:** the real generated-client/Svelte/API/PostgreSQL/Redis flow created and deleted private data, created global classification/item data, and published the dynamic filter. The one-off CSRF-shaped 403 did not reproduce; no implementation exception is claimed. |
| `python3 scripts/test_generate_api_types.py` | **UNRELATED/PRE-EXISTING FAIL:** `23/24` pass; `test_optimization_terminal_vocabulary_and_similarity_projection_match_generated_contract` still expects the older phrase `Quantity-weighted Jaccard similarity`. Task 261 already recorded this stale Phase 07 wording assertion. Current generator output drift, OpenAPI lint, frontend typecheck/build/tests, and Phase 08 contracts pass; task 262 did not alter the optimization wording or test. |
| `command -v gosec` | **ENVIRONMENT NOTE:** no installed `gosec` binary. Task 262 requires vet, race, security integration tests, and the pinned vulnerability scan, all of which passed; no unpinned tool was downloaded. |
| `git diff --check` | **PASS.** |

## Backend coverage evidence

Direct package totals from the final aggregate profile:

| Package | Coverage | Package | Coverage |
|---|---:|---|---:|
| `app` | 84.8% | `cache` | 90.1% |
| `curation` | 90.3% | `customitem` | 90.9% |
| `dataimporter` | 88.3% | `deletionworker` | 100.0% |
| `externaldata` | 99.8% | `httpapi` | 87.3% |
| `itemcurator` | 74.7% | `observability` | 85.6% |
| `repository` | 86.5% | `search` | 96.5% |
| `security` | 99.7% | `tagmanager` | 100.0% |
| `useradmin` | 93.8% | `userdata` | 97.2% |

The exact changed Phase 08 runtime profile below deduplicates identical source blocks emitted by multiple `-coverpkg` test binaries and counts a block covered when any integration package executes it. This is the human-readable line-range summary; the authoritative machine-checked table in Phase 08 of `04_OPEN.md` records exact `line.column-line.column` blocks. Type-only files and `cmd/worker/main.go` are excluded under the documented entrypoint build/smoke rule. Files at 100% are: `cache/user_purger.go`, `deletionworker/account_deletion.go`, all external clients/normalizer except the two rate telemetry branches, `httpapi/admin_controller.go`, `external_search_controller.go`, `filter_option_controller.go`, `repository/allergen_vocabulary_repository.go`, `repository/errors.go`, `repository/postgres.go`, `search/catalog_service.go`, `security/normalizer.go`, `tagmanager/service.go`, and `userdata/deletion.go`.

| Changed runtime file | Statements | Coverage | Exact uncovered profile ranges |
|---|---:|---:|---|
| `internal/app/app.go` | 109/115 | 94.8% | `97-99,115-117,117-119,122-124,157-161` |
| `internal/cache/classification_generation.go` | 14/18 | 77.8% | `53-55,60-62,69-71,78-80` |
| `internal/cache/classification_invalidator.go` | 23/26 | 88.5% | `48-49,49-51,52` |
| `internal/cache/search_cache.go` | 152/167 | 91.0% | `94-96,109-111,115-117,147-149,158-160,162-164,223-225,229-231,275-276,276-280,294-295,295-297,490-492` |
| `internal/curation/validation.go` | 88/93 | 94.6% | `145-147,150-152,155-157,171-174` |
| `internal/customitem/service.go` | 153/165 | 92.7% | `114-116,118-120,129-131,153-155,170-172,174-176,190-192,207-209,243-244,293-295,368-370,395-397` |
| `internal/dataimporter/service.go` | 68/77 | 88.3% | `94-96,99-101,143-145,146-148,160-161,170-172,194-196,198-200,209-211` |
| `internal/externaldata/rate_limit.go` | 184/186 | 98.9% | `348-350,398-400` |
| `internal/httpapi/auth_controller.go` | 94/103 | 91.3% | `78-80,95-97,104-106,107-109,123-125,136-138,149-151,165-167,233-235` |
| `internal/httpapi/classification_admin_controller.go` | 58/77 | 75.3% | `55-57,64-66,74-76,78-80,88-90,92-94,100-102,110-112,114-116,118-120,122-124,126-128,136-138,144-146,148-150,159-161,168-170,178-180,188-190` |
| `internal/httpapi/curation_validation.go` | 99/108 | 91.7% | `122-124,149-151,159-161,168-170,198-200,210-212,214-216,227-229,231-232` |
| `internal/httpapi/custom_item_controller.go` | 99/125 | 79.2% | `30-32,33-35,37-39,53-55,57-59,60-62,74-76,78-80,81-83,85-87,89-91,99-101,103-105,106-108,109-111,119-121,128-130,150-152,153-155,178-180,225,232-234,242-244,268-270,276-277,286-287` |
| `internal/httpapi/import_controller.go` | 45/55 | 81.8% | `56-58,59-61,63-65,67-69,99-101,108-110,134-135,142-143,144-145,151-153` |
| `internal/httpapi/manual_item_controller.go` | 65/88 | 73.9% | `48-50,51-53,55-57,79-81,82-84,86-88,96-98,99-101,103-105,107-109,119-121,122-124,126-128,138-140,147-149,173-174,188-190,219-221,227-228,231-232,235-236,237-238` |
| `internal/httpapi/profile_controller.go` | 34/36 | 94.4% | `58-60,97-99` |
| `internal/httpapi/router.go` | 228/233 | 97.9% | `192-194,313-315,426-428,432-434,502-503` |
| `internal/httpapi/search_validation.go` | 189/193 | 97.9% | `396-398,398-400,415-417` |
| `internal/httpapi/user_admin_controller.go` | 64/79 | 81.0% | `49-51,52-54,56-58,60-62,64-66,74-76,77-79,82-84,89-91,126-128,160-161,162-163,166-167,168-169,181-183` |
| `internal/itemcurator/service.go` | 59/79 | 74.7% | `93-95,97-99,104-106,108-110,121-123,130-132,133-135,146-148,150-152,153-155,157-159,160-162,164-166,173-175,176-178,180-182,183-185,193-195,231-233,245-247` |
| `internal/observability/admin_external.go` | 99/102 | 97.1% | `63-65,235-237,248-249` |
| `internal/repository/admin_user_repository.go` | 45/55 | 81.8% | `60-62,67-69,72-74,81-83,100-102,106-108,126-128,129-131,131-133` |
| `internal/repository/classification_repository.go` | 58/65 | 89.2% | `73-75,99-101,110-112,120-122,153-155,156-158,200-202` |
| `internal/repository/compliance_repository.go` | 272/274 | 99.3% | `642-644,651-653` |
| `internal/repository/curated_import_repository.go` | 98/129 | 76.0% | `64-66,79-81,88-90,93-95,111-113,116-118,119-121,123-125,128-130,134-136,140-142,144-146,157-159,163-165,171-173,177-179,182-184,188-190,192-194,195-197,205-207,208-210,218-220,221-223,254-256,264-266,291-293,307-309,311-313,314-316,318-320` |
| `internal/repository/custom_food_repository.go` | 135/154 | 87.7% | `108-110,112-114,118-121,124-127,132-134,144-146,156-158,164-166,170-172,180-182,211-213,228-230,253-255,262-264,265-267,269-271,272-274` |
| `internal/repository/food_repository.go` | 199/200 | 99.5% | `241-243` |
| `internal/repository/manual_food_repository.go` | 63/86 | 73.3% | `45-47,54-56,63-65,73-75,77-79,81-83,86-88,90-92,96-98,105-107,108-110,112-114,115-117,124-126,128-130,131-133,147-149,160-162,185-187,194-196,197-199,201-203,204-206` |
| `internal/search/filter_options.go` | 79/80 | 98.8% | `98-100` |
| `internal/search/substitution_service.go` | 211/225 | 93.8% | `60-62,159,187-189,191-193,266-268,270-272,275-277,318-321,321-323,334-336,340-342,471-473` |
| `internal/useradmin/service.go` | 61/65 | 93.8% | `115-117,123-125,133-135,195-197` |
| `internal/userdata/export.go` | 93/98 | 94.9% | `108-110,114-116,169-171,176-178,243-245` |

Accepted backend exception: `93.4%` changed-scope coverage is below the 100% goal. The uncovered ranges are defensive dependency/encoder/claim-corruption branches, repeated safe HTTP/repository mappings, cache/configuration fallbacks, and instrumentation paths. Nominal and adversarial acceptance behavior is covered through unit, HTTP, live PostgreSQL/Redis, race, SWE.5, and browser suites. No authorization, ownership, private/global isolation, CSRF, idempotency/replay, parameterized persistence, mutation-plus-audit atomicity, provider bounding/degradation, account erasure, classification invalidation, sanitized observability, or immediate search behavior is waived.

## Frontend coverage evidence

| Phase 08 runtime row | Functions | Lines | Disposition |
|---|---:|---:|---|
| `src/lib/admin-access.ts` | 100.00% | 100.00% | Complete. |
| `src/lib/admin-workflows.ts` | 90.91% | 98.51% | Accepted; Bun emits no stable uncovered-line range. |
| `src/lib/api/account-data-client.ts` | 100.00% | 98.00% | Accepted; Bun emits no stable uncovered-line range. |
| `src/lib/api/admin-client.ts` | 97.22% | 100.00% | Accepted function instrumentation only. |
| `src/lib/api/external-admin-client.ts` | 100.00% | 100.00% | Complete. |
| `src/lib/api/filter-options-client.ts` | 100.00% | 100.00% | Complete. |
| `src/lib/api/generated.ts` | 100.00% | 98.98% | Accepted generated fallback line `185`; regenerated Phase 08 contracts shifted the prior exact Phase 07 measurement. |
| `src/lib/shell-routing.ts` | 100.00% | 100.00% | Complete. |
| `src/lib/substitution-filter-options.ts` | 100.00% | 100.00% | Complete. |

Svelte components do not emit Bun runtime rows. Their behavior is covered by component tests and the complete Playwright/axe desktop/mobile suite. The exact frontend exception is recorded under Phase 08 in `04_OPEN.md`; no generated-contract decoder, fail-closed access state, authoritative refresh, destructive confirmation, degraded state, or accessibility behavior is waived.

## Phase 08 disposition

Every Phase 08 entry in `docs/implementation/04_OPEN.md` now has a terminal evidence-backed disposition:

- all three planning assumptions are `CONFIRMED` by Tasks 238-252 and current aggregate/race evidence;
- the Task 239 provisional exception and Task 250 open deviation are `SUPERSEDED` by the final backend aggregate exception;
- the final backend and frontend below-100 measurements are precise `ACCEPTED EXCEPTION` entries;
- the carried Phase 03 custom-item persistence/export/erasure action is `CLOSED` by Tasks 238-240;
- the carried Phase 05 hardcoded-filter action is `CLOSED` by Tasks 241, 251, 253, and 257.

No Phase 08 assumption, coverage deviation, or action remains `OPEN`.

## Failure classification and security disposition

- **Task-owned and resolved:** initial aggregate frontend policy failure. Resolution was exact coverage disposition followed by a complete passing aggregate rerun.
- **Task-owned transient and disclosed:** one real-stack CSRF-shaped 403. The immediate rerun and ten consecutive clean-stack repetitions passed (11/11 after the failure); valid admin/verified claims and a current API PID were confirmed. It is not an accepted implementation exception and no unsafe retry was added.
- **Accepted precise exceptions:** backend 93.4% changed-scope coverage and the exact frontend rows above; both are enforced against current measured output by the Phase 08 contracts in `04_OPEN.md`, with non-waived behavior and focused/live/browser evidence categories.
- **Accepted existing contract warning:** OAuth callback intentionally has a `302` response and no 2XX response.
- **Concurrent/pre-existing unrelated failure:** the stale Phase 07 optimization-description assertion in `scripts/test_generate_api_types.py`; executable OpenAPI/type drift passes.
- **Controlled diagnostics, not failures:** browser fixture proxy connection messages and expected 401/500 responses exercise degraded/anonymous/error states; all browser and axe commands passed.
- **Security:** admin/user authorization, CSRF, input bounds, parameterized SQL, audit rollback, privacy-safe observability, race detection, vet, and reachable dependency scanning passed. `govulncheck` found zero called/imported vulnerabilities. `gosec` is absent and was not an acceptance criterion or dynamically installed exception.

## Evidence hashes

| Evidence | SHA-256 |
|---|---|
| `docs/implementation/implemented/08_PHASE_REPORT.html` (`778,396` bytes; includes exact accepted-exception tables) | `5b77dc62452a3a7cda9756cfca89efdf1f223ccd492fb00b4da98953bdc5d03a` |
| All 20 `08_PHASE_REPORT-*.png` screenshot hash lines, hashed in sorted shell-glob order | `031efe191cdac5782f13a5b49681ec967b0e7cc95e700fc590d21a6a6a669564` |
| Representative desktop screenshot (`33,936` bytes) | `5e53c2313bd3ee25e9136451d5909fcb58823589191220f299069cc98bf5f875` |
| Representative mobile screenshot (`18,125` bytes) | `f77d1561526a35e86b63404af157b037b40e908a92e08f7ce72038f9ddf943d2` |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `frontend/src/lib/api/generated.ts` | `f732d86079c10056959292ad2dea3c0163b83b43185620169cd243e074c7829a` |
| Original pre-review Phase 08 changed implementation/test/contract manifest (196 files; retained as historical preparation evidence and superseded for repaired files below) | `4186bb6aec5737322cc35bc1a45da57f81531f67cc17d985fe1cefbc41d3b345` |
| `scripts/check.py` | `823c29736665bc876b368a9027c0fe101c7968dcf4baa90eaa561d8c1b18039e` |
| `scripts/generate_report.py` | `370bd770b65ca94c1c12295611730a093bc8583cdde25678a83dc924b82aceef` |
| `scripts/test_check_coverage.py` | `bd044b1cef28153c339718d55e402b7f21b763cb9743a4ad7930584e9ae33cdf` |
| `backend/internal/deletionworker/account_deletion_test.go` | `96256c81a99438397631dd7676413f232f4190542e3205295df9b25efc1c8dc6` |
| `docs/implementation/02_TASK_LIST.md` (read-only during repair) | `2df7e143fc22b40cd0ba5ce077ab3559f76b928e9a5a958d6256bc728857f247` |
| `docs/implementation/04_OPEN.md` | `fb63852a3d5bdd128a46db90c62b7a2f89bd6a8de7061d43b0dc29b44063fc9f` |

The preparation document cannot contain its own stable digest; its final SHA-256 is reported with the handoff after the last validation pass.
