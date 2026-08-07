# Task 304 Preparation Evidence

Date: 2026-08-07
Task: 304 — Classification Validation Scenario Completion
Stage: `implementation/preparation`
Branch: `task-304-classification-validation`
Baseline: `aedbb3b1` (`origin/multistep-phase-08`), including Tasks 303 and 304
Implementation commit: `3a90006d`

The task-list and finding ledger were not modified. This preparation covers only
Task 304 and the missing Task 283 evidence surface for `P08-FIND-283-006`.

## Implemented surface

| File | Surface | Evidence |
| --- | --- | --- |
| `frontend/tests/task283-manual-catalog.spec.ts` | Two-context production API scenarios for canonical and inactive micronutrients, allergens, classifications, exact rollback, cross-instance reads, and Redis snapshots | Isolated real-stack desktop/mobile run |
| `scripts/run-task283-acceptance.py` | Continued Task 283 evidence collection when the independent Task 294 probe is non-pass; read-only persisted relationship and request-audit proofs | Runner unit tests and real-stack backend proof index |
| `backend/internal/repository/task304_classification_validation_integration_test.go` | Real PostgreSQL canonical persistence, invalid-value rollback, audit rollback, and idempotency assertions | `TestTask304CanonicalValuesPersistAndInvalidValuesRollback` |
| `backend/internal/repository/sql/task304_*.sql` | Embedded read-only count queries for item, audit, and idempotency proof | Traceability validator |

## Acceptance results

The fresh isolated real-stack run was `a982b4a4a1b279b5d05b9992`. It used the
managed disposable PostgreSQL/Redis stack, two API instances, and both
`real-stack-desktop-chromium` and `real-stack-mobile-chromium` projects. The
harness diagnostics result was `passed`; the command-level exit `2` reflects
the intentionally grep-filtered Task 283 matrix retaining unrelated criteria as
`BLOCKED`, not a failure in the Task 304 scenarios.

| Criterion | Result |
| --- | --- |
| `P08-SWR090-STEP-01` — canonical Sodium plus active vocabulary key persists | PASS in desktop and mobile; exact micronutrients, classifications, allergens, audit, idempotency, and item generation proof |
| `P08-SWR090-STEP-02` — `Na` rejected | PASS in desktop and mobile; HTTP 400, no item/audit/idempotency row, generation unchanged |
| `P08-SWR090-STEP-03` — unknown key rejected | PASS in desktop and mobile; HTTP 400, no mutation, generation unchanged |
| `P08-SWR090-STEP-04` — inactive vocabulary key rejected | PASS in desktop and mobile; HTTP 400, no mutation, generation unchanged |
| `P08-SWR090-ACCEPT-01` — only active canonical keys persist | PASS in desktop and mobile |
| `P08-SWR056-ACCEPT-03` — invalid allergen and wrong-kind classification rejected | PASS in desktop and mobile; both produce no persisted or audited mutation |
| `P08-SWR057-STEP-01/02` — food category and culinary role production creation | PASS in desktop and mobile; each has one row, one audit action, and one generation increment |

The valid item proof records `manual_create=1`, request-correlated
classification/micronutrient lifecycle audit counts, one idempotency claim,
canonical persisted relationships, and exactly one Redis generation increment.
Vocabulary create/deactivate/reactivate operations leave the item-generation
counter unchanged. The backend integration test additionally proves an audit
snapshot failure rolls back the item, audit entry, and idempotency claim.

## Verification ledger

| Command | Result |
| --- | --- |
| `cd backend && ... go test ./internal/repository -run 'TestTask304CanonicalValuesPersistAndInvalidValuesRollback' -count=1` | PASS against isolated PostgreSQL |
| `cd backend && ... go test ./internal/itemcurator ./internal/httpapi ./internal/micronutrient ./internal/repository -run 'Test(ServiceRejectsInvalidFieldsAndLiquidDensity\|ManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership\|Task304CanonicalValuesPersistAndInvalidValuesRollback\|Micronutrient)' -count=1` | PASS |
| `cd frontend && ... bun run typecheck` | PASS |
| `cd frontend && ... bun run check:api-types` | PASS; generated API types current |
| `python3 -m py_compile scripts/run-task283-acceptance.py` | PASS |
| `python3 -m unittest scripts/test_run_task283_acceptance.py` | PASS; 28 tests |
| `python3 scripts/validate-task-list.py` | PASS; 304 sequential tasks |
| `python3 scripts/validate-traceability.py` | PASS |
| `git diff --check` | PASS |

## Risks and blockers

`P08-FIND-283-006` remains unchanged in the append-only finding ledger as
required by the task scope; this evidence supplies the mapped SW-REQ-090
scenario results for its later synchronization. The full phase-wide aggregate
gate was not rerun because this task required focused checks and the remaining
Task 283 criteria are unrelated to the classification/micronutrient scenario
completion.
