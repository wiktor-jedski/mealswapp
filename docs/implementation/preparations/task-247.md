# Task 247 preparation — latest AdminController review repair

## Outcome and scope

- Task: 247, `DESIGN-009: AdminController`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Dependencies 61, 65, 77, 93, and 242 remain `PASSED`.
- Task 247 remains `PREPARED`; this repair did not edit `docs/implementation/02_TASK_LIST.md` or any status cell.
- Task-list SHA-256 before and after this repair: `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8`.
- Repair scope is limited to the latest findings in `docs/implementation/reviews/task-247-review.md`: semantic route collisions, free-text audit snapshot privacy, and task-local coverage.

## Baseline and preservation

The worktree was already dirty with unrelated Phase 08 implementation and review work. The complete initial `git status --short` was captured before edits. Task-relevant states included modified shared router/repository/task-list/open-items files and untracked task-247 controller, tests, preparation, and review evidence. All unrelated files and hunks were preserved; no reset, checkout, clean, migration, generated-client update, API contract edit, or task transition was performed.

| Repair baseline path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8` |
| `docs/implementation/04_OPEN.md` | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |
| previous `docs/implementation/preparations/task-247.md` | `146d0761d6a824cfd75dde899125413d74266d87728d32f41a4d4dab672bd472` |
| `backend/internal/httpapi/admin_controller.go` | `2340e90fb4b4901697ca0dc3a8acefe69dc55a35eab5fc8db24449666adba01d` |
| `backend/internal/httpapi/admin_controller_test.go` | `c2950c8fa1a999e14cb0e5ea6808dd08913c9429afe55e2127e89d5c3eeb03e6` |
| `backend/internal/httpapi/router.go` | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/repository/compliance_repository.go` | `71011478dfb551bb14bcb130377dbe3e9c2b2d277e25ebf92cf48a3dc2656a93` |
| `backend/internal/repository/admin_audit_security_test.go` | `145f49f7828169efc6f41570d7bdee0c16e4e085e6de02a4340ab5677c09f22e` |
| `backend/internal/repository/postgres_repository_test.go` | `df3673ecc513549eb07b22e36c7563c870446143dd73c18cd47aef5db9377f60` |

## Findings repaired

### Semantic route collisions

`(*AdminController).Routes` now compares every same-method route against previously validated templates with `adminRoutePathsCollide`. Two templates collide when they have equal segment counts and every segment pair can match the same request: equal static segments or either side being a named parameter. This rejects exact duplicates, parameter aliases such as `/:id` versus `/:name`, and static/parameter overlaps such as `/search` versus `/:id`, independent of registration order. Different methods, segment counts, or incompatible static segments remain valid.

Adversarial tests cover both orders of parameter/parameter and static/parameter collisions, exact duplicates, non-overlapping static paths, non-overlapping segment counts, wildcard/optional/malformed templates, empty segments, duplicate parameter names, and allowed literal/identifier boundaries.

### Strict audit snapshot values

The prior generic key allowlist was replaced by explicit entity/action schemas:

- `fixture` + `fixture.update`: boolean `active`/`deleted` and enum `status` (`draft`, `published`).
- `food_item` + `update_food`: enum `status` (`conflict`, `draft`, `imported`, `rejected`).

No free-text key remains. Names, reasons, identifiers, versions, provider text, PII, secrets, unknown actions/entities, nested values, malformed JSON, wrong scalar types, and values outside fixed enums are rejected. The 4096-byte input cap remains. Accepted fields are re-encoded in a fixed order, so duplicate-key smuggling cannot preserve rejected earlier text in the persisted snapshot.

Adversarial coverage places sensitive text under every formerly allowed string key and wrong string types under both retained boolean keys. It also proves unknown schemas fail, unsafe `Before` and `After` snapshots fail, valid schemas pass, transaction rollback remains fail-closed, and canonical output removes a hidden duplicate secret.

### Task-local coverage

No exception was needed or added to `docs/implementation/04_OPEN.md`; its SHA-256 remained `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa`.

The final focused race profile `/tmp/task-247-final.cover` reports 100% line coverage for every task-owned executable function:

- all functions in `admin_controller.go`, including route grammar/collision and transactional response branches;
- `serverRequestID` and the modified `registerV1Routes` admin branch;
- `PersistAuditEntry`, `WithMutationAudit`, `validateAdminAuditEntry`, and `sanitizeAdminAuditSnapshot`.

## Changed paths and symbols

### Production

- `backend/internal/httpapi/admin_controller.go`
  - Modified `(*AdminController).Routes`.
  - Added `adminRoutePathsCollide`.
- `backend/internal/repository/compliance_repository.go`
  - Modified `(*PostgresAdminImportAuditRepository).PersistAuditEntry` and `validateAdminAuditEntry` to pass fixed audit action into snapshot validation.
  - Replaced `adminAuditSnapshotFields` with type `adminAuditSnapshotRule` and `adminAuditSnapshotSchemas`.
  - Modified `sanitizeAdminAuditSnapshot` to enforce explicit schemas and canonical fixed-value output.

### Tests and evidence

- `backend/internal/httpapi/admin_controller_test.go`
  - Modified `TestAdminRouteRegistrationRejectsMissingControls` and `TestAdminMutationResponseWaitsForCommitAndPreservesDomainErrors`.
  - Added `TestAdminRouteRegistrationRejectsSemanticCollisionsInEitherOrder`, `TestAdminRoutePathGrammarBranches`, `assertAdminRoutesPanic`, and `TestGatewayRegistrationRejectsAdminRouteWithoutAuthentication`.
  - Added invalid success-status, no-content-with-data, and direct missing-admin wrapper rollback/error branches.
- `backend/internal/repository/admin_audit_security_test.go`
  - Extended `TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData` with every former free-text key, retained-key type attacks, explicit schemas, unsafe `Before`, unknown action, and duplicate-field canonicalization.
  - Added `TestAdminMutationAuditSuccessfulCommitPath`.
- `backend/internal/repository/postgres_repository_test.go`
  - Replaced free-text food-name audit fixtures with fixed status metadata in `TestPostgresComplianceAndAdminRepositories`.
- `docs/implementation/preparations/task-247.md`
  - Refreshed baseline, findings, symbols, coverage, commands, and hashes.

`backend/internal/httpapi/router.go` and `docs/implementation/04_OPEN.md` were unchanged controls during this repair.

## Verification

All commands ran on 2026-07-21.

| Command | Result |
|---|---|
| `go test -count=1 -race -coverprofile=/tmp/task-247-final.cover ./internal/httpapi ./internal/repository` (`backend/`) | PASS; no race report. HTTP package 88.5%, repository package 93.4%; every task-owned executable function is 100%. |
| `go tool cover -func=/tmp/task-247-final.cover` filtered to task-owned symbols (`backend/`) | PASS; all listed task-247 functions report 100.0%. |
| `go test -count=1 ./...` (`backend/`) | Task-247 packages PASS. Overall command FAILS only in untouched `internal/app.TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`. |
| `go test -count=1 -race ./...` (`backend/`) | Task-247 packages PASS and no race is reported. Overall command has the same untouched task-240 functional failure. |
| `go vet ./...` (`backend/`) | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` (`backend/`) | PASS: no vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; one existing OAuth callback 302-only warning remains. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 ordered tasks; task 247 remains `PREPARED`. |
| `gofmt -d` on scoped Go files | PASS: no diff. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | Traceability, task-list, Go Doc, OpenAPI, script tests, vet, vulnerability scan, focused backend suites, local stack, Phase 02/03 UAT, migrations/repositories/queue/worker/app focused suites, and frontend verification PASS. Aggregate stops at serial `go test ./... -p 1 -count=1` on the same untouched task-240 integration failure. |

Task 240 was not modified because this repair is explicitly scoped to task 247.

## Final hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/httpapi/admin_controller.go` | `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| `backend/internal/httpapi/admin_controller_test.go` | `0718be6b3969ef020da66b3256520e76cc1b8a7aa8a74b624548ed4a08236f3a` |
| unchanged control `backend/internal/httpapi/router.go` | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/repository/compliance_repository.go` | `d185aed065dd59ade5d3f7330efa5defc1e4acabd5958f2a8ed1e9c83f111f88` |
| `backend/internal/repository/admin_audit_security_test.go` | `06aa705afc27beeee0f6de781279d3a92d46cdc36ff9ed9b2773311a00e84d39` |
| `backend/internal/repository/postgres_repository_test.go` | `ceae02b7ee90824286ed621d3f00b756a6ba6101083620a344fb48e74f19d7cd` |
| unchanged control `docs/implementation/02_TASK_LIST.md` | `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8` |
| unchanged control `docs/implementation/04_OPEN.md` | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |

## Handoff

- Both latest review findings are repaired without weakening any prior task-247 fix.
- Task-local executable coverage is 100%; no coverage exception exists or is required.
- Task-list status was not edited; task 247 remains `PREPARED`.
- Full/race/aggregate continue to expose only the untouched task-240 erasure integration failure recorded above.
