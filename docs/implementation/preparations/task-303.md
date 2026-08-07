# Task 303 Preparation Evidence — Mobile Global-Item Discovery Repair

- Task: 303, Phase 08.02 Mobile Global-Item Discovery Repair
- Stage: `implementation/preparation`
- Baseline: `aedbb3b1e4b1d69c76645968f3d7d644bcac4933` (`add remediation tasks for phase findings`)
- Worktree: `/home/wiktor/Work/worktrees/mealswapp/303`
- Branch: `task-303-preparation`
- Task-list status: Task 303 is `PREPARED`; Task 304 remains `OPEN`. Only Task 303's status cell was changed as explicitly authorized.

## Baseline and scope confirmation

`git fetch origin multistep-phase-08 && git pull --ff-only origin multistep-phase-08` completed before implementation and reported `Already up to date.` The baseline task list contains both Tasks 303 and 304. Its SHA-256 was `9c4233acb5f713d56eba8f6cc6494de15f741ab83d865f4f918fba9d7a9c98da`; after commit `4b196165`, the current SHA-256 is `597c8ce527255e1b90d554904f1f6fdcd128ead0fe1c2ef08f505cd0cc88c459`, with only Task 303 changed from `PASSED` to `PREPARED`.

Task 303 depends on Task 283 and is scoped to `P08-FIND-283-001` and `P08-FIND-283-005`. Task 304 and all other task rows were not edited.

## Repair implemented

Autocomplete was reading a plain Redis key even though food mutations already advanced the shared classification generation. A request on another API instance could therefore retain a stale absence after a global item was created, or retain a deleted item until TTL expiry. Autocomplete now uses the same generation-scoped key and guarded `SetIfCurrent` write used by Catalog Search. The added barrier test holds an in-flight old-generation loader across invalidation, proves its stale write is rejected, and proves the next current-generation request loads fresh data. Desktop behavior and the existing global/private repository boundary remain unchanged.

The mobile real-stack scenario warms API-2 autocomplete, Catalog, and the administrator picker before creation, creates the ownerless global source and target through the production API, verifies the mobile administrator picker plus mobile Catalog and Substitution UI, checks API-2 projections and private-item exclusion, deletes the target, and verifies deletion exclusion in every projection and in mobile autocomplete. Explicit substitution searches now preserve the typed query; the default empty-query desktop flow is unchanged, while the mobile evidence can target the newly created item even when the full suite has more than one result page.

The operation evidence includes a read-only PostgreSQL partition proof. It asserts exactly one global row, exactly one owner-owned private fixture row, no `owner_id` column on `food_items`, and a non-null private `owner_id`. The acceptance harness runs Task 294's recovery test as a preserved, separately diagnosed preflight, but its failure or missing exact-effect proof no longer aborts Task 283 execution or backend proof collection. The mobile Task 303 attachment declares its required mobile project, while the manifest-owned allow-list rejects that scope for every other criterion and restores the configured desktop-plus-mobile requirement. Guarded autocomplete now fails closed when generation lookup fails: it returns the source response without writing either a generation-scoped or unscoped key, with a regression test covering a successful write-capable store whose `Current` call fails. The frontend coverage contract was remeasured and reconciled, including the newly measured external-admin row and current uncovered ranges. The finding ledger remains unchanged; this preparation leaves Task 303 `PREPARED` for re-review rather than asserting `PASSED`.

## Changed task surface

| File | Symbols or executable surface | Evidence |
| --- | --- | --- |
| `backend/internal/cache/search_cache.go` | `GetOrLoadAutocompleteResponse` | Shared food/classification generation-aware reads and guarded writes prevent stale mobile discovery after mutation. |
| `backend/internal/cache/search_cache_test.go` | Generation fixture and in-flight invalidation barrier test | A generation change forces a fresh autocomplete load, rejects stale in-flight writes, and stores separate generation-scoped entries. |
| `frontend/src/lib/stores/search.ts`, `frontend/src/lib/stores/search.test.ts` | Explicit Substitution query commit | Mobile-targeted Substitution searches retain the current query; empty-query behavior remains covered. |
| `frontend/tests/task283-manual-catalog.spec.ts` | Mobile Task 303 real-stack scenario and `OperationEvidence.privateItemId` | Production mobile picker, Catalog, Substitution, API-2 cross-instance, private isolation, deletion exclusion, and request-correlated evidence. |
| `frontend/tests/phase08-acceptance-reporter.ts`, `frontend/tests/task281-acceptance-helpers.ts` | Criterion attachment project scope | Mobile-only Task 303 criteria are evaluated against their own mobile run; ordinary criteria retain both-project coverage. |
| `docs/testing/phase08/acceptance-manifest.json`, `docs/testing/phase08/acceptance-manifest.json-trace.md` | `projectScopeAllowList` | Only the ten manifest-mapped mobile criteria may narrow required projects; unauthorized producer scopes fall back to both configured projects. |
| `frontend/src/lib/phase08-acceptance-reporter.test.ts` | Project-scope policy regression tests | Proves an authorized Task 303 scope is accepted and an unallowlisted criterion cannot narrow coverage. |
| `scripts/test_check_coverage.py`, `docs/implementation/04_OPEN.md` | Current frontend and stale Phase 07 coverage contracts | Exact current Bun rows and machine-checked Go declaration rows are recorded so coverage drift is reported rather than hidden. |
| `backend/internal/cache/search_cache.go` | Guarded autocomplete write barrier | A failed generation lookup cannot fall through to an unscoped cache write. |
| `backend/internal/cache/search_cache_test.go` | `TestAutocompleteSkipsWriteWhenGuardedGenerationLookupFails` | A successful loader plus failing generation lookup proves no plain-key write and no unexpected cache key. |
| `scripts/run-task283-acceptance.py` | Item `operation_proof` private partition projection | Parameterized PostgreSQL `READ ONLY` proof for exact global/private counts and ownership. |
| `scripts/run-task283-acceptance.py` | Independent Task 294 preflight/proof diagnostics and Task 283 suite finalization | Task 294 behavior/proof remains checked and diagnosed independently; Task 303 evidence is still collected and combined when that proof is unavailable. |
| `scripts/test_run_task283_acceptance.py` | Partition-proof and harness-isolation safety assertions | Confirms the proof uses the global/private table boundary and never invents `food_items.owner_id`; confirms Task 294 failures do not gate Task 283. |

## Verification

| Command or run | Result |
| --- | --- |
| `python3 -m unittest scripts/test_run_task283_acceptance.py scripts/test_check_coverage.py` | PASS, 53 tests |
| `python3 -m py_compile scripts/run-task283-acceptance.py scripts/test_run_task283_acceptance.py` | PASS |
| `cd backend && ... go test ./internal/cache -run 'Autocomplete|InFlight' -count=1` | PASS |
| `cd backend && ... go test -race ./internal/cache -run 'Autocomplete|InFlight' -count=1` | PASS |
| `cd backend && ... go test ./internal/cache ./internal/search ./internal/httpapi ./internal/itemcurator -count=1` | PASS |
| `cd backend && ... go test -race ./internal/cache -count=1` | PASS |
| `cd backend && ... go vet ./...` | PASS |
| `cd frontend && ... bun run typecheck` | PASS |
| `cd frontend && ... bun test` | PASS, 571 tests / 3,053 expectations |
| `cd frontend && ... bun test src/lib/phase08-acceptance-reporter.test.ts` | PASS, 2 tests / 2 expectations |
| `cd frontend && ... bun run build` | PASS |
| `bunx playwright test tests/task283-manual-catalog.spec.ts --config=playwright.real-stack.config.ts --list` | PASS; 19 tests are registered, including the mobile discovery scenario |
| `python3 scripts/validate-task-list.py` | PASS; 304 sequential tasks; only Task 303 status changed |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/check.py --quick` | PASS; changed-area, static, API, Go, frontend, and acceptance-contract lanes passed |
| `python3 scripts/check.py` | PASS; static, requirements, traceability, task-list, frontend build/unit/coverage, backend, local-stack, frontend verifier, and complete browser lanes passed. The browser lane completed with 349 passed and 83 intentional skips; Phase 08 Go coverage was the exact contracted 4,936/5,335 (92.5%). |
| `python3 scripts/validate-task-list.py` | PASS; 304 sequential tasks; only Task 303 is `PREPARED` and Task 304 remains `OPEN` |
| `git diff --check` | PASS |

The final official `python3 scripts/run-task283-acceptance.py --timeout-seconds 900` run was `7767d35ec910e4ad2a36aa09` and returned exit 2 because five unrelated Task 283 criteria remain BLOCKED. Both browser projects completed. The aggregate was 39 PASS / 5 BLOCKED; the Task 303-mapped criteria `P08-SWR033-STEP-05`, `P08-SWR056-STEP-01`, `P08-SWR056-STEP-02`, and `P08-SWR056-ACCEPT-01` were all PASS.

The final read-only proof is `logs/real-stack-e2e/7767d35ec910e4ad2a36aa09/acceptance/backend/task283-real-stack-mobile-chromium-mobile-global-discovery-proof.json`. It has `assertionFailures: []`, `globalCount=1`, `privateCount=1`, `globalOwnerless=true`, `privateOwned=true`, `mobilePickerContainsGlobal=true`, `mobileCatalogContainsGlobal=true`, `mobileSubstitutionContainsGlobal=true`, private exclusion true on Catalog/Substitution, and false deletion-presence flags across autocomplete, picker, Catalog, and Substitution. The Task 294 preflight still reports its own `toBeFocused()` failure and missing proof, recorded as `task294-diagnostics.txt` and `task294-proof-diagnostics.txt`; Task 283 evidence collection continued and Task 303 rows were not blocked by it.

## Remaining blockers and risks

- The broader Task 283 aggregate still has five unrelated BLOCKED rows: `P08-SWR033-STEP-01` through `P08-SWR033-STEP-04` and `P08-SWR090-STEP-04`. They are outside Task 303's authorized scope.
- The complete `python3 scripts/check.py` gate is green. Its browser lane intentionally skips unrelated capability-gated scenarios; those skips do not gate Task 303.
- Task 294's recovery-focus/proof issue remains its own non-pass and is preserved as diagnostic evidence; it no longer gates Task 303.
- The frontend coverage exception contract is resolved and its static/full coverage lanes pass.
- `P08-FIND-283-001` and `P08-FIND-283-005` remain ledger/Open Points entries for the Task 302 remediation gate; this task does not edit that ledger.
