# Task 282 preparation evidence

prepared_at_utc: 2026-07-28T07:46:20Z
task_id: 282
task_status_observed: PREPARED
component: Phase 08.02 External Curation and Provider Acceptance Scenarios
static_aspect: DESIGN-009 DataImporter
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677
baseline_confidence: HIGH
test_delivery_result: COMPLETE
requirement_results: SW-REQ-033=FAIL, SW-REQ-055=FAIL, SW-REQ-090=BLOCKED

## Outcome

Task 282 now owns a disposable real-stack acceptance harness for all 24 mapped
SW-REQ-033, SW-REQ-055, and SW-REQ-090 criteria. It starts controlled USDA and
OpenFoodFacts HTTP fixtures, production API composition, PostgreSQL, a
run-labeled Redis instance, the production frontend, and Chromium. The
Administration browser scenarios use the generated frontend contracts and do
not intercept provider, application API, Catalog, or Substitution routes.

Test delivery is complete. Requirement acceptance is truthfully non-pass:

| Requirement | Result | Exit | Criteria | Non-pass evidence |
|---|---:|---:|---:|---|
| SW-REQ-033 | FAIL | 1 | 4 PASS, 1 FAIL | `P08-SWR033-STEP-02`: `ROOT-T282-USDA-OPTIONAL-PORTION`, `P08-FIND-282-002` |
| SW-REQ-055 | FAIL | 1 | 13 PASS, 1 FAIL | `P08-SWR055-STEP-02`: `ROOT-T282-OFF-METADATA`, `P08-FIND-282-001` |
| SW-REQ-090 | BLOCKED | 2 | 4 PASS, 1 BLOCKED | `P08-SWR090-STEP-04`: `ROOT-T282-VOCABULARY-DISABLE`, `P08-FIND-282-004` |

The accepted flow persisted exactly one edited, ownerless food, one curated
import, and one `import_food` audit. Redis was reachable and the imported item
was discovered through real Catalog UI and Substitution API behavior. Search
and all rejected/fault scenarios left no extra curated persistence.

The final isolated repair run is `d08e098642ba3e9c87de8721`. Its sanitized producer
evidence is under
`logs/real-stack-e2e/d08e098642ba3e9c87de8721/acceptance/`. Task 280 finalized
three requirement reports:

- `logs/phase08-acceptance/task282-d08e098642ba3e9c87de8721-sw-req-033/report.json`
- `logs/phase08-acceptance/task282-d08e098642ba3e9c87de8721-sw-req-055/report.json`
- `logs/phase08-acceptance/task282-d08e098642ba3e9c87de8721-sw-req-090/report.json`

The authoritative Task 282 status remains `PREPARED`; this repair did not edit
the task-list row or any task-list status.

## Review repair outcome

All findings in `task-282-review.md` are repaired without changing provider,
curation, parser-defect, or vocabulary-blocker coverage:

- The managed Vite proxy now drops exactly the first committed import response
  outside the browser route layer. The Administration UI reaches its ambiguous
  state and retries through the generated client with the original UUID key and
  byte-identical body. `P08-SWR055-ACCEPT-06` is PASS and
  `P08-FIND-282-003` is CLOSED with fresh synchronized evidence.
- The global identity query projects `food_items` with `NULL::uuid AS owner_id`,
  unions private `custom_food_items.owner_id`, and requires
  `owner_id IS NULL`; focused validation regressions prove a private-only
  identity cannot satisfy the evidence predicate.
- Task 282 now writes a private run-scoped capability manifest and Playwright
  validates schema, run ID, nonce, evidence root, exact loopback URL, capability
  ownership/mode, frontend PID/start token, cwd, strict port, and command line.
  Missing capability and foreign base URL regressions fail closed.
- `combine_results` treats any duplicate browser criterion ID as malformed
  producer output and blocks the complete manifest with
  `ROOT-T282-ACCEPTANCE-INFRASTRUCTURE`; PASS/FAIL ordering cannot overwrite a
  prior result or root.
- A caught product-level Playwright non-pass now yields
  `diagnostics.result=product_nonpass`. The final run retained truthful
  SW-REQ-033 FAIL, SW-REQ-055 FAIL, and SW-REQ-090 BLOCKED reports while its
  isolated lifecycle and cleanup completed.

## Orchestration protocol and preserved worktree

The phase-orchestrator protocol was loaded before implementation. The
configured developer-agent delegation surface was not available in this
session, and a local Codex CLI delegation attempt could not start because its
installed npm package was already in a conflicting update state. The parent
agent therefore performed the required isolated implementation, verification,
function-level review, and evidence capture directly. No task status was
changed in lieu of an independent subagent review.

- Repository: `/home/wiktor/Work/mealswapp`
- HEAD and fixed baseline: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Initial worktree: dirty with Tasks 276-281 and other Phase 08 work.
- Existing tracked and untracked changes were preserved. No reset, checkout,
  clean, revert, or broad rewrite was used.
- Task-owned edits were restricted to provider fixture configuration,
  production client composition, the Task 282 producer/harness/contracts,
  Task 280 reporting reuse, exact coverage metadata shifted by composition,
  and synchronized finding records.
- `P08-FIND-282-005` reserves the grouped
  `ROOT-T282-ACCEPTANCE-INFRASTRUCTURE` blocker for lifecycle or malformed
  producer failures. It is absent from the successful-infrastructure final
  run.

## Function-level implementation evidence

### Production client fixture seam

- `backend/internal/config/config.go`
  - `ExternalDataConfig` defines USDA/OFF endpoint and request deadline
    overrides.
  - `loadExternalDataConfig` accepts only development/test loopback HTTP
    endpoints with no credentials, query, or fragment; rejects overrides in
    production; and bounds the provider deadline.
  - `Load` composes the validated external-data configuration.
- `backend/internal/config/config_test.go`
  - `TestLoadExternalDataFixtureOverrides` proves accepted loopback fixtures.
  - `TestLoadRejectsUnsafeExternalDataOverrides` covers production, remote,
    credentialed, and invalid deadline rejection.
- `backend/internal/app/app.go`
  - `newProduction` passes the validated endpoint/deadline into the concrete
    `NewUSDAClient` and `NewOpenFoodFactsClient`; the test harness therefore
    exercises production parsing, normalization, quota, and cancellation code.

### Controlled provider

- `scripts/task282_provider_fixture.py`
  - `usda_food` builds canonical USDA success and optional-portion variants.
  - `off_product` builds normal, legitimate-metadata, and malformed-supported-
    nutrient OpenFoodFacts variants.
  - `Handler.do_GET` serves health, each-provider success, both-provider merge,
    partial failure, complete outage, malformed envelope, malformed consumed
    nutrient, timeout/cancellation, stale response, and quota/reset responses.
  - `Handler._json`, `Handler.log_message`, and `main` provide deterministic,
    dependency-free fixture execution.

### Production browser scenarios

- `frontend/tests/task282-external-curation.spec.ts`
  - `openAdministration` and `search` drive the production administration UI.
  - The primary scenario searches USDA, OpenFoodFacts, and both providers;
    edits the candidate; assigns classifications; imports with confirmation;
    replays the identical idempotency request; verifies edited persistence;
    and observes the item in Catalog and Substitution.
  - The fault scenario covers partial/complete failure, malformed envelopes,
    malformed consumed data, timeout, cancellation/stale suppression,
    quota/reset, safe warnings, and absence of raw provider payloads/secrets.
  - Dedicated tests retain FAIL evidence for legitimate OFF metadata and
    missing optional USDA portions.
  - The primary production scenario loses the first committed import response,
    observes the safe ambiguous UI, and retries the identical key/body through
    the generated client. Disabled-vocabulary administration remains
    truthfully BLOCKED because that production capability does not exist.
  - Canonical, alias, and unknown micronutrient request cases exercise the
    generated API contracts and production validation.
- `frontend/tests/phase08-acceptance-reporter.ts`
  - `onEnd` now supports a caller-scoped infrastructure root and synchronized
    root allowlist while retaining fail-closed result aggregation.
- `frontend/playwright.real-stack.config.ts`
  - managed Task 282 runs require the exact private harness capability and
    activate the Phase 08 reporter; ordinary frontend gates continue to skip
    managed-only real-stack scenarios.
- `frontend/vite.config.ts`
  - the run-owned loopback dev proxy provides the one-shot committed-response
    loss boundary only when the Task 282 harness explicitly enables it.

### Isolated lifecycle and Task 280 synchronization

- `scripts/run-task282-acceptance.py`
  - `Task282Harness.application_environment` injects only the private loopback
    provider endpoints and bounded timeout.
  - `Task282Harness.execute` reserves isolated ports, creates disposable
    PostgreSQL/Redis state, starts the provider/API/frontend processes,
    bootstraps an administrator, grants entitlement, runs Playwright, captures
    evidence, publishes reports, and always invokes inherited cleanup.
  - `write_backend_evidence` queries exact food/import/audit/edited counts,
    ownerless identity through a global/private owner projection, and Redis
    `PONG`.
  - `combine_results` emits exactly 24 fail-closed criterion results, rejects
    duplicate IDs, and preserves only synchronized product roots.
  - `finalize_reports` invokes Task 280 once per requirement and propagates
    truthful 0/1/2 semantics.
  - `parse_args` and `main` provide the operator entry point and lifecycle
    fallback.
- `scripts/test_task282_acceptance.py`
  - proves 24 unique criteria, complete fail-closed synthesis, synchronized
    known roots, rejection of unknown roots, and fixture response shapes.
- `scripts/check.py`
  - registers Task 282 contracts and traceability paths in focused/full gates.
- `scripts/test_check_coverage.py`
  - tracks the unchanged `110/116` app composition coverage with line ranges
    shifted by production provider wiring.

### Transactional persistence scenarios reused at the production boundary

`backend/internal/dataimporter/integration_test.go` already contains
`TestCuratedImportTransactionalWorkflow` against PostgreSQL and the production
DataImporter. The focused/full backend runs execute its exact natural-key and
idempotency replay, changed-body key conflict, name conflict and confirmed
merge, audit failure rollback, repository failure rollback, recovery, exact
food/import/audit counts, ownerless identity, and Catalog/Substitution
visibility assertions. Task 282 composes that transactional evidence with the
browser/provider run instead of duplicating or bypassing the production
service.

## Scenario and acceptance coverage

- Success: individual USDA/OFF search, deterministic both-provider merge,
  edited curation, classification selection, confirmation, import, replay, and
  local discovery pass.
- Read-only boundary: all provider searches and rejected fault cases precede
  confirmation and produce no additional local food/import/audit rows.
- Normalization: solid data persists on `100g`; the accepted flow does not
  invent density; canonical micronutrients pass while alias/unknown keys fail.
- Provider safety: partial failure remains usable; complete outage, malformed
  envelope, malformed supported nutrient, timeout, cancellation/stale result,
  and quota/reset produce safe bounded errors without raw payload or secret
  leakage.
- Idempotency and rollback: exact replay commits once; changed body, name
  conflict, audit/repository failure, rollback, and recovery are covered by
  the production DataImporter integration workflow.
- Known defects/capability gaps remain non-pass: OFF legitimate metadata,
  optional USDA portion degradation, and vocabulary disablement are
  synchronized to owners, evidence, and retest instructions. Response-loss
  recovery now passes with its original key and body.

## Verification

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_task282_acceptance.py scripts/test_phase08_acceptance.py scripts/test_task281_acceptance.py` | PASS, 44 tests |
| `python3 scripts/phase08_acceptance.py validate` | PASS, 12 scenarios and 91 criteria |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks |
| `python3 scripts/validate-traceability.py` | PASS |
| `cd backend && go test -count=1 ./internal/config ./internal/externaldata ./internal/dataimporter ./internal/app` | PASS |
| `cd backend && go test -race ./internal/externaldata ./internal/dataimporter` | PASS |
| `cd frontend && bun run check` | PASS, unit/build/typecheck/API drift |
| `python3 scripts/run-task282-acceptance.py` | Final run `d08e098642ba3e9c87de8721`: P08-SWR055-ACCEPT-06 PASS; product result SW-REQ-033 FAIL, SW-REQ-055 FAIL, SW-REQ-090 BLOCKED; diagnostics `product_nonpass`; infrastructure and cleanup PASS |
| `python3 scripts/check.py --quick` | PASS, including changed Playwright, backend, static analysis, and vulnerability scan |
| `python3 scripts/check.py` | PASS; 309 browser tests passed, 29 managed-only/real-stack tests skipped in the ordinary gate; full static/frontend/backend/browser/coverage lanes passed |

The fresh repair quick and full gates passed without changing the documented
`internal/app/app.go` measured `110/116` coverage result.

## SHA-256 inventory

| Path | SHA-256 |
|---|---|
| `backend/internal/config/config.go` | `a089b35f800d9b308727e076b54f87155d6df92ad25b5c610ed5723c12078f44` |
| `backend/internal/config/config_test.go` | `e8b4370c7bcca6d4d439c94aeae5c0873e1d9c4c719623c2a0ed86a7213ef344` |
| `backend/internal/app/app.go` | `29f575dc974b5170b229f62d8cbaa34dc936dde25b72895cf52c7b3b473d0307` |
| `scripts/task282_provider_fixture.py` | `871844c12919b250b3da34bc7616787dcc4b9e702b606e80f170d507e1416c1c` |
| `scripts/run-task282-acceptance.py` | `653bec46d4f5a5966fc5c5c75ac33171f923385728dae2f119dbfb7e8b933966` |
| `scripts/test_task282_acceptance.py` | `0e2da5cfd23fa3197de7f6bf64757b786ee77fce84a7c532c19759f139912a40` |
| `frontend/vite.config.ts` | `5938c937f565e0b2a94d5ad8788ecebf36ad092cfeb6c986c0e591b27562857c` |
| `frontend/playwright.real-stack.config.ts` | `c964611230ceb902df37a3de5e843cb11f76451e478e0ca786f7805c14766471` |
| `frontend/tests/phase08-acceptance-reporter.ts` | `3f565b3e5e6009549b47d09bb9bbe75effc524f9cfc56762dd11f4557c6fe8e6` |
| `frontend/tests/task282-external-curation.spec.ts` | `70b349dd936ae958cd5a9427f2beb49f92c5969ca72b48e682f733660d2bad09` |
| `scripts/check.py` | `77d35bb4967eacff99edffc9f2c4abc9020c281164c4cb9d31c7ebe8e1b1fa89` |
| `scripts/test_check_coverage.py` | `f9d45d91f3aa41109670e6496bf10c4ed2fbf20b0f573d9120dc1a369a62fb0b` |
| `docs/implementation/04_OPEN.md` | `19f45e0a7551fa55785bfa3dd6e390ef1e0ef52af62b8221fdb09bdc92952ab9` |
| `docs/testing/phase08/finding-history.json` | `ef0cb5698dec68555ab5e46594f2d580dfb20c210bb057925448184b656cecd0` |
| final diagnostics `diagnostics.json` | `d036dd3fef5b7107fbb3d234f5769820a668163cab9447b9f2d31c64c48bf801` |
| final producer `acceptance/results.json` | `e1fa731c8f3ad0ea3c1b3bd3bcca01beac49107558c8b66cf6004a8483242a35` |
| final producer `acceptance/backend/curation.json` | `c0c41f703cab7ddc08c43e33ec1561a40aefbff9fc8ee1e5e699ebab2ea1d91b` |
| final SW-REQ-033 report | `2729a7c749e47f844753a9b03e3df900c3b1a59695ac3b480d9ad917fab55e04` |
| final SW-REQ-055 report | `a0e2fdf3d1198a7ebe1f0f90cb77a035ff9c319769957d32e4f82e641bc2f6dc` |
| final SW-REQ-090 report | `d3d35a54ddc5deb8f419bfa0ff626fb96c01fb5e23a28730160cd13bc783f80e` |

This evidence document is intentionally excluded from its own hash inventory.
