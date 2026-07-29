# Task 283 repair evidence

task_id: 283
status: PREPARED (task-list status intentionally untouched)
baseline: pre-existing Phase 8 worktree; unrelated and existing Task 283 work preserved
review_repaired: `docs/implementation/evidence/task-283-review.md`
final_managed_run: `6347dedd9d831a9da23a07e4`

## Repair summary

The five original review findings, both findings from the earlier 2026-07-28
independent re-review, and the remaining focused-regression finding were
addressed in the acceptance implementation, harness, tests, and finding ledger:

1. The Playwright scenarios now assert returned stable identities and exact Catalog, Substitution, and autocomplete result membership; separate solid/liquid density rules; invalid-input rollback; idempotent replay; deletion; metric/imperial calculations and display bases; classification lifecycle behavior; and canonical Sodium persistence.
2. API-2 is called directly for cross-instance reads, committed updates, and audit-failure injection. The stale API-1 write and failed API-2 mutation are measured instead of inferred from two pages targeting one API.
3. The runner emits 34 operation-scoped proof files. Each proof is tied to returned entity IDs, idempotency keys, and request IDs, and compares exact database mutation, audit, idempotency, rollback, canonical-value, deletion, and Redis-generation state. The former aggregate proof is not used.
4. Setup, spawn, timeout, reporter, signal, and teardown failures finalize synchronized `browser.json`, `results.json`, and six requirement shards with all 44 criteria represented. Signal handling is connected to the shared harness controller.
5. The full-check wrapper's parent writes were diagnosed as concurrent `TextIOWrapper` writes reaching a nonblocking output descriptor. Parent output is now serialized and written completely with retry-on-`EAGAIN`; focused partial-write and redirected-stream regressions pass. The substantive full gate then completed with exit 0.
6. Redis generation values are no longer copied from browser expectations into backend actuals. A harness-owned observer answers each browser handshake with a separate query against the run-owned Redis container and persists 38 immutable operation snapshots. Backend proofs load those snapshots by validated UUID, assert exact values and deltas, prove `+1` committed classification invalidation, and prove zero-delta invalid-input, conflict, and audit rollback paths. Missing, altered, malformed, incomplete, and duplicate-identity snapshots are now covered by focused regressions: invalid evidence fails closed, while altered values remain visible as exact evidence mismatches.
7. Historical findings `P08-FIND-283-003` and `P08-FIND-283-004` are now `CLOSED` with `closedDate`, `passingEvidence`, and retest metadata tied to the fresh all-PASS SW-REQ-019 and SW-REQ-032 evidence. The genuine SW-REQ-033, SW-REQ-056, SW-REQ-057, and SW-REQ-090 failures and blockers remain open and visible.
8. Table-driven snapshot-identity regressions now exercise omitted and extra mappings, duplicate UUIDs, malformed UUIDs, invalid JSON, non-object observations, and invalid schema, ID, key, and value fields. Each rejection is asserted before any actual generation evidence can be manufactured, and the expected browser evidence remains unchanged.

The source still names exactly the Task 283 manifest's 44 unique criterion IDs and uses only the six requirement-specific roots.

## Function-level evidence

| File | Repaired surface | Evidence |
|---|---|---|
| `frontend/tests/task283-manual-catalog.spec.ts` | Production scenarios and operation attachments | Exact create/update/delete bodies and IDs; include/exclude stable-ID filters; Catalog/Substitution/autocomplete result sets; density/provenance; independent invalid cases; idempotent replay; classification duplicate/cycle/in-use/detach/delete; API-2 stale-write and audit-failure paths; metric/imperial equivalence; canonical/alias/unknown micronutrients; browser/direct and harness-independent Redis observation handshakes. |
| `frontend/tests/phase08-acceptance-reporter.ts` | `onTestEnd` attachment collection | Merges every operation attachment produced by one test instead of retaining only one attachment, preserving operation-level evidence. |
| `scripts/run-task283-acceptance.py` | `Task283Harness`, `observe_redis_generation_requests`, `redis_generation_actual`, operation proof generation/comparison, failure finalization, `main` | Runs API-1/API-2, independently queries Redis at each operation boundary, persists immutable snapshots, proves exact generation values/deltas plus parameterized read-only SQL state, maps semantic mismatches to exact criteria, and always writes the synchronized 44-row result set on infrastructure failure. |
| `scripts/test_run_task283_acceptance.py` | Focused harness regressions | Covers exact criteria, API-2 consumption, operation-scoped proof semantics, no aggregate proof, independent generation equality/change comparisons, altered and missing snapshots, incomplete mappings, duplicate identities, malformed UUID/schema/key/ID/value/JSON observations, truthful mismatch behavior, malformed/missing browser output, timeout/setup/spawn/signal finalization, and six-shard 44-row synchronization. |
| `docs/implementation/04_OPEN.md` | Phase 08 finding ledger | Closes only SW-REQ-019 and SW-REQ-032 historical blockers with fresh PASS evidence and required closure metadata; preserves genuine non-pass records. |
| `scripts/check.py` | `_write_text`, `safe_print` | Serializes parent-process gate output and retries partial/nonblocking writes without dropping bytes. Existing gate lanes and unrelated work remain intact. |
| `scripts/test_check_coverage.py` | Check-wrapper output regressions | Reproduces `EAGAIN`, verifies complete partial-write recovery, and verifies streams without a file descriptor. |

## Final managed acceptance evidence

`python3 scripts/run-task283-acceptance.py --timeout-seconds 900` completed the managed desktop/mobile run and returned exit 1 because the preserved product findings are real FAIL results, not because the harness failed.

The browser producer completed both projects. The runner generated 34 operation
records, 34 operation-scoped backend proofs, 38 immutable harness-owned Redis
observations, one proof index, one combined 44-row result, and six Task 280
reports. The stable-ID proof independently records generation `6 -> 7`
(`generationDelta=1`), while the audit rollback proof records `14 -> 14`
(`generationDelta=0`).

| Requirement | Distribution | Report status / exit |
|---|---|---|
| SW-REQ-019 | 4 PASS | PASS / 0 |
| SW-REQ-032 | 3 PASS | PASS / 0 |
| SW-REQ-033 | 1 PASS, 4 BLOCKED | BLOCKED / 2 |
| SW-REQ-056 | 10 PASS, 3 FAIL | FAIL / 1 |
| SW-REQ-057 | 12 PASS, 2 FAIL | FAIL / 1 |
| SW-REQ-090 | 4 PASS, 1 BLOCKED | BLOCKED / 2 |
| Total | 34 PASS, 5 FAIL, 5 BLOCKED | truthful non-pass |

Remaining truthful findings:

- `P08-SWR056-STEP-05`, `P08-SWR056-STEP-06`, and `P08-SWR056-ACCEPT-06` are FAIL: both desktop and mobile prove the row is deleted and absent from active Catalog/Substitution reads, but autocomplete still returns the deleted stable ID (`autocompleteContainsDeleted=true`).
- `P08-SWR057-STEP-08` and `P08-SWR057-ACCEPT-05` are FAIL: API-2 commits the fresh update, but API-1 accepts the stale absolute write with HTTP 200 instead of 409 and records the extra update audit.
- `P08-SWR033-STEP-01` through `P08-SWR033-STEP-04` remain BLOCKED by unavailable provider-import behavior in this Task 283 scenario.
- `P08-SWR090-STEP-04` remains BLOCKED because the product has no supported vocabulary disable/restore capability.

All other review-targeted behavior, including failed-mutation non-invalidation, exact rollback, density, stable-ID filtering, canonical persistence, and metric/imperial calculations, is PASS with operation-scoped proof.

## Validation

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_run_task283_acceptance.py` | PASS, 28 focused Task 283 tests |
| `python3 -m unittest scripts/test_phase08_acceptance.py scripts/test_task281_acceptance.py scripts/test_task282_acceptance.py scripts/test_run_task283_acceptance.py scripts/test_check_coverage.py` | PASS, 95 tests |
| `cd frontend && bun run typecheck` | PASS |
| `python3 scripts/phase08_acceptance.py validate` | PASS, 12 scenarios / 91 criteria |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks; Task 283 row not edited |
| `python3 scripts/validate-traceability.py` | PASS |
| `git diff --check` | PASS |
| Fresh `python3 scripts/check.py --quick` after the focused snapshot repair | PASS, exit 0; changed backend, frontend, static, and browser areas completed; Task 283 task-list row remained untouched |
| `python3 scripts/run-task283-acceptance.py --timeout-seconds 900` | Truthful product non-pass, exit 1; managed run `6347dedd9d831a9da23a07e4`, both browser projects complete, 34 PASS / 5 FAIL / 5 BLOCKED after independent backend proof |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-283-review.md` | PASS; review evidence structurally valid |
| First `python3 scripts/check.py` after the Redis observer repair | PASS, exit 0; substantive static, frontend build/unit/coverage, browser, local-stack, backend integration/race/coverage/vulnerability lanes completed; browser 309 passed / 63 skipped |
| Two final `python3 scripts/check.py` reruns after snapshot-identity hardening | FAIL only at the aggregate backend race command after the static, frontend, local-stack, integration, vulnerability, and 309-pass/63-skip browser work completed; both failures were preserved |
| `cd backend && MEALSWAPP_REDIS_URL=redis://localhost:6379/11 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -p 1 -count=1` | PASS on the immediate exact-command diagnostic rerun; the aggregate-only race-command failure remains an intermittent gate issue and is not hidden |
| Fresh `python3 scripts/check.py` after the focused snapshot repair | PASS, exit 0; all configured lanes completed, including static analysis, frontend build/unit/coverage, local-stack and backend integration/race/coverage/vulnerability checks; browser 309 passed / 63 skipped |

## Fresh SHA-256 hashes

Repair sources:

- `frontend/tests/task283-manual-catalog.spec.ts`: `7fcd9081b6e86ce6d61fea53f2fdbb7abdaa5563f1348de91a0b32c85a3c6c7d`
- `frontend/tests/phase08-acceptance-reporter.ts`: `6dcc854de96195876777fb34dde079d034cc42f96f45e789db9cae135cc41af9`
- `scripts/run-task283-acceptance.py`: `f03b5ae5f4876baaeb044540cc15dbe60a044e3af6d9cc1561e6f9117ae28989`
- `scripts/test_run_task283_acceptance.py`: `f106d17cd09672c6fb3775825afca000e6a7cd680deb3a7e27bbeaf8b16020dd`
- `scripts/check.py`: `ff99fd7854fd88b1413d9fa96a3c14ae3eeaf55a2fea423e216ec19dd894c674`
- `scripts/test_check_coverage.py`: `292200dd2cd25756601be5410e506d051f89c197436522dfd22baefceadfaf04`
- `docs/implementation/04_OPEN.md`: `90ca2b98a7079b02c9b562f892e3250a425a54e55f4d629c3b680b6444d744db`
- `docs/implementation/evidence/task-283-preparation.md`: self-referential evidence file; intentionally not hashed

Managed-run core:

- `logs/real-stack-e2e/6347dedd9d831a9da23a07e4/acceptance/results.json`: `8dc02748053ab555cd8bb3c13400d7657bd42bcff8f69e77b6bdae3fa40e8de1`
- `logs/real-stack-e2e/6347dedd9d831a9da23a07e4/acceptance/browser.json`: `69f4268e39bf8fe0c265de262b188b408f23445c25b98eaf44adf6b2b721b8ec`
- `logs/real-stack-e2e/6347dedd9d831a9da23a07e4/acceptance/backend/task283-proof-index.json`: `4163ae5eb9430a1ee7e01e7a28f719086a6fba6b4c09e96844defc9dd749c0d6`
- stable-ID filter proof: `729a9265e13c874ec685764b78739a7ee402e08199ba15bb16be017f33d5546d`
- audit rollback proof: `e99e93a661d54d39c2bf68e22721576cdf54e13a8f6527f1f173f0fe5482bd04`
- classification failure non-invalidation proof: `fb1f37a1a98644d4fd131494bf5405eadac2234942b6cb38e025a38e6bda6617`

Task 280 report JSON:

- SW-REQ-019: `fc5828382aaa2bd6271c8f0f12a17b1da09e7271e280890ac3a7afbae3c66d00`
- SW-REQ-032: `2616737a5b93c81b0f54e0f15f040f23548d327cd20799e4e5fc7f377ec20fe0`
- SW-REQ-033: `bcbadcc27d9f6e439701ab95679ce1a52485951896f3ad8101cd802f7ad97a9c`
- SW-REQ-056: `257e978a0e1cb2893922f1696797ee583248fbb74b3236a3167e3a39fcb5b753`
- SW-REQ-057: `b679383923c3a98854c144f81c337b2d83cfd9bc82e8c4372e4a4435c3fdb631`
- SW-REQ-090: `63e34dc9da2c17c350d31f92ef93b449126ad2fb93d06e5de29ae4bff2f65a25`
