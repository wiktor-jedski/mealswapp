# ARCH-009 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for ARCH-009, the Administration Module. It verifies architectural collaboration across authenticated HTTP routing, administration services, persistence and audit transactions, external-data curation, search and filter consumers, account export and deletion, generated clients, and the Administration Panel.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-009 |
| Name | Administration Module |
| Source Documents | `docs/architecture/ARCH-009.md`, `docs/architecture/01_SOFT_ARCH_DESIGN.md`, `docs/design/DESIGN-009.md` |
| Related Units | AdminController, ExternalSearchProxy, DataImporter, ItemCurator, TagManager, UserAdminPanel, authentication middleware, repositories, audit coordinator, search/filter consumers, generated clients, Administration Panel |
| Collaborating Architecture | ARCH-001, ARCH-002, ARCH-005, ARCH-006, ARCH-008, ARCH-011, ARCH-012, ARCH-013, ARCH-015, ARCH-018 |
| Related Requirements | SW-REQ-043, SW-REQ-054, SW-REQ-055, SW-REQ-056, SW-REQ-057, SW-REQ-072, SW-REQ-073, SW-REQ-090 |

## IT-ARCH-009-001 Authenticated Administration Authorization and UI Isolation

### Intent

Verify that server-derived authentication and role state controls the Administration Panel and every admin route, while anonymous, standard-user, malformed, and spoofed identities remain isolated from administrative behavior.

### System Under Test

ARCH-009 AdminController and UserAdminPanel authorization boundary.

### Real Components

- Fiber router, JWT-cookie authenticator, AdminController, and `RequireAdmin`
- frontend auth-session projection, SidebarComponent, AdministrationPanel, and generated contracts

### Allowed Test Doubles

- Browser route interception may provide generated backend envelopes.
- The authorized fixture handler may stand in for a specific admin feature after the real gateway admits the request.

### Trigger / Stimulus

Anonymous, standard-user, verified-admin, malformed-session, and header-spoofed clients request documented and undocumented administration routes.

### Expected Integrated Behavior

1. Anonymous requests receive 401 and standard users receive 403 before admin feature dispatch.
2. Client-supplied role, user ID, and request ID do not override verified server state.
3. Verified admins reach only documented routes and receive a server-generated correlated request ID.
4. The Administration navigation and panel render only for a verified admin session; direct non-admin navigation fails closed.
5. Keyboard, responsive, and light/dark UI paths retain the same authorization projection.

### Required Evidence

- Observable HTTP status/envelopes, handler admission, route allowlist, browser URL, navigation visibility, panel state, keyboard access, and accessibility result.

### Requirement Traceability

- SW-REQ-054

### Verification Status

Implemented by:

- `backend/internal/httpapi/admin_controller_test.go::TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation`
- `frontend/tests/admin-access-shell.spec.ts::verified admins reach the keyboard-operable responsive panel in light and dark themes`
- `frontend/tests/admin-access-shell.spec.ts::anonymous and standard users see no administration control and direct routes fail closed`

Status: PASS.

## IT-ARCH-009-002 External Candidate Curation to Local Search

### Intent

Verify the nominal ARCH-009 flow from read-only external search through normalization and editable curation to explicit local persistence and immediate search visibility.

### System Under Test

ARCH-009 ExternalSearchProxy and DataImporter collaboration with ARCH-012, ARCH-005, search consumers, generated clients, and the Administration Panel.

### Real Components

- ExternalSearchProxy, DataNormalizer, and bounded provider orchestration
- production composition, authenticated Fiber external-search/import controllers, and CSRF enforcement
- DataImporter with PostgreSQL food/import/audit repositories
- catalog and substitution search services
- generated frontend client types and ExternalImportWorkflow

### Allowed Test Doubles

- Provider HTTP endpoints may use deterministic `httptest.Server` boundary fixtures while the production provider clients, proxy, controller, importer, and PostgreSQL repositories remain real.
- Supporting browser contract tests may use deterministic transport fixtures; they do not replace the executable provider-to-persistence path.
- Redis may be omitted where immediate uncached search visibility is the asserted outcome.

### Trigger / Stimulus

An administrator searches USDA, OpenFoodFacts, and both providers; edits a normalized candidate, classifications, macros, and liquid-density provenance; confirms import; then opens the item in local search.

### Expected Integrated Behavior

1. External search is read-only and returns bounded normalized candidates and warnings.
2. Provider selection and pagination survive generated-client serialization.
3. The administrator can correct normalization warnings and assign global classifications.
4. Persistence occurs only after explicit confirmation and atomically creates the food, import, and audit records.
5. The imported item is immediately visible to catalog and substitution consumers and the browser local-search handoff.

### Required Evidence

- Provider HTTP queries, normalized fields and warnings, read-only pre-import counts, curated request body, PostgreSQL row counts, audit row, local search results, and rendered result identity.

### Requirement Traceability

- SW-REQ-055
- SW-REQ-090

### Verification Status

Implemented by:

- `backend/internal/app/task261_external_import_integration_test.go::TestTask261ProviderHTTPImportPostgresFlow`
- `backend/internal/dataimporter/integration_test.go::TestCuratedImportTransactionalWorkflow`
- `frontend/tests/external-import-workflow.spec.ts::searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result`

Status: PASS.

## IT-ARCH-009-003 Replay, Conflict, and Transactional Rollback

### Intent

Verify that idempotent retry, natural-identity conflict, normalized-name confirmation, validation failure, and audit/repository failure preserve one authoritative mutation outcome without false success.

### System Under Test

ARCH-009 DataImporter and AdminController transactional mutation boundary.

### Real Components

- DataImporter, PostgreSQL import/food/idempotency/audit repositories, and search repository
- AdminController, CSRF/rate/validation middleware, and audit coordinator
- generated import client and ExternalImportWorkflow retry state

### Allowed Test Doubles

- A failing audit coordinator may force the architecture failure boundary.
- Browser transport may emulate an ambiguous response after the server-side commit.

### Trigger / Stimulus

Exact and changed-body retries, provider-identity and normalized-name conflicts, concurrent confirmations, repository rejection, audit failure, and an ambiguous client retry are submitted.

### Expected Integrated Behavior

1. Exact replay returns the original identity without duplicate food, import, idempotency, or audit rows.
2. Changed-body key reuse and provider identity conflict return conflict without mutation.
3. Normalized-name merge requires explicit confirmation and preserves one authoritative food identity.
4. Validation, repository, or audit failure rolls back mutation and claim state and never returns an uncommitted success body.
5. The browser retains the same key for ambiguous replay, creates no duplicate item, and exposes merge confirmation only for the documented conflict.

### Required Evidence

- Stable IDs, exact database counts, rollback absence checks, concurrent result counts, HTTP error/success envelopes, and browser key/body/result assertions.

### Requirement Traceability

- SW-REQ-055
- SW-REQ-090

### Verification Status

Implemented by:

- `backend/internal/dataimporter/integration_test.go::TestCuratedImportTransactionalWorkflow`
- `backend/internal/httpapi/admin_controller_test.go::TestAdminMutationRollsBackWhenTransactionalAuditFails`
- `frontend/tests/external-import-workflow.spec.ts::searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result`
- `frontend/tests/external-import-workflow.spec.ts::replays one ambiguous import with the same key and displays one local item identity`

Status: PASS.

## IT-ARCH-009-004 Private Item Isolation, Export, and Erasure

### Intent

Verify that administration and global curation do not weaken private custom-item ownership, and that export and account deletion coordinate storage, authentication, and cache boundaries without affecting another user or global data.

### System Under Test

ARCH-009 collaboration with private-item, export, account-deletion, repository, cache, and authentication boundaries.

### Real Components

- production HTTP composition, custom-item and compliance repositories, export service, account deletion service, authentication, PostgreSQL, and Redis
- generated account-data client, Administration Panel private-data controls, and real browser/backend transport

### Allowed Test Doubles

- A failing cache purger forces a retryable deletion attempt before the real Redis purger recovers.
- Browser fixture setup may create owner-safe data through the live API; the evidence action itself uses the rendered Administration Panel and generated client against the real backend.

### Trigger / Stimulus

Two users and one global item are created; one owner exports and deletes private data, requests account erasure, encounters a cache failure, then retries to completion.

### Expected Integrated Behavior

1. Private item access and export remain owner scoped and omit owner identifiers from client payloads.
2. Pending deletion blocks new private writes.
3. Failed cache cleanup remains retryable without restoring removed active private data.
4. Completion removes owner PII, private items, sessions, and owner cache, then denies stale access and login.
5. Another user's private row/cache and the global curated row remain byte-for-byte unchanged; the receipt remains pseudonymous.

### Required Evidence

- HTTP status, export payload, PostgreSQL rows, Redis keys, retry transitions, post-erasure authentication denial, and survivor comparisons.

### Requirement Traceability

- SW-REQ-043
- SW-REQ-072
- SW-REQ-073

### Verification Status

Implemented by:

- `backend/internal/app/task240_custom_item_erasure_integration_test.go::TestTask240CustomItemErasureIntegration`
- `frontend/tests/task261-real-admin-flow.spec.ts::Admin Panel generated client deletes exported private data and publishes a dynamic filter`

Status: PASS.

## IT-ARCH-009-005 Classification Mutation and Consumer Invalidation

### Intent

Verify that global classification CRUD, conflict/in-use safeguards, transactional audit, Redis generation invalidation, search/filter consumers, generated clients, and the Administration Panel converge on committed authoritative state.

### System Under Test

ARCH-009 TagManager collaboration with ARCH-005, ARCH-011, search/filter consumers, and the Administration Panel.

### Real Components

- classification HTTP controller/service, AdminController, audit coordinator, and invalidator
- PostgreSQL classification repository and filter-option service
- Redis shared generation and multiple filter/search service instances
- generated filter/admin clients and Svelte administration/substitution UI

### Allowed Test Doubles

- An in-memory classification repository is allowed for HTTP sequencing and forced audit failure.
- Supporting browser contract tests may expose deterministic committed state; the dynamic-filter evidence uses the real Administration Panel, generated clients, backend, PostgreSQL, Redis, and Chromium without route stubs.

### Trigger / Stimulus

An admin creates, renames, attempts duplicate/in-use deletion, and deletes a classification while peer service instances and substitution UI hold prior projections.

### Expected Integrated Behavior

1. Successful mutations commit audit before invalidation and return authoritative state.
2. Duplicate and in-use conflicts do not invalidate or mutate consumers.
3. Audit failure rolls back and does not invalidate.
4. A committed generation change refreshes peer filter/search consumers and rejects stale cache writes.
5. Renamed/created/deleted labels appear in generated-client UI and substitution filters without hardcoded fallback policy.

### Required Evidence

- HTTP statuses, audit and invalidation counts, PostgreSQL options, Redis generation tokens, stale-write rejection, rendered labels, and accessibility results.

### Requirement Traceability

- SW-REQ-057

### Verification Status

Implemented by:

- `backend/internal/httpapi/classification_admin_controller_test.go::TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation`
- `backend/internal/cache/classification_generation_integration_test.go::TestClassificationGenerationLiveRedisCoordinatesInstancesAndRejectsStaleWrite`
- `backend/internal/search/filter_options_integration_test.go::TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration`
- `frontend/tests/task261-real-admin-flow.spec.ts::Admin Panel generated client deletes exported private data and publishes a dynamic filter`
- `frontend/tests/task259-frontend-gate.spec.ts::classification administration refreshes substitution filters and remains accessible in every viewport and theme`
- `frontend/tests/admin-data-management.spec.ts::classification conflicts and legal deletion retries preserve authoritative state`

Status: PASS.

## IT-ARCH-009-006 Manual Global Item Lifecycle and Audit-Safe UI

### Intent

Verify nominal manual global item create/read/update/delete, stable creation replay, validation, transactional audit, private-route isolation, and authoritative UI refresh.

### System Under Test

ARCH-009 ItemCurator collaboration with AdminController, repositories, audit persistence, generated clients, and AdminDataManagement.

### Real Components

- manual-item HTTP controller, ItemCurator service contract, AdminController, CSRF, and audit coordinator
- PostgreSQL manual-food repository coverage
- generated admin client and AdminDataManagement UI

### Allowed Test Doubles

- The HTTP test may use an ItemCurator service fixture while retaining the real gateway and audit sequence.
- Browser transport may provide committed authoritative item responses and forced audit failure.

### Trigger / Stimulus

An admin creates, replays, reads, updates, and deletes a global item; submits invalid liquid and ownership fields; cancels/accepts deletion; and encounters audit failure.

### Expected Integrated Behavior

1. Valid create/read/update/delete succeeds with before/after audit evidence.
2. Exact create replay has no second audit side effect; conflict and invalid fields fail closed.
3. Global items remain separate from owner-scoped private routes.
4. The UI validates liquid density, confirms deletion, refreshes authoritative state, and never renders false success after audit failure.

### Required Evidence

- HTTP results, replay/audit counts, persisted CRUD state, route isolation, confirmation focus, refreshed fields, and safe failure copy.

### Requirement Traceability

- SW-REQ-056

### Verification Status

Implemented by:

- `backend/internal/httpapi/manual_item_controller_test.go::TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots`
- `backend/internal/repository/manual_food_repository_test.go::TestPostgresManualFoodItemCRUD`
- `frontend/tests/admin-data-management.spec.ts::manual global item CRUD validates, confirms, refreshes, and never shows audit false success`

Status: PASS.

## IT-ARCH-009-007 Restricted User Deletion Retry

### Intent

Verify that the restricted user-administration surface exposes only the documented deletion retry, enforces actor/request scope and CSRF, audits accepted transitions, and refreshes authoritative UI after conflict or success.

### System Under Test

ARCH-009 UserAdminPanel collaboration with authentication, deletion persistence, audit coordination, generated clients, and AdminDataManagement.

### Real Components

- AdminController, JWT/CSRF middleware, UserAdminController, and audit coordinator
- user-admin service/repository legal-transition and concurrent-claim coverage
- generated admin client and AdminDataManagement UI

### Allowed Test Doubles

- The HTTP service fixture may force cross-scope and legal-result states after the real gateway.
- Browser transport may model stale/concurrent server outcomes.

### Trigger / Stimulus

An administrator looks up the privacy-minimized projection and retries an eligible deletion; missing CSRF, client role fields, cross-scope IDs, stale conflict, and a later legal retry are attempted.

### Expected Integrated Behavior

1. Unauthorized, unscoped, or malformed requests fail before retry dispatch.
2. A legal retry claims the request once and commits a safe audit linked to the deletion request.
3. Responses omit internal deletion, credential, token, and unrelated-user data.
4. Conflict does not display optimistic success; the UI reloads authoritative state and permits only a later eligible transition.

### Required Evidence

- HTTP status/call counts, actor and request IDs, audit changes, response-field exclusions, concurrent repository claim, and refreshed browser state.

### Requirement Traceability

- SW-REQ-054
- SW-REQ-073

### Verification Status

Implemented by:

- `backend/internal/httpapi/user_admin_controller_test.go::TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit`
- `backend/internal/repository/admin_user_repository_test.go::TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits`
- `frontend/tests/admin-data-management.spec.ts::classification conflicts and legal deletion retries preserve authoritative state`

Status: PASS.

## Coverage Matrix

| Required path | Obligations |
| --- | --- |
| Nominal | IT-ARCH-009-002, IT-ARCH-009-005, IT-ARCH-009-006, IT-ARCH-009-007 |
| Authorization | IT-ARCH-009-001, IT-ARCH-009-007 |
| Isolation | IT-ARCH-009-001, IT-ARCH-009-004, IT-ARCH-009-006 |
| Replay | IT-ARCH-009-003, IT-ARCH-009-006 |
| Conflict | IT-ARCH-009-003, IT-ARCH-009-005, IT-ARCH-009-007 |
| Rollback | IT-ARCH-009-003, IT-ARCH-009-005, IT-ARCH-009-006 |
| Provider | IT-ARCH-009-002 |
| Normalization | IT-ARCH-009-002, IT-ARCH-009-003 |
| Deletion | IT-ARCH-009-004, IT-ARCH-009-006, IT-ARCH-009-007 |
| Invalidation | IT-ARCH-009-005 |
| UI | IT-ARCH-009-001 through IT-ARCH-009-007 |
| Degraded | IT-ARCH-009-003, IT-ARCH-009-004, IT-ARCH-009-005, IT-ARCH-009-006, IT-ARCH-009-007 |

## SWE.5 Completion Criteria

- Every obligation is implemented by at least one test carrying its obligation ID.
- Every listed test exercises at least two real collaborating units and verifies observable architectural outcomes.
- Test doubles are limited to provider, browser transport, clock/failure, or feature boundaries where real infrastructure is impractical.
- Focused backend and frontend integration suites pass.
- Requirement/design traceability and task-list validation pass.
- No obligation remains uncovered.
