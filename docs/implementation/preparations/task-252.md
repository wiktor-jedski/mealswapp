# Task 252 preparation — Restricted User Administration

## Outcome and scope

- Task: 252, `DESIGN-009: UserAdminPanel`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69` on `multistep-phase-08`.
- Dependencies 98, 113, and 247 were inspected and remain `PASSED`.
- Task 252 remains `OPEN`; this preparation did not edit `docs/implementation/02_TASK_LIST.md` or any status cell.
- Task-list SHA-256 before and after preparation: `689954f8dc9a17c2344db0e03be72e1555aabc9d8756c0d8213763e9ba7c3a96`.
- `docs/implementation/04_OPEN.md` initial SHA-256: `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa`; final SHA-256: `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d`. Concurrent task-239/task-250 coverage notes changed that shared file; task 252 did not edit it and added no exception.
- Scope is restricted to privacy-minimized lookup and an administrator-triggered retry of eligible deletion failures. No role mutation, password access, session impersonation, arbitrary account editing, OpenAPI work, frontend work, or migration was added.

## Baseline and preservation

The worktree was already dirty with unrelated Phase 08 implementation and review work. The initial `git status --short`, branch, commit, task row, dependency rows, and relevant design/deletion/encryption/admin-gateway sources were captured before edits. Shared files were already modified by tasks 238-251 and continued to receive unrelated changes while task 252 was prepared. Those files and hunks were preserved; no reset, checkout, clean, deletion, status transition, or generated-client update was performed.

| Captured baseline path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `689954f8dc9a17c2344db0e03be72e1555aabc9d8756c0d8213763e9ba7c3a96` |
| `docs/implementation/04_OPEN.md` | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |
| `backend/internal/app/app.go` | `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `backend/internal/repository/types.go` | `5534be37a865c95390f84687ed82007e0adbca63a94fbd8c7e849ccb8cc40ac6` |
| `backend/internal/repository/compliance_repository.go` | `d185aed065dd59ade5d3f7330efa5defc1e4acabd5958f2a8ed1e9c83f111f88` |
| `backend/internal/httpapi/admin_controller.go` | `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| `api/openapi.yaml` | `af5a676d54220079d5f852139a57e0737fee7ffa4e3ca595a6ed302417d4d0c7` |

## Prepared behavior

### Privacy-minimized lookup

- `GET /api/v1/admin/users` is registered through the task-247 admin gateway and therefore requires verified JWT-cookie admin authorization and a user-scoped rate limit.
- The query accepts one exact `userId`, one exact normalized email, or one bounded UUID-cursor page. Unknown and duplicate parameters, conflicting selectors, malformed UUIDs, and limits outside 1-25 fail validation.
- Exact email search derives the existing keyed HMAC lookup digest; SQL never searches plaintext email.
- Persistence selects only encrypted email envelope fields, email verification, account creation time, and a bounded deletion summary. It does not select role, password hash/salt, tokens, OAuth identity, profile, sessions, deletion reason, lease, receipt, or unrelated user-owned data.
- AES-GCM email decryption occurs only after the service-level actor check confirms server-derived `admin` role, non-nil identity, and request correlation.
- The API projection is limited to `id`, decrypted `email`, `emailVerified`, `createdAt`, and optional deletion `requestId`, `status`, fixed `failureCategory`, `retryCount`, and `requestedAt`.
- Every successful lookup persists a request-correlated `lookup_users` admin audit with no PII snapshot. Audit failure returns no page to the caller.

### Legal deletion retry

- `POST /api/v1/admin/users/:userId/deletion-requests/:requestId/retry` requires verified admin authorization, CSRF, strict UUID scope, an empty body, a user-scoped rate limit, and the task-247 transactional audit wrapper.
- The locked SQL transition matches both user ID and request ID, requires `failed`, and permits only `permanent`, `unknown`, or `transient` with `retry_count >= 3`.
- A successful action moves the request to `pending`, clears failure detail/category and scheduling metadata, resets its automatic retry budget, and records fixed `failed -> pending` deletion history with note `admin_retry`.
- `FOR UPDATE` plus the status predicate makes concurrent retries claim once. A loser, a cross-user request ID, an ineligible transient failure, a pending/processing/completed request, or a missing request all receive the same not-found repository result.
- The deletion transition, deletion history, and request-correlated admin audit commit atomically. Audit snapshots contain only fixed status/category codes; no deletion reason or other internal value is accepted.
- The response contains only the already-scoped request ID and `pending` status.

### Explicitly absent capabilities

No route, service method, repository query, DTO, or response was added for roles, password material, password reset, authentication tokens, OAuth data, session access, impersonation, arbitrary field edits, or bulk mutation. HTTP tests assert representative undocumented paths remain 404 and that client-controlled mutation fields are rejected.

## Changed paths and symbols

### New production paths

- `backend/internal/repository/admin_user_repository.go`
  - `PostgresAdminUserRepository`, `NewPostgresAdminUserRepository`
  - `LookupAdminUsers`, `RetryAdminDeletion`
  - `validateAdminUserLookup`, `scanAdminUser`
- `backend/internal/repository/sql/admin_user_list.sql`
  - deterministic UUID cursor and SQL-enforced limit
- `backend/internal/repository/sql/admin_user_get_by_id.sql`
  - exact user-ID projection
- `backend/internal/repository/sql/admin_user_get_by_digest.sql`
  - exact keyed-digest projection
- `backend/internal/repository/sql/admin_deletion_retry.sql`
  - scoped locked transition and deletion-history insert
- `backend/internal/useradmin/service.go`
  - `Actor`, `LookupRequest`, `User`, `Deletion`, `Page`, `RetryResult`
  - `Service`, `NewService`, `Lookup`, `RetryDeletion`
  - `lookupRequest`, `authorize`, narrow decrypt/digest/audit dependency interfaces
- `backend/internal/httpapi/user_admin_controller.go`
  - `UserAdminService`, `UserAdminController`, `NewUserAdminController`, `AdminRoutes`
  - `Lookup`, `RetryDeletion`
  - strict lookup/retry validators and safe error mapping

### Shared production paths

- `backend/internal/repository/types.go`
  - added `AdminUserRecord`, `AdminDeletionSummary`, `AdminUserLookup`, `AdminDeletionRetry`, and `AdminUserRepository`.
- `backend/internal/repository/compliance_repository.go`
  - added the fixed `deletion_request` + `retry_deletion` audit snapshot schema and canonical `failureCategory` ordering.
- `backend/internal/app/app.go`
  - composed `useradmin.Service` and merged its two definitions into the existing audited admin route group.

These shared files contain unrelated pre-existing/concurrent task changes. Only the symbols listed above belong to task 252.

### Tests

- `backend/internal/repository/admin_user_repository_test.go`
  - `TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized`
  - `TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures`
  - `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits`
- `backend/internal/useradmin/service_test.go`
  - approved projection, bounded lookahead/cursor, exact email digest, authorized decryption, lookup audit, audit failure, invalid scopes, service-level authorization, retry forwarding, and cross-scope failure
- `backend/internal/httpapi/user_admin_controller_test.go`
  - anonymous/non-admin denial, spoofed input isolation, strict query/body/path validation, approved JSON projection, absent capabilities, CSRF, safe retry response, atomic admin audit metadata, and sanitized cross-scope error

All additions carry adjacent `DESIGN-009 UserAdminPanel` traceability comments. No JSON file was changed, so no JSON sidecar was required.

## Verification evidence

| Command | Result |
|---|---|
| `cd backend && ... go test ./internal/repository -run 'TestPostgresAdminUser|TestPostgresAdminDeletion' -count=1` | Passed; real PostgreSQL lookup, legal-state, cross-scope, concurrent claim-once, deletion-audit, and admin-audit checks. |
| `cd backend && ... go test ./internal/useradmin ./internal/httpapi -run 'TestLookup|TestRetryDeletion|TestUserAdmin' -count=1` | Passed. |
| `cd backend && ... go test -race ./internal/useradmin ./internal/httpapi -run 'TestLookup|TestRetryDeletion|TestUserAdmin' -count=1` | Passed. |
| `cd backend && ... go test -race ./internal/repository -run 'TestPostgresAdminUser|TestPostgresAdminDeletion' -count=1` | Passed. |
| `cd backend && ... go vet ./internal/useradmin ./internal/httpapi ./internal/repository ./internal/app` | Passed. |
| `cd backend && ... go test ./internal/app -run '^$' -count=1` | Passed production composition compile check. |
| `python3 scripts/validate-task-list.py` | Passed: 263 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | Passed. |

A broader combined package run also passed the complete repository package. Its complete app package then failed in the pre-existing task-240 integration test at `task240_custom_item_erasure_integration_test.go:164` (`transactional account cleanup left 2 owner custom items`). That test and its custom-item deletion implementation are outside task 252; no task-240 file was changed here. The isolated app composition check and all task-252 behavior/race checks pass.

## Final task-owned file hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/repository/admin_user_repository.go` | `c16d32d40b459ec2faef972464022bc4f92ab030ade99bafdc3d65f08b732326` |
| `backend/internal/repository/admin_user_repository_test.go` | `8f60600c2f3a0f65e34417820e692e8c4b924fcd4157de32a59b9bec1a95457a` |
| `backend/internal/repository/sql/admin_user_list.sql` | `459f9223f0102d632ec6abdfc9d24b5cd36c600ddfdc3577db305568770c8801` |
| `backend/internal/repository/sql/admin_user_get_by_id.sql` | `99b0bf5403da6e772141b9dbcbde12ab4c6e4b9087a19e325e319a6869a81a22` |
| `backend/internal/repository/sql/admin_user_get_by_digest.sql` | `0125bd5d03e287fbfab9c7fb9bde42a978463d8b9f0d35a60448d8fa838bee14` |
| `backend/internal/repository/sql/admin_deletion_retry.sql` | `5a4c43103305adb37b4aedde810db15b0146162c7555a09f467e6ed9474aea16` |
| `backend/internal/useradmin/service.go` | `ff7c43b484a4ced32e6036cea8676707b8854ff912719d324c0eeb4a3b85e8df` |
| `backend/internal/useradmin/service_test.go` | `de51a99c69009e1ee08ea065f8be1cadaf7953e4cb085a1066246a18360c221f` |
| `backend/internal/httpapi/user_admin_controller.go` | `ffc6606c3599956aa7877a400a9c1f4f9bdf37976e91e31a921e0c7aefbc4e3f` |
| `backend/internal/httpapi/user_admin_controller_test.go` | `2d5b0b79af1478764a9f416f7805f5234e441a9bfad8ffe72c1a90c973c128c9` |

The hashes above were captured before this preparation document was added. Shared-file hashes are intentionally omitted from the final task-owned manifest because concurrent tasks changed those files; their task-252 symbols are recorded explicitly instead.
