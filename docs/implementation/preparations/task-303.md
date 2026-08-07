# Task 303 Preparation Evidence — Mobile Global-Item Discovery Repair

- Task: 303, Phase 08.02 Mobile Global-Item Discovery Repair
- Stage: `implementation/preparation`
- Baseline: `aedbb3b1e4b1d69c76645968f3d7d644bcac4933` (`add remediation tasks for phase findings`)
- Worktree: `/home/wiktor/Work/worktrees/mealswapp/303`
- Branch: `task-303-preparation`
- Task-list status intentionally untouched: Task 303 remains `OPEN`; Task 304 remains `OPEN`.

## Baseline and scope confirmation

`git fetch origin multistep-phase-08 && git pull --ff-only origin multistep-phase-08` completed before implementation and reported `Already up to date.` The baseline task list contains both Tasks 303 and 304. Its SHA-256 is `9c4233acb5f713d56eba8f6cc6494de15f741ab83d865f4f918fba9d7a9c98da`; the current task list has the same hash and has no diff.

Task 303 depends on Task 283 and is scoped to `P08-FIND-283-001` and `P08-FIND-283-005`. Task 304 and the task list were not edited.

## Repair implemented

Autocomplete was reading a plain Redis key even though food mutations already advanced the shared classification generation. A request on another API instance could therefore retain a stale absence after a global item was created, or retain a deleted item until TTL expiry. Autocomplete now uses the same generation-scoped key and guarded `SetIfCurrent` write used by Catalog Search. Desktop behavior and the existing global/private repository boundary remain unchanged.

The mobile real-stack scenario warms API-2 autocomplete, Catalog, and the administrator picker before creation, creates the ownerless global source and target through the production API, verifies the mobile administrator picker plus mobile Catalog and Substitution UI, checks API-2 projections and private-item exclusion, deletes the target, and verifies deletion exclusion in every projection and in mobile autocomplete.

The operation evidence includes a read-only PostgreSQL partition proof. It asserts exactly one global row, exactly one owner-owned private fixture row, no `owner_id` column on `food_items`, and a non-null private `owner_id`. The acceptance reporter still preserves non-pass statuses; no finding ledger or acceptance status was closed by this preparation.

## Changed task surface

| File | Symbols or executable surface | Evidence |
| --- | --- | --- |
| `backend/internal/cache/search_cache.go` | `GetOrLoadAutocompleteResponse` | Shared food/classification generation-aware reads and guarded writes prevent stale mobile discovery after mutation. |
| `backend/internal/cache/search_cache_test.go` | `TestGetOrLoadAutocompleteResponseUsesFoodGenerationForDiscovery`, `generationMemoryStore` | A generation change forces a fresh autocomplete load and stores separate generation-scoped entries. |
| `frontend/tests/task283-manual-catalog.spec.ts` | Mobile Task 303 real-stack scenario and `OperationEvidence.privateItemId` | Production mobile picker, Catalog, Substitution, API-2 cross-instance, private isolation, deletion exclusion, and request-correlated evidence. |
| `scripts/run-task283-acceptance.py` | Item `operation_proof` private partition projection | Parameterized PostgreSQL `READ ONLY` proof for exact global/private counts and ownership. |
| `scripts/test_run_task283_acceptance.py` | Partition-proof safety assertions | Confirms the proof uses the global/private table boundary and never invents `food_items.owner_id`. |

## Verification

| Command or run | Result |
| --- | --- |
| `python3 -m unittest scripts/test_run_task283_acceptance.py` | PASS, 28 tests |
| `python3 -m py_compile scripts/run-task283-acceptance.py scripts/test_run_task283_acceptance.py` | PASS |
| `cd backend && ... go test ./internal/cache ./internal/search ./internal/httpapi ./internal/itemcurator -count=1` | PASS |
| `cd backend && ... go test -race ./internal/cache -count=1` | PASS |
| `cd backend && ... go vet ./...` | PASS |
| `cd frontend && ... bun run typecheck` | PASS |
| `cd frontend && ... bun test` | PASS, 569 tests / 3,044 expectations |
| `bunx playwright test tests/task283-manual-catalog.spec.ts --config=playwright.real-stack.config.ts --list` | PASS; new scenario compiles and is registered |
| `python3 scripts/validate-task-list.py` | PASS; 304 sequential tasks; task list unchanged |
| `python3 scripts/validate-traceability.py` | PASS |
| `git diff --check` | PASS |

The official `python3 scripts/run-task283-acceptance.py --timeout-seconds 900` was run twice. Both runs stopped before Task 303 at the pre-existing Task 294 recovery assertion (`expect(locator).toBeFocused()`), and the harness truthfully emitted infrastructure-blocked criteria; no acceptance result was promoted to PASS.

For direct Task 303 behavior evidence, an in-memory diagnostic harness skipped only the already-failing Task 294 preflight and selected the unchanged mobile scenario. Run `a43a3e011d282f65e3832f33` produced an assertion-clean operation proof at `logs/real-stack-e2e/a43a3e011d282f65e3832f33/acceptance/backend/task283-real-stack-mobile-chromium-mobile-global-discovery-proof.json`: the mobile scenario passed, the read-only proof had `assertionFailures: []`, and all discovery/isolation/deletion observations were true. Because this diagnostic selected one test, its aggregate 44-row acceptance file remains blocked and is not treated as closure evidence for either finding.

## Remaining blockers and risks

- The official Task 283 aggregate cannot currently reach the new scenario because the pre-existing Task 294 production transport test fails its recovery-focus assertion. This is recorded as a blocker, not hidden or repaired under Task 303.
- `P08-FIND-283-001` and `P08-FIND-283-005` remain open in `docs/implementation/04_OPEN.md`; final closure requires the fresh mapped Task 283 acceptance run to pass all criteria.
- No full phase-wide `scripts/check.py --quick` result is claimed here; the changed-area checks above are the preparation gate.
