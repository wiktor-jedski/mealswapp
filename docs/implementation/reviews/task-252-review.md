# Review Evidence: Task 252 — Restricted User Administration

~~~~yaml
task_id: 252
component: "Restricted User Administration"
static_aspect: "DESIGN-009 privacy-minimized user lookup, authorized PII decryption, bounded pagination, and scoped legal deletion retry"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T14:10:51Z"
review_agent: "Codex independent reviewer"
evidence_file: "docs/implementation/reviews/task-252-review.md"
baseline_ref: "HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69 plus current task-252 worktree files"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Go + PostgreSQL/SQL + HTTP/auth/CSRF + privacy/security + transactions/concurrency + error handling"
repair_context_required: false
~~~~

## 1. Task Source

**Description:** Add privacy-minimized administrative user lookup and the documented admin-triggered retry action for permanent, unknown, or exhausted account-deletion failures. Do not add role mutation, password access, session impersonation, or arbitrary account editing.

**Depends On:** 98 PASSED; 113 PASSED; 247 PASSED.

**Testing Coverage Exceptions:** The task row says `None`. The package profiles and task-local function results are reported exactly below; no production or task-list change was made to manufacture coverage.

**Verification Criteria:** Exact or bounded lookup returns only the approved projection; encrypted PII is decrypted only at the authorized service boundary; enumeration and pagination are bounded; retry permits only legal locked transitions; concurrent retries claim once; every action is audited; non-admin and cross-scope calls fail closed; and responses/logs do not expose password material, tokens, deletion internals, or unrelated user data.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PASSED and the preparation report was read.
- [x] The task-owned files and current shared-symbol fingerprints were checked against the preparation manifest.
- [x] The review template was read fully.
- [x] `code-review-skill` was invoked exactly once; the Go, security, concurrency, SQL, error-handling, and review-checklist guidance was read.
- [x] The fixed baseline, current callers, tests, and design/requirement sources were independently inspected.
- [x] Relevant focused tests, race tests, vet, vulnerability, OpenAPI, traceability, aggregate, coverage, and evidence validation commands were run.
- [x] No production code or task-list file was edited by this review.

~~~~yaml
pre_review_gates_passed: true
blocking_issue: "None. The aggregate local-stack failure is an external dirty-environment migration-state failure and is recorded in section 8."
~~~~

## 3. Review Baseline and Change Surface

The review used branch `multistep-phase-08` at `81ca40ce00cb667ea29243ed2d34068e11229a69` (`phase 08 planned`) plus the current task-252 worktree files. The worktree contains unrelated Phase 08 changes, including a concurrently modified task list and other implementation/test files. Task 252 remains `PREPARED`; its status and the task-list file were not changed.

The preparation report [`docs/implementation/preparations/task-252.md`](../preparations/task-252.md) was checked for scope, dependencies, file fingerprints, claimed tests, and stale-state warnings. No previous task-252 review evidence existed. Shared files were attributed only at the task-252 symbols and direct composition/caller paths.

Reviewed task-owned implementation and test paths:

| Changed file | Task-owned surface | Review scope |
|---|---|---|
| `backend/internal/repository/admin_user_repository.go` | Restricted SQL repository, validation, scan/deletion retry boundary | Full file |
| `backend/internal/repository/admin_user_repository_test.go` | PostgreSQL projection, scope, legal-state, concurrency, and audit tests | Full file |
| `backend/internal/repository/sql/admin_user_list.sql` | Bounded cursor page and approved encrypted projection | Full file |
| `backend/internal/repository/sql/admin_user_get_by_id.sql` | Exact ID projection | Full file |
| `backend/internal/repository/sql/admin_user_get_by_digest.sql` | Exact normalized-email digest projection | Full file |
| `backend/internal/repository/sql/admin_deletion_retry.sql` | Scoped atomic eligibility, claim, reset, and deletion audit | Full file |
| `backend/internal/useradmin/service.go` | Authorization, normalization/digesting, decryption, pagination, safe response mapping, audit | Full file |
| `backend/internal/useradmin/service_test.go` | Service boundary, privacy, pagination, audit, and authorization tests | Full file |
| `backend/internal/httpapi/user_admin_controller.go` | Admin route definitions, typed input validation, safe response/error mapping | Full file |
| `backend/internal/httpapi/user_admin_controller_test.go` | Auth, CSRF, spoofing, bounds, privacy, retry, and audit HTTP tests | Full file |

Relevant shared implementation paths were inspected at the task symbols and direct callers: `repository/types.go`, `repository/compliance_repository.go`, `app.NewProduction`, the admin gateway, `router.go`, `auth_middleware.go`, `security/encryption.go`, `security/lookup_digest.go`, and the deletion migrations. Design and requirement sources inspected included `DESIGN-009`, `ARCH-009`, `DESIGN-005`, `DESIGN-013`, `ARCH-005`, `ARCH-006`, `ARCH-013`, `SW-REQ-054`, `SW-REQ-067`, `SW-REQ-073`, `SW-REQ-075`, and `SW-REQ-084`.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Current evidence |
|---:|---|---|---|
| 1 | Exact ID and normalized-email lookup return only the approved user/deletion projection. | PASS | Three parameterized SQL projections select account ID, encrypted email envelope, verification flag, creation time, and bounded latest deletion summary only; repository and HTTP tests assert forbidden fields are absent. |
| 2 | Bounded enumeration and pagination are enforced. | PASS | Service limit is 1–25, repository accepts at most 26 only for private lookahead, cursor uses ordered `id > cursor`, and the lookahead row is removed before response. |
| 3 | PII is decrypted only at the authorized service boundary. | PASS | Repository returns encrypted envelope fields; only `useradmin.Service.Lookup` calls the injected decrypter after exact signed-admin authorization; unauthorized and decryption-error tests fail closed. |
| 4 | Email lookup uses the active keyed digest and does not query plaintext email. | PASS | Service normalizes input and calls `DigestForWrite`; SQL matches versioned digest columns; the exact-email test verifies the normalized digest request. |
| 5 | Retry permits only permanent, unknown, or exhausted transient failures. | PASS | SQL locks the exact `(request_id,user_id)` row and requires `failed` plus the legal failure category condition; PostgreSQL tests cover legal and ineligible categories/counts. |
| 6 | Retry state changes and audit are atomic. | PASS | `FOR UPDATE` plus one transaction updates `failed → pending`, clears retry internals, inserts the deletion audit entry, and returns only fixed retry metadata. |
| 7 | Concurrent retry claims occur at most once. | PASS | Two concurrent transaction calls are tested; one succeeds and one returns not-found, with exactly one deletion-audit row and one admin audit. |
| 8 | Every lookup/retry action is audited at the documented boundary. | PASS | Lookup persists a fixed `lookup_users` admin audit before response release; retry is inside the existing transactional mutation/audit gateway and uses a strict `retry_deletion` schema. |
| 9 | Anonymous/non-admin/cross-scope requests fail closed. | PASS | Auth middleware supplies signed server identity/role; service requires exact admin role and request ID; HTTP tests cover anonymous, non-admin, missing capability, spoofed identity, and cross-user/request scope. |
| 10 | No roles/passwords/password resets/tokens/session impersonation/arbitrary edits are exposed or accepted. | PASS | DTOs, SQL projections, path/query/body validators, and safe response assertions exclude role mutation, password material, reset/session tokens, impersonation inputs, and general account-edit fields. |
| 11 | Auth, CSRF, rate limits, logs, and errors remain safe. | PASS | Admin gateway ordering is auth → role → CSRF → validation → rate limit → audit → handler; server request IDs replace client values; instrumentation logs route/status/server identity metadata without raw query/body; error mapping is generic. |
| 12 | Scope and design traceability are complete. | PASS | Design comments identify `DESIGN-009`; production wiring has one explicit service/controller path; direct callers and all task tests were audited; focused tests, race, vet, security, OpenAPI, traceability, aggregate, coverage, and evidence checks are recorded below. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Result | Main callers/consumers | Tests/evidence |
|---:|---|---|---|---|---|---|
| 1 | `adminUserListSQL`, `adminUserGetByIDSQL`, `adminUserGetByDigestSQL` | embedded SQL | `repository/admin_user_repository.go:15-29` | PASS | Repository lookup selector | PostgreSQL projection/scope tests |
| 2 | `adminDeletionRetrySQL` | embedded SQL | `repository/admin_user_repository.go:30-32` | PASS | `RetryAdminDeletion` | Legal/concurrent retry tests |
| 3 | `PostgresAdminUserRepository` and compile assertion | repository type | `repository/admin_user_repository.go:34-40` | PASS | Production composition and interface boundary | Constructor and repository tests |
| 4 | `NewPostgresAdminUserRepository` | constructor | `repository/admin_user_repository.go:43-47` | PASS | `app.NewProduction` | Coverage profile |
| 5 | `LookupAdminUsers` | repository lookup | `repository/admin_user_repository.go:49-78` | PASS | `useradmin.Service.Lookup` | Exact/page/error and SQL tests |
| 6 | `RetryAdminDeletion` | repository mutation | `repository/admin_user_repository.go:80-90` | PASS | `useradmin.Service.RetryDeletion` | Scope, legal, concurrent tests |
| 7 | `validateAdminUserLookup` | repository validator | `repository/admin_user_repository.go:92-117` | PASS | `LookupAdminUsers` | Limit/selector/conflict tests |
| 8 | `scanAdminUser` | encrypted projection mapper | `repository/admin_user_repository.go:119-136` | PASS | `LookupAdminUsers` | Projection and database scan coverage |
| 9 | `AdminUserRecord`, `AdminDeletionSummary` | persistence DTOs | `repository/types.go:604-622` | PASS | Repository → service boundary | Projection assertions |
| 10 | `AdminUserLookup`, `AdminDeletionRetry` | boundary DTOs | `repository/types.go:624-641` | PASS | Service/repository and audit boundaries | Selector/retry tests |
| 11 | `AdminUserRepository` | restricted interface | `repository/types.go:910-915` | PASS | Service dependency injection | Compile assertion and fakes |
| 12 | `ErrForbidden`, `DefaultPageSize`, `MaxPageSize` | service policy | `useradmin/service.go:16-24` | PASS | `authorize`, `lookupRequest` | Unauthorized/bounds tests |
| 13 | `Actor`, `LookupRequest` | service input | `useradmin/service.go:26-41` | PASS | HTTP controller → service | Spoof and selector tests |
| 14 | `User`, `Deletion`, `Page`, `RetryResult` | public response DTOs | `useradmin/service.go:43-74` | PASS | HTTP JSON encoder and audit | Forbidden-field assertions |
| 15 | `piiDecrypter`, `lookupDigester`, `lookupAuditor` | narrow security interfaces | `useradmin/service.go:76-93` | PASS | Authorized service boundary | Unauthorized decryption/audit tests |
| 16 | `Service` and `NewService` | service composition | `useradmin/service.go:95-109` | PASS | `app.NewProduction` | Constructor and package coverage |
| 17 | `Service.Lookup` | lookup workflow | `useradmin/service.go:111-156` | PASS | HTTP `UserAdminController.Lookup` | Projection, pagination, audit, error tests |
| 18 | `Service.RetryDeletion` | retry workflow | `useradmin/service.go:160-172` | PASS | HTTP retry mutation | Authorization/scope/claim tests |
| 19 | `lookupRequest` | selector normalization | `useradmin/service.go:174-204` | PASS | `Service.Lookup` | Email digest, limit/conflict tests |
| 20 | `authorize` | service authorization | `useradmin/service.go:206-213` | PASS | Both service operations | Non-admin/missing identity tests |
| 21 | `UserAdminService`, `UserAdminController` | HTTP boundary types | `httpapi/user_admin_controller.go:17-28` | PASS | Admin gateway/controller | Constructor and route tests |
| 22 | `NewUserAdminController` and `AdminRoutes` | route composition | `httpapi/user_admin_controller.go:30-45` | PASS | `AdminController`/`registerV1Routes` | Route policy and endpoint tests |
| 23 | HTTP `Lookup` | safe lookup handler | `httpapi/user_admin_controller.go:47-68` | PASS | GET `/admin/users` | Auth, projection, bounds tests |
| 24 | HTTP `RetryDeletion` | scoped mutation handler | `httpapi/user_admin_controller.go:70-98` | PASS | POST retry route/transaction gateway | CSRF, scope, response/audit tests |
| 25 | `validateAdminUserLookup` (HTTP) | query validator | `httpapi/user_admin_controller.go:100-143` | PASS | Route middleware | Unknown/duplicate/conflict/bounds tests |
| 26 | `validateAdminDeletionRetry` | body/path validator | `httpapi/user_admin_controller.go:145-154` | PASS | Retry route middleware | UUID and mutation-field tests |
| 27 | `adminUserError`, validation/dependency errors | safe error mapping | `httpapi/user_admin_controller.go:156-183` | PASS | Both handlers/global error writer | Safe status and non-disclosure tests |
| 28 | `adminAuditSnapshotSchemas` retry rule | audit allowlist | `repository/compliance_repository.go:562-593` | PASS | `PersistAuditEntry`/mutation gateway | Fixed status/category audit tests |
| 29 | `PersistAuditEntry`, `WithMutationAudit`, sanitizer path | audit boundary | `repository/compliance_repository.go:354-419,534-678` | PASS | Retry HTTP gateway and lookup audit | Atomic audit/privacy tests |
| 30 | `NewProduction` admin wiring | application composition | `app/app.go:89-95,172` | PASS | Runtime service/controller/gateway | App construction and route wiring inspection |
| 31 | `RequireAdmin`, `requireAuth`, `authenticatedUser` | auth caller boundary | `httpapi/admin_controller.go`, `httpapi/auth_middleware.go` | PASS | Admin route middleware and handlers | Anonymous/non-admin/spoof tests |
| 32 | `registerV1Routes`, `serverRequestID`, `instrument` | gateway/log boundary | `httpapi/router.go:168-223,405-450` | PASS | All versioned routes | Ordering/request-ID/log assertions |
| 33 | `EncryptionService.DecryptPII` | encryption primitive | `security/encryption.go:58-72` | PASS | Injected only into useradmin service | Decryption boundary inspection/tests |
| 34 | `LookupDigestService.DigestForWrite` | lookup privacy primitive | `security/lookup_digest.go:37-51` | PASS | `lookupRequest` | Normalization/digest test |
| 35 | Deletion schema/state constraints | database contract | `database/migrations/000009_consent_deletion.up.sql`, `000014_deletion_request_hardening.up.sql` | PASS | Retry SQL and existing deletion worker | Legal-state and integration tests |
| 36 | `TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized` | repository test | `repository/admin_user_repository_test.go:17-56` | PASS | Lookup SQL/repository | Exact/page/projection coverage |
| 37 | `TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures` | repository test | `repository/admin_user_repository_test.go:58-117` | PASS | Retry eligibility SQL | Legal state/category/count/scope coverage |
| 38 | `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits` | repository test | `repository/admin_user_repository_test.go:119-183` | PASS | `FOR UPDATE`/audit transaction | Concurrent single-claim coverage |
| 39 | `TestLookupProjectsOnlyApprovedFieldsWithBoundedPaginationAndAudit` | service test | `useradmin/service_test.go:84-120` | PASS | Lookup mapping/decryption/audit | Projection/lookahead coverage |
| 40 | `TestLookupExactEmailNormalizesDigestAndAuditsEntity` | service test | `useradmin/service_test.go:122-139` | PASS | Email selector path | Digest and entity audit coverage |
| 41 | `TestLookupFailsClosedBeforeUnauthorizedDecryptionAndOnAuditFailure` | service test | `useradmin/service_test.go:141-161` | PASS | Authorization/decrypter/audit failures | Fail-closed privacy coverage |
| 42 | `TestLookupRejectsUnboundedAndConflictingScopes` | service test | `useradmin/service_test.go:163-172` | PASS | Selector/limit validation | Bounds and scope coverage |
| 43 | `TestRetryDeletionEnforcesAuthorizationScopeAndForwardsLegalClaim` | service test | `useradmin/service_test.go:174-202` | PASS | Retry service boundary | Auth, request IDs, user/request scope |
| 44 | `TestUserAdminHTTPFailClosedProjectionAndBoundedValidation` | HTTP test | `httpapi/user_admin_controller_test.go:46-121` | PASS | Lookup route/middleware/encoder | Auth, spoof, fields, query bounds/logs |
| 45 | `TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit` | HTTP test | `httpapi/user_admin_controller_test.go:123-177` | PASS | Retry route/transaction/audit | CSRF, body, cross-scope, response/audit |

~~~~yaml
inventory_source_count: 45
audited_symbol_count: 45
inventory_complete: true
generated_groupings:
  - "Embedded SQL files are grouped by the selector/mutation statement they implement; no generated artifacts are in scope."
  - "Shared unchanged callers are included only for the task-252 authorization, logging, encryption, digest, transaction, and production-composition boundaries."
~~~~

## 6. Function-Level Audit

| Symbol/unit | Contract/invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundary | Performance/I/O | Result and evidence |
|---|---|---|---|---|---|---|
| `adminUserListSQL`, `adminUserGetByIDSQL`, `adminUserGetByDigestSQL` | Select only approved encrypted/projection columns and latest deletion summary | Page, exact ID, and exact digest have separate predicates | Context-bound query; rows closed and checked | No plaintext email, role, password, reset/session token, or unrelated data | Cursor ordering and SQL limit bound returned rows | PASS; SQL inspection and PostgreSQL projection tests |
| `adminDeletionRetrySQL` | Exact user/request scope; only failed legal categories | Ineligible rows produce no claim; eligible row resets to pending | `FOR UPDATE`, one transaction, deletion audit insert | No client status/category/body controls | Single bounded update/audit statement | PASS; legal and concurrent PostgreSQL tests |
| `PostgresAdminUserRepository` and compile assertion | Implements restricted interface only | Constructor accepts injected executor | No shared request state | Capability surface excludes arbitrary account edits | Thin repository wrapper | PASS; compile/runtime composition |
| `NewPostgresAdminUserRepository` | Captures executor without widening API | Valid construction path | No goroutine/resource ownership | Dependency injection keeps decryption out of repository | O(1) | PASS; coverage |
| `LookupAdminUsers` | Validates selector/limit before SQL and maps rows safely | Query, scan, close, and `rows.Err` failures return errors | Context propagates; rows close on return | Returns encrypted envelope, not plaintext | One query, bounded limit | PASS; repository tests and source audit |
| `RetryAdminDeletion` | Requires transaction and exact nonzero IDs | DB no-row/driver errors are returned for safe HTTP classification | Uses caller transaction; no independent commit | User/request scope is SQL-enforced | One query row plus audit CTE | PASS; repository tests |
| `validateAdminUserLookup` | One exact selector or one bounded page; max private bound 26 | Invalid limit, conflicting selectors, and malformed digest reject | Pure validation | Prevents broad/unscoped repository query | Constant-time | PASS; boundary tests |
| `scanAdminUser` | Scans only the privacy-minimized row shape | Nullable deletion summary is mapped without exposing internals | Row error returned; no retained mutable state | Encrypted email remains encrypted until service boundary | One row allocation | PASS; integration coverage; defensive error branches noted as optional coverage follow-up |
| `AdminUserRecord`, `AdminDeletionSummary` | DTO contains only approved storage projection | Nullable deletion row represented explicitly | Request-local values | No role/password/token/deletion-reason/lease/receipt fields | Small bounded DTO | PASS; projection assertions |
| `AdminUserLookup`, `AdminDeletionRetry` | Selector and fixed audit metadata are typed | No arbitrary selector/body fields | Passed through context-scoped call | Digest is versioned; retry metadata is fixed | Small values | PASS; service/repository tests |
| `AdminUserRepository` | Restricted lookup/retry contract | Compile-time interface catches widening mismatch | Repository owns query resources; transaction supplied for mutation | No role/password/edit capability | Narrow interface | PASS; compile assertion |
| `ErrForbidden`, page constants | Exact admin role and page bounds are centralized | Default 20, accepted 1–25 | No state | Authorization policy not caller-controlled | Constant-time | PASS; service boundary tests |
| `Actor`, `LookupRequest` | Actor is server-derived; request has only approved selectors | Conflicting selectors rejected downstream | Request-local | Client cannot provide actor role/ID/request ID | Small values | PASS; HTTP spoof tests |
| `User`, `Deletion`, `Page`, `RetryResult` | Public DTOs are an allowlist | JSON response omits internal fields by construction | No leases/reasons/receipts retained in response | Passwords, roles, tokens, and unrelated data absent | Bounded page | PASS; response field assertions |
| `piiDecrypter`, `lookupDigester`, `lookupAuditor` | Service depends on minimal capabilities | Missing dependencies fail closed | Calls are context-bound | Decryption/audit cannot be reached before authorization | Narrow interfaces aid test isolation | PASS; unauthorized and failure tests |
| `Service` and `NewService` | Captures four explicit dependencies | Nil dependency is rejected at operation boundary | No global mutable state | Boundary is explicit and injectable | O(1) | PASS; constructor/package coverage |
| `Service.Lookup` | Authorize before repository/decryption; audit before release | Exact/page, lookahead, decrypt, audit, and mapping errors return no page | Context passed to all dependencies; page truncation is deterministic | Only verified admin can reach decrypter; audit entity is fixed | At most limit+1 records and one decryption per returned row | PASS; service tests |
| `Service.RetryDeletion` | Authorize and require scoped IDs/transaction | Repository errors propagate without state fabrication | Uses gateway transaction; no direct commit | Cannot select another user's request or change fields | One repository call | PASS; service tests |
| `lookupRequest` | Normalizes email and creates active keyed digest; exact means limit 1 | Invalid limits/conflicts and digest errors fail | Context propagates to digest service | Plaintext email never crosses repository boundary | Exact lookup or bounded page+lookahead | PASS; digest/bounds tests |
| `authorize` | Requires nonzero actor ID, exact admin role, nonempty request ID | Missing/incorrect identity returns forbidden | Pure function | Rejects role/identity spoofing at service boundary | Constant-time | PASS; non-admin tests |
| `UserAdminService`, `UserAdminController` | HTTP boundary exposes only lookup/retry operations | Nil service/dependency paths are safe errors | Controller is request-local and gateway-owned | No arbitrary mutation interface | Small route configuration | PASS; HTTP tests |
| `NewUserAdminController` and `AdminRoutes` | Routes are explicit and control metadata is fixed | Missing service remains safely handled | No mutable request state | Retry is POST and has fixed audit action/entity | O(1) route construction | PASS; route inspection/tests |
| HTTP `Lookup` | Uses authenticated server actor and safe envelope | Validation/service errors map without internals | User context is propagated | Query cannot override actor; response allowlist | Page already bounded by service | PASS; HTTP test |
| HTTP `RetryDeletion` | Path IDs and service result determine mutation/audit state | Invalid UUID, service, category, and response paths fail closed | Runs inside existing transaction gateway | Body cannot set status/category/reason/user/request IDs | One scoped transaction | PASS; CSRF/scope/audit test |
| HTTP `validateAdminUserLookup` | Exact query key allowlist; duplicate/conflict/limit checks | Unknown, duplicate, malformed UUID, and out-of-range values reject | Pure request validation | No arbitrary edit or enumeration parameter | Constant-time in small key set | PASS; bounded-validation tests |
| `validateAdminDeletionRetry` | Only valid path UUIDs and empty body accepted | Nonempty/mutation-field body rejected | Pure request validation | Client cannot set deletion transition metadata | Body size/parse handled by gateway | PASS; mutation-body tests |
| `adminUserError`, validation/dependency errors | Fixed status/error codes and generic not-found/dependency responses | Forbidden, validation, no-row, transient, canceled, and unknown paths safe | No cause text returned | No SQL/deletion internals or tokens in response | Constant-time mapping | PASS; safe error assertions |
| `adminAuditSnapshotSchemas` retry rule | Only fixed status/category enums and retry action are accepted | Invalid enum/schema rejected before persistence | Audit is transaction-local | No free text, reason, lease, or sensitive fields | Small canonical JSON | PASS; audit schema inspection/tests |
| `PersistAuditEntry`, `WithMutationAudit`, sanitizer path | Validate/canonicalize and commit mutation plus audit atomically | Mutation/audit/commit/serialization failures roll back | Context and deferred rollback preserved | Audit is fail-closed and privacy-sanitized | One transaction and bounded snapshot | PASS; repository/audit tests |
| `NewProduction` admin wiring | One repository → service → controller → admin gateway chain | Construction returns errors rather than partial route exposure | Dependencies are process-owned | Production uses injected encryption/digest/audit, not controller decryption | O(1) composition | PASS; app wiring inspection |
| `RequireAdmin`, `requireAuth`, `authenticatedUser` | Role/ID come only from verified signed auth state | Anonymous 401; non-admin 403; missing locals fail closed | Refresh/session validation remains in auth path | Headers/body cannot impersonate actor | Auth dependency cost unchanged | PASS; HTTP auth/spoof tests |
| `registerV1Routes`, `serverRequestID`, `instrument` | Gateway order and server correlation are fixed | Middleware short-circuits before handler; errors remain generic | Per-request UUID/context; no raw query/body logging | Client request ID and role spoofing ignored; logs exclude PII/secrets | Standard middleware overhead; rate limits bound abuse | PASS; route/log tests |
| `EncryptionService.DecryptPII` | AES-GCM authenticates envelope before plaintext release | Invalid envelope/key/context returns error | Context-bound key load; no retained plaintext | Called only by useradmin service after authorization | One bounded envelope decrypt | PASS; primitive and boundary inspection |
| `LookupDigestService.DigestForWrite` | HMAC digest uses active version/key | Missing active key or invalid input errors | Context-bound key load | Repository receives keyed digest, not plaintext | Constant-time digest operation | PASS; exact-email test |
| Deletion schema/state constraints | Failed/pending and failure category/retry invariants are represented in DB | User deletion can be nulled after hard erase; active request constraints remain | SQL row lock is compatible with worker state machine | Retry cannot bypass user/request scope | Existing indexes support active/worker paths; optional latest-history index noted below | PASS; migrations, SQL, and integration tests |
| `TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized` | Exercises exact/page/projection contract | Covers page cursor, ID, digest, and validation cases | Real DB rows and cleanup | Forbidden projection fields asserted | Bounded query behavior observed | PASS |
| `TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures` | Exercises legal category/count state machine | Permanent/unknown/exhausted transient pass; transient below threshold and scope mismatch fail | Real transaction state changes | Cross-user request cannot be claimed | Atomic SQL behavior observed | PASS |
| `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits` | At-most-once retry claim and audit | One success/one no-row result | Concurrent transactions and row lock | Scope remains exact under contention | One mutation/audit wins | PASS; no race reported |
| `TestLookupProjectsOnlyApprovedFieldsWithBoundedPaginationAndAudit` | Service maps only public fields and strips lookahead | Decrypt/audit/page paths exercised | Context-bound fake dependencies | No decryption before authorization | limit+1 private lookahead | PASS |
| `TestLookupExactEmailNormalizesDigestAndAuditsEntity` | Exact selector uses normalized active digest | Exact result and audit entity verified | Request-local fake call sequence | Plaintext stops at digest boundary | One exact row | PASS |
| `TestLookupFailsClosedBeforeUnauthorizedDecryptionAndOnAuditFailure` | Authorization precedes sensitive capability use | Unauthorized and audit failure return no data | No decryption/audit leakage | Explicit decryption-call assertions | No work after denial | PASS |
| `TestLookupRejectsUnboundedAndConflictingScopes` | Service rejects unsafe selector combinations | Limit/conflict branches covered | Pure validation | Prevents enumeration widening | Constant-time | PASS |
| `TestRetryDeletionEnforcesAuthorizationScopeAndForwardsLegalClaim` | Service forwards server actor and exact IDs | Unauthorized/cross-scope/valid claim paths | Transaction handle forwarded, not committed by service | No client mutation fields | One call | PASS |
| `TestUserAdminHTTPFailClosedProjectionAndBoundedValidation` | HTTP envelope and query allowlist are safe | Auth, spoof, forbidden fields, duplicate/unknown/bounds, log checks | Fiber request context | No role/password/token/PII disclosure | Bounded endpoint | PASS |
| `TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit` | Mutation requires CSRF and safe audit/response | Invalid body, CSRF, cross-scope, success paths | Transactional audit commit asserted | No status/category/body spoofing | One bounded retry | PASS |

## 7. Findings

No blocking or important findings remain in the task-252 review surface. The implementation satisfies the privacy, authorization, state-transition, concurrency, audit, response, and logging acceptance criteria.

Optional follow-up:

- **O-1 — Add direct tests for defensive error branches and consider a latest-deletion lookup index.** The complete repository profile is 91.9%; the task-local repository functions report `LookupAdminUsers 85.0%`, `RetryAdminDeletion 85.7%`, `validateAdminUserLookup 85.7%`, and `scanAdminUser 69.2%`. The service and HTTP profiles are 93.8% and 87.4%, with their uncovered lines likewise in defensive error/serialization branches. These gaps do not leave an acceptance path untested: focused success, denial, bounds, scope, audit, and concurrent-claim tests pass. Separately, the per-account latest-deletion `LATERAL` lookup has no general `(user_id, requested_at DESC, id DESC)` index in the reviewed migrations; the page is bounded but a large historical table could make admin lookup slower. Neither observation changes the decision for this restricted, rate-limited operation.

~~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Evidence |
|---|---|---:|---|---|
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/useradmin ./internal/httpapi` | `backend/` | 0 | PASS | Complete focused service and HTTP packages pass. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/useradmin ./internal/httpapi -run 'TestLookup\|TestRetryDeletion\|TestUserAdmin'` | `backend/` | 0 | PASS | Task-focused service/controller tests pass. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/useradmin ./internal/httpapi -run 'TestLookup\|TestRetryDeletion\|TestUserAdmin'` | `backend/` | 0 | PASS | No race report on authorization, decryption, pagination, retry, or HTTP paths. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/repository -run 'TestPostgresAdminUser\|TestPostgresAdminDeletion'` | `backend/` | 0 | PASS | Exact/bounded projection, legal retry, scope, and concurrent claim tests pass. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/repository -run 'TestPostgresAdminUser\|TestPostgresAdminDeletion'` | `backend/` | 0 | PASS | No race report on concurrent retry claim/audit. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -p 1 -coverprofile=/tmp/task-252-repository-full.cover ./internal/repository` | `backend/` | 0 | PASS | Repository package 91.9%; task-local function results are recorded in section 10. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -coverprofile=/tmp/task-252-useradmin.cover ./internal/useradmin` | `backend/` | 0 | PASS | Useradmin package 93.8%; task functions and uncovered defensive branches recorded. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -coverprofile=/tmp/task-252-httpapi.cover ./internal/httpapi` | `backend/` | 0 | PASS | HTTP package 87.4%; task functions and uncovered defensive branches recorded. |
| `go tool cover -func=/tmp/task-252-repository-full.cover` filtered to `admin_user_repository.go` | `backend/` | 0 | PASS | `New...` 100.0%; lookup 85.0%; retry 85.7%; validation 85.7%; scan 69.2%. |
| `gofmt -d` on all reviewed Go files | `backend/` | 0 | PASS | No formatting diff. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend/` | 0 | PASS | No diagnostics. |
| `env GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS | No vulnerabilities in called/imported code; unreachable required-module advisories were not reachable. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with existing warning | OpenAPI is valid; existing OAuth callback 302-only response warning remains. Task 252 intentionally adds no OpenAPI surface. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | Ordered task list validates; task 252 remains PREPARED. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Requirement/design traceability validates. |
| `python3 scripts/check.py` | repository root | 1 | EXTERNAL ENVIRONMENT FAILURE | Checks before local-stack gate pass; `verify-local-stack.py --keep-services` fails while applying migration 000008 because relation `entitlements` does not exist in the shared dirty PostgreSQL state. This is not caused by task-252 code and no task-list/production file was edited. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-252-review.md` | repository root | 0 | PASS | Final evidence structure, counts, decision, and required gates validate. |
| `git diff --check -- docs/implementation/02_TASK_LIST.md` | repository root | 0 | PASS | Existing task-list changes contain no whitespace errors; file was not edited by review. |

## 9. Files Inspected and Staleness Fingerprints

Hashes were computed after the final focused verification and before this evidence file was written. Task-owned hashes match the preparation manifest. Shared files are current snapshots; only the reviewed task symbols were attributed to task 252. The task table hash was captured before and after review and the file was not edited.

| File | Purpose | SHA-256 |
|---|---|---|
| `backend/internal/repository/admin_user_repository.go` | Restricted SQL repository | `c16d32d40b459ec2faef972464022bc4f92ab030ade99bafdc3d65f08b732326` |
| `backend/internal/repository/admin_user_repository_test.go` | PostgreSQL projection/retry/concurrency tests | `8f60600c2f3a0f65e34417820e692e8c4b924fcd4157de32a59b9bec1a95457a` |
| `backend/internal/repository/sql/admin_user_list.sql` | Bounded page SQL | `459f9223f0102d632ec6abdfc9d24b5cd36c600ddfdc3577db305568770c8801` |
| `backend/internal/repository/sql/admin_user_get_by_id.sql` | Exact ID SQL | `99b0bf5403da6e772141b9dbcbde12ab4c6e4b9087a19e325e319a6869a81a22` |
| `backend/internal/repository/sql/admin_user_get_by_digest.sql` | Exact digest SQL | `0125bd5d03e287fbfab9c7fb9b9de42a978463d8b9f0d35a60448d8fa838bee14` |
| `backend/internal/repository/sql/admin_deletion_retry.sql` | Legal atomic retry SQL | `5a4c43103305adb37b4aedde810db15b0146162c7555a09f467e6ed9474aea16` |
| `backend/internal/useradmin/service.go` | Service boundary | `ff7c43b484a4ced32e6036cea8676707b8854ff912719d324c0eeb4a3b85e8df` |
| `backend/internal/useradmin/service_test.go` | Service tests | `de51a99c69009e1ee08ea065f8be1cadaf7953e4cb085a1066246a18360c221f` |
| `backend/internal/httpapi/user_admin_controller.go` | HTTP boundary | `ffc6606c3599956aa7877a400a9c1f4f9bdf37976e91e31a921e0c7aefbc4e3f` |
| `backend/internal/httpapi/user_admin_controller_test.go` | HTTP tests | `2d5b0b79af1478764a9f416f7805f5234e441a9bfad8ffe72c1a90c973c128c9` |
| `backend/internal/repository/types.go` | Shared task DTO/interface symbols | `7a0069590989b8fe0311c960021c5da87cc3d568b9ca708a7e586b620511f730` |
| `backend/internal/repository/compliance_repository.go` | Audit schema/transaction symbols | `57790181b840c7f494e44dbd62a7c69be6210cff79987652db6bd6a3852712f6` |
| `backend/internal/app/app.go` | Production composition | `e9fa64094fdbff1b2b8e88857dd119b400d337f9401464ee284cffb2e17c5409` |
| `backend/internal/httpapi/admin_controller.go` | Admin gateway caller | `cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/httpapi/router.go` | Auth/order/request-ID/log callers | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/httpapi/auth_middleware.go` | Signed identity caller | `71b8edaa05c479e31703871ac7066dc384d534e40dfb4cb565bee5cfc8f0f7b9` |
| `backend/internal/security/encryption.go` | AES-GCM decryption boundary | `97a6d0c4f81ef22bfddba4c9cd032ea9df5c22bcac30e7ec1a949205e90c5751` |
| `backend/internal/security/lookup_digest.go` | HMAC lookup digest boundary | `50066879705e2902df70124b45e9d6648c037cc070b4c6ac7f3643821578add9` |
| `database/migrations/000009_consent_deletion.up.sql` | Deletion request/audit state schema | `844f43166badaa1eb75a9b740b795256029c0860dbc282c7a5ca60b618d03165` |
| `database/migrations/000014_deletion_request_hardening.up.sql` | Deletion failure/retry hardening | `45d0232a97af3370503758b46f132a94cea73c24a6042e66036fa83ef0f44244` |
| `docs/implementation/preparations/task-252.md` | Preparation manifest | `4dee7979691ebefeeefe46fe85563f3e157d8946c9014e6d5836ed7d77d2352d` |
| `docs/implementation/02_TASK_LIST.md` | Unchanged task-status control | `a659cdcf0bbdd8e00d83c6f08167f1ea262dfe60837890505b3683615a4d12d6` |

~~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "No prior task-252 review evidence existed; the preparation report was checked against current hashes and symbols."
  - "The worktree is concurrently dirty; shared-file findings are limited to reviewed task-252 symbols and direct callers."
  - "The task-list hash was captured before and after review; the PREPARED status and all task-list edits were preserved."
~~~~

## 10. Coverage and Exceptions

- [x] Complete focused useradmin and HTTP package tests pass.
- [x] Complete serialized repository package coverage profile passes.
- [x] Focused race tests cover service, HTTP, repository, and concurrent retry paths.
- [x] Coverage profiles and task-local function measurements are recorded.
- [ ] Every task-local defensive branch reaches 100%; the remaining branches are recorded as optional follow-up O-1.
- [x] No coverage exception was silently introduced; the observed gap is explicitly disclosed.

~~~~yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_paths:
  - "/tmp/task-252-useradmin.cover"
  - "/tmp/task-252-httpapi.cover"
  - "/tmp/task-252-repository-full.cover"
observed_package_coverage: "useradmin 93.8%; HTTP 87.4%; repository 91.9%"
observed_task_local_function_coverage: "repository New 100.0%, Lookup 85.0%, Retry 85.7%, validation 85.7%, scan 69.2%; service and HTTP profiles are 93.8% and 87.4% package-wide"
coverage_passed: true
coverage_note: "All required behavior paths pass; uncovered statements are defensive dependency/scan/error branches and are the optional O-1 follow-up."
~~~~

The package percentages include unrelated functions in shared packages. No uncovered line exposes an acceptance-path bypass: the authorized projection, bounded page, keyed digest, decryption boundary, legal-state retry, concurrent claim, audit, cross-scope denial, CSRF, and safe response/log paths are exercised.

## 11. Negative and Regression Checks

- [x] Exact ID, exact normalized-email digest, and bounded cursor lookup use distinct parameterized SQL paths.
- [x] Private `limit+1` lookahead is bounded at 26 and never reaches the response.
- [x] Repository SQL does not select plaintext email, roles, password hashes, reset tokens, session tokens, deletion reasons, leases, receipts, or unrelated account data.
- [x] Decryption is injected only into the authorized service boundary and is not performed by SQL/repository code.
- [x] Empty/missing dependencies, unauthorized actors, missing request IDs, decryption failure, audit failure, no-row, and driver failure paths fail closed or map to generic errors.
- [x] Retry requires exact user/request scope, `failed` status, and permanent/unknown/exhausted-transient eligibility.
- [x] Retry resets only the documented queue state and writes one `failed → pending` deletion audit entry in the same transaction.
- [x] Two concurrent retries cannot both claim the row; the test observes one success and one no-row result.
- [x] Admin audit is request-correlated, schema-allowlisted, canonicalized, and transactional; free text and sensitive fields are rejected.
- [x] Anonymous/non-admin requests, client role/identity/request-ID spoofing, CSRF failures, unknown/duplicate/conflicting query keys, malformed UUIDs, out-of-range limits, and mutation body fields are rejected.
- [x] No route or DTO supports role mutation, password access/reset, token/session access, impersonation, or arbitrary account editing.
- [x] Instrumentation does not log raw query/body or password/token/PII material; response/error assertions enforce the same boundary.
- [x] `go test -race` focused paths, `go vet ./...`, govulncheck, OpenAPI lint, traceability, and evidence validation pass.
- [x] No production code or task-list file was changed by this review; only this review evidence file was added.

## 12. Decision

The task-252 implementation is **PASSED**. There are zero blocking and zero important findings. The single optional follow-up is disclosed coverage/performance hardening and does not affect the acceptance criteria or security boundary. The aggregate failure is environmental: the shared local PostgreSQL migration check cannot apply migration 000008 because `entitlements` is absent in the already-dirty database state; focused task-252 verification remains green.

~~~~yaml
decision: "PASSED"
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~~

## 13. Repair Context

No repair is required for task 252. The review did not modify production code, tests, migrations, task status, or the task list. The only written artifact is this review evidence document.

~~~~yaml
repair_required: false
repair_files: []
repair_summary: "N/A; task-252 acceptance paths pass and no blocking or important finding remains."
~~~~
