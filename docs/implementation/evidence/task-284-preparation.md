# Task 284 Repair Evidence

Date: 2026-07-28  
Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`  
Task status: `PREPARED` (intentionally unchanged)  
Review addressed: `docs/implementation/evidence/task-284-review.md`

## Result

Every Task 284 review finding has been repaired without changing
`docs/implementation/02_TASK_LIST.md`. Its SHA-256 remains
`01e858020bc72b7599c4fb0adac5dac4f4037ed33f42a5ab4b9c6672efb48cd9`.
The authoritative managed run remains truthful: 15 criteria pass and the two
criteria covered by open product finding `P08-FIND-284-001` fail. No criterion
is blocked, backend erasure proof has no assertion failures, and the disposable
stack reports `cleaned`.

## Review-finding repairs

### Run-owned proof capability

`scripts/run-task284-acceptance.py` now uses the versioned
`mealswapp.task284-snapshot-request.v2` request. The private mode-0600
capability binds the run ID, nonce, disposable users A/B, selected deleted item,
and immutable expected fixture targets. `validate_snapshot_request` rejects
nonce, user, deleted-item, shape, and filename mismatches.
`collect_snapshot` additionally rejects target rebinding, unknown rows, and
non-parameterized or mutating SQL before collection. Unit coverage proves that
valid UUIDs from another database and later attempts to rebind the original
request are rejected.

### Structural CSV assertions

`frontend/tests/task284-private-erasure.spec.ts` parses downloaded CSV with an
RFC 4180-compatible state machine. It asserts exactly three columns, the exact
header, allowed sections, exact user fields and values, exact custom-item IDs,
and exact saved-diet and saved-diet-entry rows. Recursive owner-leakage checks
also inspect parsed JSON cells. Raw substring matching is not used to establish
CSV contents.

### Saved-diet contract parity

`api/openapi.yaml` requires `savedDiets` in the closed `ExportBundle` and
defines the owner-free `ExportSavedDiet` projection. The API generator,
generated TypeScript, runtime decoder, fixtures, and unit/browser acceptance
assertions all require the field and its exact entries. Backend export maps
repository diets to API-safe DTOs so saved-diet ownership is not serialized.
Generator drift and OpenAPI lint pass.

### Real failure injection and retry

`TestTask240CustomItemErasureIntegration/Task284RealRetryFailureInjectionAndPartialCompletion`
uses real PostgreSQL, Redis, production application wiring, and the account
deletion service. It injects a cache purge failure after transactional database
erasure, observes `failed` state, the same request identity, retry count one,
and retained cache keys, then retries with the real Redis purger and observes
completion without reclaim. The existing repository failure-record outage,
expired-attempt rejection, production-worker, and concurrent-write tests remain
selected. The full selection passes both normally and under `-race`.

### Complete erasure and leakage proof

The managed fixture now gives user A actual rows for OAuth identity, password
reset, profile, sessions, saved items, saved diets and entries, search history,
consent, entitlement, usage, mutation idempotency, and private custom items,
plus a run-owned Redis key. Independent read-only snapshots observe every
surface. Before/after proof for run
`8bcec0b4e5b1ee53576e5c94` records:

- OAuth identities `1 -> 0`
- password-reset tokens `1 -> 0`
- profiles `1 -> 0`
- sessions `2 -> 0`
- saved items `1 -> 0`
- saved diets `1 -> 0`
- saved-diet entries `1 -> 0`
- history `1 -> 0`
- consent `1 -> 0`
- entitlements `1 -> 0`
- usage windows `1 -> 0`
- mutation idempotency rows `3 -> 0`
- active/retained private items `1/2 -> 0/0`
- users `1 -> 0`
- user-A cache keys `1 -> 0`

The completed receipt is pseudonymous. User B's private-row hash/count and
cache count are unchanged, the global-row hash/count is unchanged, and the
observer scans all keys and string values in the run-owned Redis instance for
user-A identifiers and fixture leakage. Proof metadata is observed rather than
hard-coded: read-only transaction `true`, parameterized queries `true`, 268
query observations, run-owned cache target `true`, and a live production worker
PID/start token.

## Truthful product finding

`P08-FIND-284-001` remains an open defect in
`docs/implementation/04_OPEN.md` and remains registered in
`docs/testing/phase08/finding-history.json`. The managed production run
continues to fail only:

- `P08-SWR043-ACCEPT-01`
- `P08-SWR072-STEP-04`

Both use `ROOT-T284-EXPORT-OWNER-PROJECTION`. This preserves the observed
remaining export-owner projection defect; the harness does not convert it into
infrastructure failure or false success.

## Fresh authoritative evidence

Command:

```text
python3 scripts/run-task284-acceptance.py --timeout-seconds 240
```

Run ID: `8bcec0b4e5b1ee53576e5c94`  
Expected exit: `1` for the preserved product finding  
Result: `15 PASS`, `2 FAIL`, `0 BLOCKED`  
Cleanup: `cleaned`

Primary artifacts:

- `logs/real-stack-e2e/8bcec0b4e5b1ee53576e5c94/acceptance/results.json`
- `logs/real-stack-e2e/8bcec0b4e5b1ee53576e5c94/acceptance/backend/task284-proof.json`
- `logs/phase08-acceptance/task284-8bcec0b4e5b1ee53576e5c94-sw-req-043/report.json`
- `logs/phase08-acceptance/task284-8bcec0b4e5b1ee53576e5c94-sw-req-072/report.json`
- `logs/phase08-acceptance/task284-8bcec0b4e5b1ee53576e5c94-sw-req-073/report.json`

## Verification

Focused:

```text
python3 -m unittest -v scripts/test_run_task284_acceptance.py scripts/test_phase08_acceptance.py scripts/test_generate_api_types.py
PASS: 58 tests

cd backend && go test ./internal/app ./internal/userdata ./internal/deletionworker ./internal/cache -count=1
PASS

cd backend && go test -race ./internal/app ./internal/userdata ./internal/deletionworker ./internal/cache -count=1
PASS

cd frontend && bun run test:e2e -- tests/admin-private-data.spec.ts tests/task272-frontend-gate.spec.ts
PASS: 6 tests

cd frontend && bun run check
PASS: 537 tests, 2818 expectations

python3 scripts/generate-api-types.py --check
PASS

npx --no-install redocly lint api/openapi.yaml
PASS: valid, with the existing OAuth 302/no-2xx warning
```

Gates and validators:

```text
python3 scripts/check.py --quick
PASS

python3 scripts/check.py --output logs/task284-repair-full-check.html
PASS

python3 scripts/validate-task-list.py
PASS: 286 sequential tasks

python3 scripts/validate-traceability.py
PASS

python3 scripts/phase08_acceptance.py validate
PASS: 12 scenarios, 91 criteria

python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-284-review.md
PASS

git diff --check
PASS
```

The full gate includes static analysis, OpenAPI and generated-contract drift,
unit tests, frontend coverage, 309 maintained browser tests with 69 managed
real-stack skips, local-stack/UAT verification, vulnerability analysis,
backend aggregate coverage, and the exact Phase 08 coverage contract
`4645/4986` statements (`93.2%`). The report and screenshots are under
`logs/task284-repair-full-check.html` and `logs/screenshots/`.

## SHA-256 inventory

```text
8d5816d02f02f6bb43c5a2a0d8c4e7ea35aa4051c7b6e90f27fea002a74b16b5  api/openapi.yaml
ae55cb9602227cb9d84915fa38a80b34fe93474b2a4edb2e0a2f4454e1f3c8c4  backend/internal/userdata/export.go
769780aa91ad12fcdde2e8e0da6f4b19851bda55727d5cf8b4db8aa035cd1cd8  backend/internal/userdata/export_test.go
75192f9b4a6336a9042a817145b37cfb6c28a46dc3adf6c183a9a6d8089d9203  backend/internal/app/task240_custom_item_erasure_integration_test.go
8bb78f52424c0a311caa70f0b8572459a5c0ae40f0e279981e022cadafba14d9  scripts/run-task284-acceptance.py
24e6446337b14107935c9b394df258acc20f031c7652f3f2015476da44c544d2  scripts/test_run_task284_acceptance.py
fc87002ba7330b3ab050885df4058568a4acb313d91b01dbf335a1d236d08701  scripts/generate-api-types.py
104f50954556ec05a06c646238efba80f9aacd3be9485464f34ac84f6ee150c3  scripts/test_generate_api_types.py
df558b6f29fd994f009557effd1a8ff318498354b9d28ed95851091129566bf0  scripts/test_check_coverage.py
a9d34a70bc39d217eba95923fb3cf5da993d35c38beefe7ba8f8db44148d400c  frontend/src/lib/api/generated.ts
71e6bbcb44d2ca5b8ae264e5fc24a02d5ae2d283edb5e87d41a6b26d70f703bf  frontend/src/lib/api/generated.test.ts
eda6183fe3e1a8541efdf5a33f8218976c42289e6d2923454af976731d55f61c  frontend/src/lib/api/account-data-client.ts
0e7f2063e73dc4d5d2f75cc71996c413af39539749f85dc2a029c992fdf74015  frontend/src/lib/api/account-data-client.test.ts
d5fcf6ca706a2dce8e9bd89254ec43593020eb266f3e8f83cddbc5742a70f0e7  frontend/tests/task284-private-erasure.spec.ts
338a990a5251b6fd41d969149b773842eb1c07414de01d843d58f349ff578bbe  frontend/tests/admin-private-data.spec.ts
7cca0aa7726794a8b7bc3a153fdf1f9f6849e11904bac47586aac3e0318fb9b8  frontend/tests/task272-frontend-gate.spec.ts
10f982da0fa6077b81c6c17bc2d14633cfb5fe0521ce643a63d559c7f673639c  docs/implementation/04_OPEN.md
01e858020bc72b7599c4fb0adac5dac4f4037ed33f42a5ab4b9c6672efb48cd9  docs/implementation/02_TASK_LIST.md
5c12b9150d589f8b3e9cf0dda0cb4aad6021ef898010f8bc46b14263d8a80355  docs/implementation/evidence/task-284-review.md
535d5fe0346e3da8212f14a9fd010ac94f3416db29da8e5d38cc3ad59d0ebc01  logs/real-stack-e2e/8bcec0b4e5b1ee53576e5c94/acceptance/results.json
5745b84a0bd1ca871db2d5caa4b485727fe47e8050e5706fb28778b218db6790  logs/real-stack-e2e/8bcec0b4e5b1ee53576e5c94/acceptance/backend/task284-proof.json
a1c87c0b7e87a6e30a159bb9633d87a4c3ff38a17edc4f1e32d75d341cd88af6  logs/task284-repair-full-check.html
```
