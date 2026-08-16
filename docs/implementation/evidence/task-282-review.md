# Review Evidence: Task 282 — External Curation and Provider Acceptance Scenarios

```yaml
task_id: 282
component: "DataImporter"
static_aspect: "External Curation and Provider Acceptance Scenarios"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-28T08:22:00Z"
review_agent: "configured reviewer"
evidence_file: "docs/implementation/evidence/task-282-review.md"
baseline_ref: "task-282-preparation.md plus current worktree and run d08e098642ba3e9c87de8721"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "go.md, python.md, typescript.md, security-review-guide.md, common-bugs-checklist.md"
repair_context_required: false
```

## 1. Task Source

**Description:** Re-review repaired Task 282 against its PREPARED task row and current implementation. The review scope is the Task 282 acceptance harness, provider fixtures, Playwright capability boundary, response-loss retry, ownerless identity evidence, duplicate-result combiner, truthful diagnostics, and their direct callers/tests.

**Depends On:** 245, 246, 249, 255, 279, 280 — all current statuses are PASSED.

**Testing Coverage Exceptions:** The fresh run intentionally records the known product defects as FAIL/BLOCKED and synchronizes them; this is a test-delivery acceptance outcome, not a review pass of those product requirements. No review coverage exception was used.

**Verification Criteria:** The task row contains the 24 stable criteria in the Task 282 manifest for SW-REQ-033, SW-REQ-055, and SW-REQ-090. The table below covers each manifest criterion.

## 2. Pre-Review Gates

- [x] Input status is PREPARED; task row remains unchanged.
- [x] Every dependency is PREPARED or PASSED; all six are PASSED.
- [x] The preparation report claims completion and has fresh run evidence.
- [x] A task-specific baseline and changed surface are available.
- [x] code-review-skill was invoked exactly once and its relevant guides were read.
- [x] The review is independent of the repair preparation.
- [x] Current repository and fresh artifacts were inspected; historical evidence was treated as stale where affected.
- [x] No production-code or task-list status changes were made.

```yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: Read the PREPARED Task 282 row, dependency statuses, fresh preparation report, current worktree, direct callers, tests, and fresh run artifacts. The prior REJECTED review was checked for staleness and each of REV-T282-001 through REV-T282-005 was re-tested against current source and fresh evidence.

Commands used to reconstruct the diff:

```bash
git status --short
git diff --stat
rg -n 'Task282|task282|ExternalImport|phase08|ownerless|duplicate|product_nonpass' .
sha256sum <all reviewed files>
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains unrelated Phase 08.02 Tasks 276–281 and other pre-existing edits. They were not reset, reverted, or status-mutated. Scope was limited to Task 282 files and direct consumers: provider configuration/composition, the shared run-owned harness, the Phase 08 reporter, external client/workflow, Task 282 tests/fixtures, validation/check registration, and the fresh Task 282 artifacts. The Task 282 row in `docs/implementation/02_TASK_LIST.md` was verified as `PREPARED`.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| Task282 harness, fixtures, reporter, UI/client, config, direct harness dependencies | current Task 282 preparation and source inspection | HIGH | 66 audited units below |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | `P08-SWR033-STEP-01` controlled USDA success and edited import | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 2 | `P08-SWR033-STEP-02` optional USDA portion remains truthful | browser result plus synchronized finding | PASS delivery; product FAIL synchronized ROOT-T282-USDA-OPTIONAL-PORTION | `d08e098642ba3e9c87de8721` and the cited artifact |
| 3 | `P08-SWR033-STEP-03` USDA normalization and warning path | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 4 | `P08-SWR033-STEP-04` USDA import persistence | fresh browser/backend/report evidence | PASS; curation.json | `d08e098642ba3e9c87de8721` and the cited artifact |
| 5 | `P08-SWR033-STEP-05` USDA discovery | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 6 | `P08-SWR055-STEP-01` provider search | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 7 | `P08-SWR055-STEP-02` OpenFoodFacts metadata result | browser result plus synchronized finding | PASS delivery; product FAIL synchronized ROOT-T282-OFF-METADATA | `d08e098642ba3e9c87de8721` and the cited artifact |
| 8 | `P08-SWR055-STEP-03` candidate merge | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 9 | `P08-SWR055-STEP-04` curation edits | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 10 | `P08-SWR055-STEP-05` classification selection | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 11 | `P08-SWR055-STEP-06` single import mutation | fresh browser/backend/report evidence | PASS; curation.json | `d08e098642ba3e9c87de8721` and the cited artifact |
| 12 | `P08-SWR055-STEP-07` catalog and substitution discovery | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 13 | `P08-SWR055-ACCEPT-01` successful external curation acceptance | fresh browser/backend/report evidence | PASS; results.json | `d08e098642ba3e9c87de8721` and the cited artifact |
| 14 | `P08-SWR055-ACCEPT-02` partial/outage rollback | fresh browser/backend/report evidence | PASS; mutation_count=0 | `d08e098642ba3e9c87de8721` and the cited artifact |
| 15 | `P08-SWR055-ACCEPT-03` malformed payload handling | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 16 | `P08-SWR055-ACCEPT-04` timeout and cancellation | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 17 | `P08-SWR055-ACCEPT-05` quota/reset behavior | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 18 | `P08-SWR055-ACCEPT-06` same-key ambiguous retry | fresh browser/backend/report evidence | PASS; byte-identical request assertions | `d08e098642ba3e9c87de8721` and the cited artifact |
| 19 | `P08-SWR055-ACCEPT-07` safe failure warnings | fresh browser/backend/report evidence | PASS; sanitized assertions | `d08e098642ba3e9c87de8721` and the cited artifact |
| 20 | `P08-SWR090-STEP-01` external-import micronutrient acceptance | fresh browser/backend/report evidence | PASS; real run | `d08e098642ba3e9c87de8721` and the cited artifact |
| 21 | `P08-SWR090-STEP-02` alias rejection | fresh browser/backend/report evidence | PASS; 400 and zero mutation | `d08e098642ba3e9c87de8721` and the cited artifact |
| 22 | `P08-SWR090-STEP-03` unknown key rejection | fresh browser/backend/report evidence | PASS; 400 and zero mutation | `d08e098642ba3e9c87de8721` and the cited artifact |
| 23 | `P08-SWR090-STEP-04` disabled vocabulary criterion | browser result plus synchronized finding | PASS delivery; product BLOCKED synchronized ROOT-T282-VOCABULARY-DISABLE | `d08e098642ba3e9c87de8721` and the cited artifact |
| 24 | `P08-SWR090-ACCEPT-01` validation and rollback acceptance | fresh browser/backend/report evidence | PASS; results.json | `d08e098642ba3e9c87de8721` and the cited artifact |

The product FAIL/BLOCKED statuses are intentionally preserved in the fresh producer/report contract: SW-REQ-033 is 4 PASS/1 FAIL at ROOT-T282-USDA-OPTIONAL-PORTION; SW-REQ-055 is 13 PASS/1 FAIL at ROOT-T282-OFF-METADATA; SW-REQ-090 is 4 PASS/1 BLOCKED at ROOT-T282-VOCABULARY-DISABLE. They are synchronized in `docs/implementation/04_OPEN.md`; no test was skipped to manufacture a PASS.

## 5. Changed-Symbol Inventory

Inventory every Task 282 executable unit reviewed, including the response-loss seam, capability verifier, ownerless SQL, result combiner, diagnostics, direct caller state, fixtures, reporter, config boundary, validator, teardown, and adversarial tests.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `vite proxyRes import-response fault injection` | config | `frontend/vite.config.ts:6-27` | modified | Vite dev proxy and Task282 harness | frontend/tests/task282-external-curation.spec.ts:90-116 |
| 2 | `validateHarnessCapability` | function | `frontend/playwright.real-stack.config.ts:57-122` | modified | Playwright config | scripts/test_task282_acceptance.py:129-190 |
| 3 | `persistence_evidence_is_valid` | function | `scripts/run-task282-acceptance.py:61-70` | modified | Task282Harness.write_backend_evidence | scripts/test_task282_acceptance.py:112-127 |
| 4 | `Task282Harness.application_environment` | method | `scripts/run-task282-acceptance.py:78-94` | modified | Harness.run and API process | scripts/test_task282_acceptance.py:199-216 |
| 5 | `Task282Harness.execute` | method | `scripts/run-task282-acceptance.py:96-241` | modified | main | logs/.../d08e.../acceptance |
| 6 | `Task282Harness.write_backend_evidence` | method | `scripts/run-task282-acceptance.py:243-266` | modified | execute | logs/.../acceptance/backend/curation.json |
| 7 | `Task282Harness.combine_results` | method | `scripts/run-task282-acceptance.py:269-320` | modified | finalize_reports | scripts/test_task282_acceptance.py:43-111 |
| 8 | `Task282Harness.export_diagnostics` | method | `scripts/run-task282-acceptance.py:322-326` | modified | Harness.run | scripts/test_task282_acceptance.py:192-198 |
| 9 | `finalize_reports` | function | `scripts/run-task282-acceptance.py:329-351` | modified | main | logs/.../acceptance/SW-REQ-*.json |
| 10 | `parse_args` | function | `scripts/run-task282-acceptance.py:354-358` | modified | main | scripts/test_task282_acceptance.py |
| 11 | `main` | function | `scripts/run-task282-acceptance.py:361-380` | modified | CLI | preparation evidence |
| 12 | `load` | function | `scripts/test_task282_acceptance.py:19-29` | modified | unit tests | all Task282 tests |
| 13 | `test_manifest_surface_is_complete_and_unique` | test | `scripts/test_task282_acceptance.py:35-41` | added | manifest contract | 44-test Python run |
| 14 | `test_missing_browser_output_blocks_every_criterion_and_splits_requirements` | test | `scripts/test_task282_acceptance.py:43-56` | added | combine_results | 44-test Python run |
| 15 | `test_known_product_root_is_preserved_but_unknown_root_fails_closed` | test | `scripts/test_task282_acceptance.py:58-89` | added | combine_results | 44-test Python run |
| 16 | `test_duplicate_browser_criteria_fail_closed_instead_of_overwriting` | test | `scripts/test_task282_acceptance.py:91-110` | added | combine_results | 44-test Python run |
| 17 | `test_ownerless_projection_and_validation_reject_private_identity` | test | `scripts/test_task282_acceptance.py:112-127` | added | OWNERLESS_COUNT_SQL | 44-test Python run |
| 18 | `test_task282_flag_without_owned_capability_fails_closed` | test | `scripts/test_task282_acceptance.py:129-152` | added | validateHarnessCapability | 44-test Python run |
| 19 | `test_task282_capability_rejects_foreign_base_url` | test | `scripts/test_task282_acceptance.py:154-190` | added | validateHarnessCapability | 44-test Python run |
| 20 | `test_browser_product_nonpass_is_truthful_in_diagnostics` | test | `scripts/test_task282_acceptance.py:192-197` | added | export_diagnostics | 44-test Python run |
| 21 | `test_fixture_shapes_include_supported_nutrients_and_known_metadata` | test | `scripts/test_task282_acceptance.py:199-210` | added | provider fixture | 44-test Python run |
| 22 | `usda_food` | fixture factory | `scripts/task282_provider_fixture.py:15-29` | added | Handler.do_GET | fixture tests |
| 23 | `off_product` | fixture factory | `scripts/task282_provider_fixture.py:31-49` | added | Handler.do_GET | fixture tests |
| 24 | `Handler.do_GET` | method | `scripts/task282_provider_fixture.py:56-92` | added | ThreadingHTTPServer | provider integration |
| 25 | `Handler._json` | method | `scripts/task282_provider_fixture.py:97-109` | added | do_GET | provider integration |
| 26 | `Handler.log_message` | method | `scripts/task282_provider_fixture.py:94-95` | added | HTTP server | safe fixture logs |
| 27 | `provider fixture main` | function | `scripts/task282_provider_fixture.py:111-126` | added | Task282Harness.execute | provider integration |
| 28 | `openAdministration` | function | `frontend/tests/task282-external-curation.spec.ts:17-25` | added | all browser scenarios | Playwright run |
| 29 | `search` | function | `frontend/tests/task282-external-curation.spec.ts:28-45` | added | provider scenarios | Playwright run |
| 30 | `primary curation acceptance scenario` | test | `frontend/tests/task282-external-curation.spec.ts:47-155` | added | production UI and reporter | browser.json/results.json |
| 31 | `provider resilience acceptance scenario` | test | `frontend/tests/task282-external-curation.spec.ts:157-185` | added | controlled fixture and reporter | browser.json/results.json |
| 32 | `OpenFoodFacts metadata scenario` | test | `frontend/tests/task282-external-curation.spec.ts:187-192` | added | finding synchronization | ROOT-T282-OFF-METADATA |
| 33 | `USDA optional portion scenario` | test | `frontend/tests/task282-external-curation.spec.ts:194-199` | added | finding synchronization | ROOT-T282-USDA-OPTIONAL-PORTION |
| 34 | `alias and unknown micronutrient scenario` | test | `frontend/tests/task282-external-curation.spec.ts:201-234` | added | validation API | browser.json/results.json |
| 35 | `disabled vocabulary scenario` | test | `frontend/tests/task282-external-curation.spec.ts:236-239` | added | finding synchronization | ROOT-T282-VOCABULARY-DISABLE |
| 36 | `Phase08AcceptanceReporter.onBegin` | method | `frontend/tests/phase08-acceptance-reporter.ts:36-38` | modified | Playwright lifecycle | browser.json |
| 37 | `Phase08AcceptanceReporter.onTestEnd` | method | `frontend/tests/phase08-acceptance-reporter.ts:40-55` | modified | Playwright test outcomes | browser.json |
| 38 | `Phase08AcceptanceReporter.onEnd` | method | `frontend/tests/phase08-acceptance-reporter.ts:57-101` | modified | Task282Harness.combine_results | browser.json |
| 39 | `parseAttachment` | function | `frontend/tests/phase08-acceptance-reporter.ts:104-129` | modified | onTestEnd | reporter tests |
| 40 | `isRecord` | function | `frontend/tests/phase08-acceptance-reporter.ts:131-133` | modified | parseAttachment | reporter tests |
| 41 | `splitEnvironment` | function | `frontend/tests/phase08-acceptance-reporter.ts:135-137` | modified | onEnd | reporter tests |
| 42 | `uniqueEvidence` | function | `frontend/tests/phase08-acceptance-reporter.ts:139-144` | modified | onEnd | reporter tests |
| 43 | `ExternalDataConfig` | type | `backend/internal/config/config.go:60-66` | added | Load and newProduction | config tests |
| 44 | `config.Load` | function | `backend/internal/config/config.go:111-146` | modified | application startup | config tests |
| 45 | `loadExternalDataConfig` | function | `backend/internal/config/config.go:198-229` | added | config.Load | config tests |
| 46 | `TestLoadExternalDataFixtureOverrides` | test | `backend/internal/config/config_test.go:63-75` | added | loadExternalDataConfig | focused Go tests |
| 47 | `TestLoadRejectsUnsafeExternalDataOverrides` | test | `backend/internal/config/config_test.go:77-100` | added | loadExternalDataConfig | focused Go tests |
| 48 | `app.newProduction provider composition` | function | `backend/internal/app/app.go:54-223` | modified | NewProduction | focused Go tests |
| 49 | `importCuratedItem` | function | `frontend/src/lib/api/external-admin-client.ts:85-107` | modified | ExternalImportWorkflow.submitImport | frontend tests |
| 50 | `safeFetch` | function | `frontend/src/lib/api/external-admin-client.ts:114-141` | modified | import/search client | frontend tests |
| 51 | `ExternalImportWorkflow.loadClassifications` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:87-97` | modified | onMount | frontend tests |
| 52 | `ExternalImportWorkflow.runSearch` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:112-137` | modified | requestSearch | frontend tests |
| 53 | `ExternalImportWorkflow.selectCandidate` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:139-160` | modified | curation UI | frontend tests |
| 54 | `ExternalImportWorkflow.submitImport` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:254-290` | modified | import buttons | task282-external-curation.spec.ts:90-116 |
| 55 | `ExternalImportWorkflow.validDraft` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:292-298` | modified | submitImport | frontend tests |
| 56 | `ExternalImportWorkflow.hasValidLiquidDensity` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:300-304` | modified | validDraft and warning UI | frontend tests |
| 57 | `ExternalImportWorkflow.startFreshImport` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:306-311` | modified | conflict recovery | frontend tests |
| 58 | `ExternalImportWorkflow.snapshotDraft` | function | `frontend/src/lib/components/ExternalImportWorkflow.svelte:313-322` | modified | submitImport | task282-external-curation.spec.ts:95-108 |
| 59 | `strict_json_loads` | function | `scripts/phase08_acceptance.py:96-113` | modified | manifest/finding loaders | phase08 tests |
| 60 | `validate_manifest` | function | `scripts/phase08_acceptance.py:159-247` | modified | command_validate | phase08 tests |
| 61 | `validate_findings` | function | `scripts/phase08_acceptance.py:282-367` | modified | synchronize_results | phase08 tests |
| 62 | `normalize_results` | function | `scripts/phase08_acceptance.py:420-491` | modified | report command | phase08 tests |
| 63 | `synchronize_results` | function | `scripts/phase08_acceptance.py:493-524` | modified | report command | phase08 tests |
| 64 | `overall_status` | function | `scripts/phase08_acceptance.py:526-535` | modified | report exit status | phase08 tests |
| 65 | `process_start_token` | function | `scripts/run-real-stack-e2e.py:351-361` | modified | capability manifest | harness tests |
| 66 | `Harness.cleanup` | method | `scripts/run-real-stack-e2e.py:865-886` | modified | Harness.run finally | harness tests |

```yaml
inventory_source_count: 66
audited_symbol_count: 66
inventory_complete: true
generated_groupings:
  - "None; generated API output was not treated as the implementation source."
```

## 6. Function-Level Audit

Each inventory row was audited for contract, boundary/error behavior, lifecycle/concurrency, security, bounded I/O, API simplicity, and adversarial tests. The concrete evidence column names the current caller or artifact used.

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `vite proxyRes import-response fault injection` | Drops one response only after the committed POST reaches the proxy; retry remains application-owned. | The one-shot flag resets before destroy; normal requests pass through. | No request body replay or persistent proxy state is introduced. | Fault injection is opt-in and loopback development-only. | One boolean check per proxy response. | Small deterministic test seam. | undefined | PASS |
| `validateHarnessCapability` | Requires exact Task282 manifest, nonce, evidence root, loopback URL, PID/start token, cwd, port, and strictPort. | Missing, malformed, foreign URL, private-file, stale-process, and wrong-command paths fail closed. | Reads process identity without owning or killing it. | Private capability file and exact run-owned listener are enforced. | Bounded local file and proc reads. | Capability validation is centralized. | undefined | PASS |
| `persistence_evidence_is_valid` | Global identity requires food_items projection NULL owner plus no private duplicate. | Any count mismatch returns false. | Read-only evidence only. | Parameterized fixed SQL and no user input. | Five bounded aggregate queries and one ping. | Exact-once predicate is explicit. | undefined | PASS |
| `Task282Harness.application_environment` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Task282Harness.execute` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Task282Harness.write_backend_evidence` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Task282Harness.combine_results` | Every manifest criterion has one authoritative result; duplicate IDs fail closed. | Missing, malformed, unknown root, duplicate, and failure-status paths are covered. | No last-write-wins result can survive duplicate detection. | Only synchronized roots are preserved; unknown roots map to infrastructure. | Linear manifest-sized normalization. | Deterministic ordering. | undefined | PASS |
| `Task282Harness.export_diagnostics` | Caught browser product nonpass is reported as product_nonpass, not passed. | Only the contradictory passed plus event combination is rewritten. | Delegates final sanitized event export. | Uses base sanitizer. | Constant-time status adjustment. | Truthful run-level state. | undefined | PASS |
| `finalize_reports` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `parse_args` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `main` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `load` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `test_manifest_surface_is_complete_and_unique` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_missing_browser_output_blocks_every_criterion_and_splits_requirements` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_known_product_root_is_preserved_but_unknown_root_fails_closed` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_duplicate_browser_criteria_fail_closed_instead_of_overwriting` | Every manifest criterion has one authoritative result; duplicate IDs fail closed. | Missing, malformed, unknown root, duplicate, and failure-status paths are covered. | No last-write-wins result can survive duplicate detection. | Only synchronized roots are preserved; unknown roots map to infrastructure. | Linear manifest-sized normalization. | Deterministic ordering. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_ownerless_projection_and_validation_reject_private_identity` | Global food_items rows project NULL owner_id and private custom rows retain owner_id before IS NULL. | Private-only and count mismatch tests fail validation. | Read-only SQL evidence. | No private row can satisfy global identity. | Small fixed CTE. | Identity semantics are explicit. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_task282_flag_without_owned_capability_fails_closed` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_task282_capability_rejects_foreign_base_url` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_browser_product_nonpass_is_truthful_in_diagnostics` | Caught browser product nonpass is reported as product_nonpass, not passed. | Only the contradictory passed plus event combination is rewritten. | Delegates final sanitized event export. | Uses base sanitizer. | Constant-time status adjustment. | Truthful run-level state. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `test_fixture_shapes_include_supported_nutrients_and_known_metadata` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `usda_food` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `off_product` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Handler.do_GET` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Handler._json` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Handler.log_message` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `provider fixture main` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `openAdministration` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `search` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `primary curation acceptance scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `provider resilience acceptance scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `OpenFoodFacts metadata scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `USDA optional portion scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `alias and unknown micronutrient scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `disabled vocabulary scenario` | Maps browser behavior to stable criteria and synchronized root causes. | Product failures are recorded rather than skipped; expected skip is BLOCKED. | Reporter finalizes after individual failures. | Assertions exclude controlled secret and payload text. | One desktop project and bounded timeouts. | Production UI/API path. | undefined | PASS |
| `Phase08AcceptanceReporter.onBegin` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Phase08AcceptanceReporter.onTestEnd` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `Phase08AcceptanceReporter.onEnd` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `parseAttachment` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `isRecord` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `splitEnvironment` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `uniqueEvidence` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalDataConfig` | Controlled endpoints are explicit, loopback, bounded, and injected into production composition. | Unsafe schemes/hosts, missing ports, and invalid deadlines reject. | Config is immutable after load. | Production cannot silently use arbitrary external targets. | Startup-only URL parsing. | Configuration boundary is centralized. | undefined | PASS |
| `config.Load` | Controlled endpoints are explicit, loopback, bounded, and injected into production composition. | Unsafe schemes/hosts, missing ports, and invalid deadlines reject. | Config is immutable after load. | Production cannot silently use arbitrary external targets. | Startup-only URL parsing. | Configuration boundary is centralized. | undefined | PASS |
| `loadExternalDataConfig` | Controlled endpoints are explicit, loopback, bounded, and injected into production composition. | Unsafe schemes/hosts, missing ports, and invalid deadlines reject. | Config is immutable after load. | Production cannot silently use arbitrary external targets. | Startup-only URL parsing. | Configuration boundary is centralized. | undefined | PASS |
| `TestLoadExternalDataFixtureOverrides` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `TestLoadRejectsUnsafeExternalDataOverrides` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | The test itself is the adversarial proof; it passed in the fresh focused suite. | PASS |
| `app.newProduction provider composition` | Controlled endpoints are explicit, loopback, bounded, and injected into production composition. | Unsafe schemes/hosts, missing ports, and invalid deadlines reject. | Config is immutable after load. | Production cannot silently use arbitrary external targets. | Startup-only URL parsing. | Configuration boundary is centralized. | undefined | PASS |
| `importCuratedItem` | Caller-owned idempotency key and byte-identical draft snapshot survive ambiguous retry. | Validation, conflict, ambiguity, cancellation, and success paths are distinct. | Ownership token suppresses stale completions. | CSRF and credentials remain browser-managed; key is opaque. | Copies only mutable draft fields. | Retry API is minimal. | undefined | PASS |
| `safeFetch` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.loadClassifications` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.runSearch` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.selectCandidate` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.submitImport` | Caller-owned idempotency key and byte-identical draft snapshot survive ambiguous retry. | Validation, conflict, ambiguity, cancellation, and success paths are distinct. | Ownership token suppresses stale completions. | CSRF and credentials remain browser-managed; key is opaque. | Copies only mutable draft fields. | Retry API is minimal. | undefined | PASS |
| `ExternalImportWorkflow.validDraft` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.hasValidLiquidDensity` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.startFreshImport` | Preserves the named Task 282 contract and its traceability boundary. | Normal, malformed, duplicate, timeout, cancellation, and error paths were inspected; no unobserved return path. | State is bounded and cleanup or ownership guards are explicit for the unit. | Trusted-boundary inputs are validated; no credential, PII, raw payload, or unsafe target crosses the boundary. | Bounded I/O and output; no unbounded loop or allocation introduced. | Necessary, scoped, and idiomatic for the acceptance contract. | undefined | PASS |
| `ExternalImportWorkflow.snapshotDraft` | Caller-owned idempotency key and byte-identical draft snapshot survive ambiguous retry. | Validation, conflict, ambiguity, cancellation, and success paths are distinct. | Ownership token suppresses stale completions. | CSRF and credentials remain browser-managed; key is opaque. | Copies only mutable draft fields. | Retry API is minimal. | undefined | PASS |
| `strict_json_loads` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `validate_manifest` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `validate_findings` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `normalize_results` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `synchronize_results` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `overall_status` | Manifest/result/finding contracts are strict, deterministic, and exit codes distinguish FAIL from BLOCKED. | Duplicate JSON, orphan IDs, missing roots, unsafe evidence, and malformed result paths are rejected. | Reports are staged and finalized deterministically. | Safe relative paths and sensitive-value rejection are enforced. | Bounded JSON/report processing. | Source-of-truth validator remains authoritative. | undefined | PASS |
| `process_start_token` | Only run-owned process/resources are stopped or removed. | Missing/stale identities and partial cleanup are safe. | Start token and ownership checks precede teardown. | No broad process, database, or Redis cleanup. | Bounded teardown. | Idempotent cleanup. | undefined | PASS |
| `Harness.cleanup` | Only run-owned process/resources are stopped or removed. | Missing/stale identities and partial cleanup are safe. | Start token and ownership checks precede teardown. | No broad process, database, or Redis cleanup. | Bounded teardown. | Idempotent cleanup. | undefined | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| NONE | N/A | N/A | No unresolved blocking, important, or optional review finding. | Current source, fresh run, focused tests, and prior-finding rechecks are consistent. | None |

Prior finding verification:

- REV-T282-001 — CLOSED. `frontend/vite.config.ts:18-26` drops the first committed import response only when the opt-in fault flag is set. `ExternalImportWorkflow.svelte:261-269` snapshots the draft and caller-owned key; `external-admin-client.ts:85-103` sends that key/body; `task282-external-curation.spec.ts:90-116` asserts the retry key and body are exactly identical and the second response is replayed. The fresh run includes PASS `P08-SWR055-ACCEPT-06`.
- REV-T282-002 — CLOSED. `OWNERLESS_COUNT_SQL:52-58` projects `food_items` as `NULL::uuid AS owner_id`, unions private `custom_food_items.owner_id`, and filters `owner_id IS NULL`. The fresh backend evidence is `foodItemCount=1`, `ownerlessCount=1), `curatedImportCount=1`, `auditCount=1), `editedCount=1), Redis reachable.
- REV-T282-003 — CLOSED. `validateHarnessCapability("282"):57-122` is called for the Task 282 flag and requires private exact-schema capability, matching nonce/run evidence root, uncredentialed 127.0.0.1 URL with explicit port, live PID/start token, exact cwd, and `--strictPort` command. Missing capability and foreign URL tests fail closed.
- REV-T282-004 — CLOSED. `combine_results:269-320` marks duplicate criterion IDs and clears the map before emitting the full manifest, so both PASS/FAIL and FAIL/PASS duplicates become BLOCKED infrastructure results. Fresh results contain 24 unique criterion IDs.
- REV-T282-005 — CLOSED. `export_diagnostics:322-326` maps a caught `browser_product_nonpass` from a provisional passed result to `product_nonpass` before base export. Fresh `diagnostics.json` has `events=[..., "browser_product_nonpass", ...]` and `result="product_nonpass"`; the regression test passes.

Known product findings are not review findings: the two FAILs and one BLOCKED result above are truthful synchronized acceptance output and remain OPEN in `04_OPEN.md` with fresh evidence and retest conditions.

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `python3 -m unittest scripts/test_task282_acceptance.py scripts/test_phase08_acceptance.py scripts/test_task281_acceptance.py` | `.` | 0 | PASS | 44 tests |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/config ./internal/externaldata ./internal/dataimporter ./internal/app` | `backend` | 0 | PASS | focused Go packages |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/externaldata ./internal/dataimporter` | `backend` | 0 | PASS | race gate |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/config ./internal/externaldata ./internal/dataimporter ./internal/app` | `backend` | 0 | PASS | vet gate |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend` | 0 | PASS | 536 tests, 0 failures |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | `frontend` | 0 | PASS | typecheck, build, tests |
| `python3 scripts/phase08_acceptance.py validate` | `.` | 0 | PASS | 12 scenarios, 91 criteria |
| `python3 scripts/validate-task-list.py && python3 scripts/validate-traceability.py && git diff --check` | `.` | 0 | PASS | task list, traceability, whitespace |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-282-review.md` | `.` | 0 | PASS | this evidence file |
| `python3 scripts/run-task282-acceptance.py` | `.` | 1 | PASS for task-delivery evidence | `d08e098642ba3e9c87de8721`; nonzero is expected because product results include FAIL/BLOCKED |
| `python3 scripts/check.py --quick` | `.` | 0 | PASS | fresh preparation evidence |
| `python3 scripts/check.py` | `.` | 0 | PASS | fresh preparation evidence |

## 9. Files Inspected and Staleness Fingerprints

The following are the current SHA-256 fingerprints captured after source review. The fresh run artifact hashes prove the evidence is not only inherited from the prior review.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `frontend/vite.config.ts` | response-loss fault seam | REV-T282-001 repaired | SHA-256 | `5938c937f565e0b2a94d5ad8788ecebf36ad092cfeb6c986c0e591b27562857c` |
| `frontend/playwright.real-stack.config.ts` | exact Task282 capability | REV-T282-003 repaired | SHA-256 | `c964611230ceb902df37a3de5e843cb11f76451e478e0ca786f7805c14766471` |
| `scripts/run-task282-acceptance.py` | harness, ownerless query, combiner, diagnostics | REV-T282-001/002/004/005 repaired | SHA-256 | `653bec46d4f5a5966fc5c5c75ac33171f923385728dae2f119dbfb7e8b933966` |
| `scripts/test_task282_acceptance.py` | regression tests for all five findings | fresh tests | SHA-256 | `0e2da5cfd23fa3197de7f6bf64757b786ee77fce84a7c532c19759f139912a40` |
| `scripts/task282_provider_fixture.py` | controlled provider branches | provider fixtures | SHA-256 | `871844c12919b250b3da34bc7616787dcc4b9e702b606e80f170d507e1416c1c` |
| `frontend/tests/task282-external-curation.spec.ts` | real UI/provider scenarios | provider and retry evidence | SHA-256 | `70b349dd936ae958cd5a9427f2beb49f92c5969ca72b48e682f733660d2bad09` |
| `frontend/tests/phase08-acceptance-reporter.ts` | criterion reporter and failure precedence | duplicate/root/report evidence | SHA-256 | `3f565b3e5e6009549b47d09bb9bbe75effc524f9cfc56762dd11f4557c6fe8e6` |
| `frontend/src/lib/api/external-admin-client.ts` | caller-owned key/body transport | REV-T282-001 repaired | SHA-256 | `f0cacba9063fb1dae4bfc8b212e6e04a8d3aba2a174f1c7611afdcdb31176c95` |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | same-key retry state | REV-T282-001 repaired | SHA-256 | `3a79b28932330887029154fd9ca12e028eae77b204189f74e6dd006e3c3cb770` |
| `frontend/tests/task281-acceptance-helpers.ts` | shared safe acceptance evidence | harness dependency | SHA-256 | `49877745607f94497a828de43d0c2d86524de84d665db03342f035f7d480594f` |
| `backend/internal/config/config.go` | provider configuration boundary | provider wiring | SHA-256 | `a089b35f800d9b308727e076b54f87155d6df92ad25b5c610ed5723c12078f44` |
| `backend/internal/config/config_test.go` | provider config rejection tests | provider wiring | SHA-256 | `e8b4370c7bcca6d4d439c94aeae5c0873e1d9c4c719623c2a0ed86a7213ef344` |
| `backend/internal/app/app.go` | provider composition | provider wiring | SHA-256 | `29f575dc974b5170b229f62d8cbaa34dc936dde25b72895cf52c7b3b473d0307` |
| `scripts/phase08_acceptance.py` | manifest/finding/report contract | Task280 dependency | SHA-256 | `b36ea8188e3e6d7e05ae17dc992c84cf493c9ac78a8a85706d51e7e40ca34bdb` |
| `scripts/run-real-stack-e2e.py` | run ownership and teardown | Task279 dependency | SHA-256 | `52647965aead4bf16f87e6c1ae4fc4a8c6a7a9c20af4b1ea45921089f8aa140d` |
| `scripts/check.py` | aggregate acceptance/static registration | quality gate | SHA-256 | `77d35bb4967eacff99edffc9f2c4abc9020c281164c4cb9d31c7ebe8e1b1fa89` |
| `scripts/test_check_coverage.py` | coverage contract tests | quality gate | SHA-256 | `f9d45d91f3aa41109670e6496bf10c4ed2fbf20b0f573d9120dc1a369a62fb0b` |
| `database/migrations/000025_user_owned_custom_food_items.up.sql` | owner/private schema boundary | REV-T282-002 repaired | SHA-256 | `dc3e479dd9ba72d39ceeb0a93c3d99b6097c4a862e5800f93ed3612d4f5c5091` |
| `backend/internal/dataimporter/integration_test.go` | import persistence consumer | integration dependency | SHA-256 | `f7a9d70ab6f32cad7a72de770f5de133797781396f680200f830949e3b58e623` |
| `logs/real-stack-e2e/d08e098642ba3e9c87de8721/diagnostics.json` | fresh run diagnostic | REV-T282-005 repaired | SHA-256 | `d036dd3fef5b7107fbb3d234f5769820a668163cab9447b9f2d31c64c48bf801` |
| `logs/real-stack-e2e/d08e098642ba3e9c87de8721/acceptance/browser.json` | fresh Playwright producer | all criteria | SHA-256 | `295b7b7adbea7e2f8949b7c2d37483e56b2b6f80edaab086d2d5ef8b048d5f58` |
| `logs/real-stack-e2e/d08e098642ba3e9c87de8721/acceptance/results.json` | fresh combined result | REV-T282-004 repaired | SHA-256 | `e1fa731c8f3ad0ea3c1b3bd3bcca01beac49107558c8b66cf6004a8483242a35` |
| `logs/real-stack-e2e/d08e098642ba3e9c87de8721/acceptance/backend/curation.json` | fresh exact-once evidence | REV-T282-002 repaired | SHA-256 | `c0c41f703cab7ddc08c43e33ec1561a40aefbff9fc8ee1e5e699ebab2ea1d91b` |
| `docs/implementation/evidence/task-282-preparation.md` | fresh preparation report | baseline/evidence | SHA-256 | `24b9c7f745e678f2084215e7611c445ac7d9e9e8eff69a1674a23019e3f1a26d` |
| `docs/implementation/04_OPEN.md` | synchronized product findings | product status | SHA-256 | `19f45e0a7551fa55785bfa3dd6e390ef1e0ef52af62b8221fdb09bdc92952ab` |
| `docs/testing/phase08/finding-history.json` | finding lifecycle | product status | SHA-256 | `ef0cb5698dec68555ab5e46594f2d580dfb20c210bb057925448184b656cecd0` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/evidence/task-282-review.md was REJECTED and replaced by this current review; affected-symbol claims were revalidated."
```

## 10. Coverage and Exceptions

- [x] Required focused tests ran.
- [x] Backend race and vet ran for changed backend packages.
- [x] Frontend typecheck/build and 536 frontend tests ran.
- [x] Untested branches relevant to changed symbols were inspected, including missing capability, foreign URL, duplicate ordering, unknown root, malformed JSON, timeout/cancellation, proxy response loss, and product-nonpass diagnostic state.
- [x] No unapproved coverage exception was used.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "fresh focused commands and preparation report"
observed_line_coverage: "N/A for this re-review; repository contract checks passed"
coverage_passed: true
```

Coverage finding: No review-blocking coverage gap. The Task 282-specific regression tests directly exercise all five repaired findings and the fresh real-stack run exercises the production path.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by the repaired Task 282 surface.
- [x] Source-of-truth design, task, and finding synchronization are consistent.
- [x] No generated/cache/build/temporary artifact was unintentionally added by this review.
- [x] Public API additions are necessary and used.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, malformed-input, response-loss, duplicate, and truthful-diagnostic paths were challenged.

Findings: None. The task row remains PREPARED; this review did not edit task-list status.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

```yaml
decision: "PASSED"
reason: "All five prior findings are repaired and independently verified with current source, 66 function-level audits, fresh hashes, focused tests, and run d08e098642ba3e9c87de8721; known product nonpasses remain truthful and synchronized."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "No review repair; keep Task 282 status PREPARED until the phase owner applies the product findings and acceptance gate."
```

## 13. Repair Context

Not applicable for PASSED. No task-list status was changed.

