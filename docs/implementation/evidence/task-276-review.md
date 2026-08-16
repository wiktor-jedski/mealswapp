# Review Evidence: Task 276 — DESIGN-009: AdminController

```yaml
task_id: 276
component: "DESIGN-009: AdminController"
static_aspect: "Phase 08.02 Safe Administrator Bootstrap"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-27T21:50:00Z"
review_agent: "independent-reviewer-third-repair-cycle"
evidence_file: "docs/implementation/evidence/task-276-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677 plus current worktree diff"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "/home/wiktor/.agents/skills/code-review-skill/reference/go.md"
repair_context_required: true
```

## 1. Task Source

Task 276 defines and implements an operator-only administrator-bootstrap CLI for local development, deployed development, and production. It must resolve one existing user through the configured encrypted-email lookup path, support an optional UUID selector, require a verified Login Method and real credential, allow only first-administrator creation, make same-target replay a no-op, serialize concurrent attempts, atomically persist the role and truthful operator-origin audit, require explicit environment and production confirmation, emit no PII or secret material, remove the unusable seeded administrator, and document sign-out/sign-in for fresh claims. It must not add a public promotion endpoint or mark email verified.

Dependency: Task 247 is PASSED. The task row is externally PREPARED and was not edited by this review.

Verification criteria include the repaired canonical email contract and collision-safe legacy digest reindexing across registration, login, OAuth, and bootstrap; parser-backed credential eligibility with malformed Argon2, Base64, short hash, and short salt denial; nullable replay audit ID semantics; isolated PostgreSQL coverage; safe CLI output; seed and reauthentication behavior; the installed `golang-security` review; and focused `go test -race`.

## 2. Pre-Review Gates

- [x] Input status is PREPARED and dependency 247 is PASSED.
- [x] The updated preparation evidence was read and checked against current files.
- [x] Baseline and current worktree diff were independently reconstructed.
- [x] `/home/wiktor/.agents/skills/phase-orchestrator/templates/review_checklist.md` was read completely.
- [x] `code-review-skill` was invoked exactly once; the relevant Go guide was read completely.
- [x] The repository-required `golang-security` guidance was read and applied.
- [x] No implementation code or task-list file was changed by this review.
- [x] All changed implementation paths and impacted audit/user-admin dependencies were inspected.

```yaml
pre_review_gates_passed: true
blocking_issue: "None. B-1, I-1, O-1, replay-ID semantics, and I-2 producer/parser bounds are verified in the current worktree."
```

## 3. Review Baseline and Change Surface

The clean baseline is `f9a646ffede5c8a63ad787a07eaba7efc0433677`. I used `git status`, tracked and untracked diff/name reconstruction, symbol searches, direct file inspection, and caller/dependency inspection. The current worktree contains the prepared Task 276 implementation and repair-cycle changes. The task-list PREPARED transition is metadata and was not touched.

The third-cycle surface is the existing Task 276 implementation plus the I-2 repair in `auth/password.go`, its boundary test, and updated preparation evidence. The constructor now accepts only 16–64 byte key lengths, matching the parser’s decoded-hash bound. The task-owned implementation and current repair files were independently enumerated; the current diff includes 33 tracked paths plus new CLI/bootstrap/resolver/migration/documentation paths. The task-list PREPARED transition remains external and untouched.

B-1 is fixed: `AdminAuditEntry` now carries `ActorKind` and `*uuid.UUID`, list SQL selects `actor_kind`, the scanner reads nullable actor IDs, validation distinguishes administrator/operator invariants, and the bootstrap lifecycle reads the operator audit back through `ListAuditForEntity`. I-1 is fixed: `ResolveCanonicalEmailIdentity` is shared by authentication and user administration, exact legacy digest reindexing is parameterized and conflict-safe, and both service and PostgreSQL tests cover mixed-case legacy lookup and collision refusal. O-1 is fixed for the checked source surface: decoded hash length is bounded to 16–64 bytes, the conversion has a narrow bound-backed suppression, and focused `gosec` reports zero issues.

I-2 is fixed: `NewPasswordHasher` rejects key lengths above `maxStoredPasswordHashBytes`, the 64-byte boundary generates a credential accepted by `VerifyPassword` and `IsUsablePasswordCredential`, and the 65-byte boundary is rejected before generation.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Dedicated operator-only CLI for local, deployed development, and production | CLI, composition, and operator-guide inspection | PASS | `backend/cmd/admin-bootstrap/main.go`; `docs/operations/administrator-bootstrap.md` |
| 2 | No public role-promotion endpoint | Router/OpenAPI search | PASS | No public bootstrap or promotion route found |
| 3 | Canonical case-insensitive email normalization | Normalizer and persisted mixed-case tests | PASS | `security.normalizeEmail`; registration, login, OAuth, and bootstrap canonicalize before digest/encryption |
| 4 | Legacy mixed-case digest can be reindexed safely | Isolated PostgreSQL mixed-case and collision tests | PASS | `TestPostgresAdministratorBootstrapReindexesLegacyEmailDigest`; auth and CLI reindex tests |
| 5 | Registration, login, OAuth, and bootstrap share the lookup contract | Caller inspection and regression tests | PASS | `auth/service.go`, `auth/oauth.go`, `adminbootstrap/service.go`; canonical and collision branches present |
| 6 | User administration preserves the same legacy lookup contract | User-admin legacy and collision behavior | PASS | Shared `ResolveCanonicalEmailIdentity`, service mixed-case/collision tests, repository reindex and unique-index collision tests |
| 7 | Optional exact UUID selector and selector exclusivity | CLI/service/repository tests | PASS | UUID path is separate and email plus UUID is rejected |
| 8 | Existing target is verified | PostgreSQL rejection test and SQL inspection | PASS | `email_verified` is read-only eligibility input |
| 9 | Real usable password credential | Authentication parser and malformed fixture coverage | PASS | `IsUsablePasswordCredential` delegates to `parseStoredPasswordCredential`; malformed params, hash/salt Base64, short values, and oversized hashes are rejected; generated credentials remain verifiable |
| 10 | Structurally complete encrypted OAuth credential | PostgreSQL OAuth-only eligibility test | PASS | OAuth envelope existence is checked without exposing plaintext |
| 11 | Only first administrator can be created | Lifecycle and existing-admin tests | PASS | First create succeeds; different administrator is refused |
| 12 | Same-target replay is a no-op | Replay result and audit-count tests | PASS | Replay does not update role or create an audit; `AuditID` is nil |
| 13 | Replay audit ID is truthful and nullable | Result type, CLI output, and replay assertion | PASS | `AdministratorBootstrapResult.AuditID *uuid.UUID`; CLI prints `none` |
| 14 | Concurrent attempts serialize | Isolated PostgreSQL concurrent test under race detector | PASS | One promotion and one audit, one replay |
| 15 | Role and audit persistence are atomic | Forced audit-failure rollback test | PASS | Role remains unchanged when audit insert fails |
| 16 | Truthful operator-origin audit is persisted and readable | Migration, insert, and shared list-reader integration | PASS | Bootstrap PostgreSQL lifecycle reads `actor_kind=operator`, nil `AdminUserID`, action, entity, and request ID through `ListAuditForEntity` |
| 17 | Explicit configured/target environment match | Service and CLI denial tests | PASS | Mismatch is rejected before repository work |
| 18 | Production requires explicit confirmation | Production negative test | PASS | Separate confirmation is required |
| 19 | CLI output contains safe metadata only | Captured stdout/stderr adversarial test | PASS | No supplied email, credentials, URL, key material, hashes, salts, tokens, or cookies |
| 20 | Interactive email input avoids shell history | Bounded stdin test | PASS | Email can be read from stdin without echo |
| 21 | Email verification is not mutated | Promotion SQL inspection | PASS | Promotion updates role and timestamp only |
| 22 | Unusable seeded administrator is removed | Seed SQL and seed test | PASS | Development seed removes the old administrator and its fixture audit |
| 23 | Existing sessions retain old claims | JWT/session regression test and docs | PASS | Fresh role requires sign-out/sign-in |
| 24 | Target-not-found and unverified cases are covered | Isolated PostgreSQL tests | PASS | Closed repository errors are asserted |
| 25 | Audit persistence rollback is covered | Triggered audit failure test | PASS | Transaction rollback is asserted |
| 26 | Design and operator documentation define semantics | Design and operator documents | PASS | Authorization, actor, environment, replay, privacy, and reauthentication are documented |
| 27 | Installed `golang-security` review is clean | Manual review plus focused source scan | PASS | Focused source `gosec` scan covers 41 files and reports zero issues; the broader package scan's unrelated pre-existing `httpapi/csrf.go` G101 is excluded from the task result |
| 28 | Focused race and full backend tests pass | Race, full suite, vet, vulnerability, and quick checks | PASS | All listed functional/static gates pass |

## 5. Changed-Symbol Inventory

The inventory intentionally includes each production declaration/function, embedded SQL/migration unit, changed test helper/test, and impacted dependency unit separately. No symbol is collapsed into a file-level row. The audit rows in Section 6 have the same count and order.

| # | Symbol/unit | Kind | File:line | Change | Callers/consumers | Tests or evidence |
|---:|---|---|---|---|---|---|
| 1 | `main` | process entry | `cmd/admin-bootstrap/main.go:23` | added | OS process | process boundary |
| 2 | `bootstrapExecutor` | function type | `main.go:29` | added | `run` | injected CLI tests |
| 3 | `run` | CLI function | `main.go:33` | added/modified | `main` | CLI table tests |
| 4 | `executeBootstrap` | composition function | `main.go:90` | added/modified | `run` | composition tests |
| 5 | `safeBootstrapError` | error mapper | `main.go:114` | added/modified | `run` | closed-mapper test |
| 6 | `lookupKeyLoader` | type | `main.go:131` | added | service composition | config tests |
| 7 | `newLookupKeyLoader` | constructor | `main.go:137` | added | `executeBootstrap` | key configuration test |
| 8 | `lookupKeyLoader.ActiveLookupKey` | method | `main.go:150` | added | digest service | config test |
| 9 | `lookupKeyLoader.LookupKey` | method | `main.go:156` | added | digest service | config test |
| 10 | `adminbootstrap.Request` | type | `service.go:17` | added/modified | `Bootstrap` | service tests |
| 11 | `adminbootstrap.Service` | type | `service.go:28` | added | CLI | service tests |
| 12 | `adminbootstrap.NewService` | constructor | `service.go:35` | added | CLI | service tests |
| 13 | `adminbootstrap.Service.Bootstrap` | method | `service.go:41` | added/modified | CLI | environment, digest, selector tests |
| 14 | `auth.OAuthIdentityStore` | interface | `auth/oauth.go` | modified | OAuth service | compile and OAuth tests |
| 15 | `auth.EncryptedIdentityRepository` | interface | `auth/service.go` | modified | auth services | compile and auth tests |
| 16 | `emailDigestIdentityRepository` | interface | `auth/service.go` | added | lookup helper | auth fixtures |
| 17 | `CoreAuthService.emailLookupDigests` | method | `auth/service.go` | added | login/reset/OAuth | mixed-case tests |
| 18 | `lookupAndReindexUserByEmail` | function | `auth/service.go` | added | login/reset/OAuth | reindex/collision tests |
| 19 | `CoreAuthService.Login` | method | `auth/service.go` | modified | auth callers | auth suite |
| 20 | `CoreAuthService.RequestPasswordReset` | method | `auth/service.go` | modified | reset caller | auth suite |
| 21 | `CoreAuthService.CompleteOAuth` | method | `auth/oauth.go` | modified | OAuth callback | OAuth suite |
| 22 | `normalizeOAuthProfile` | function | `auth/oauth.go` | modified consumer | OAuth service | canonical OAuth test |
| 23 | `PasswordHasher.VerifyPassword` | method | `auth/password.go:77` | modified | login | password tests |
| 24 | `IsUsablePasswordCredential` | function | `auth/password.go:88` | added | bootstrap repository | malformed credential test |
| 25 | `parseStoredPasswordCredential` | function | `auth/password.go:95` | added/modified | verify and eligibility | parser matrix |
| 26 | `parseEncodedHash` | function | `auth/password.go:109` | modified | shared parser | hash fixtures |
| 27 | `parseHashParams` | function | `auth/password.go:128` | modified | hash parser | malformed parameter tests |
| 28 | `security.normalizeEmail` | function | `security/normalizer.go` | modified | `NormalizeInput` | mixed-case normalization test |
| 29 | `PasswordCredentialValidator` | interface | `repository/types.go` | added | bootstrap repository | constructor validation |
| 30 | `AdministratorBootstrapSelector` | type | `repository/types.go:548` | added/modified | service/repository | selector tests |
| 31 | `AdministratorBootstrapResult` | type | `repository/types.go:555` | added/modified | CLI/service/repository | replay assertions |
| 32 | `AdministratorBootstrapRepository` | interface | `repository/types.go:564` | added | service/repository | compile boundary |
| 33 | `ErrAdministratorBootstrapTargetNotFound` | sentinel | `repository/errors.go` | added | mapper/repository | target test |
| 34 | `ErrAdministratorBootstrapUnverified` | sentinel | `repository/errors.go` | added | mapper/repository | verification test |
| 35 | `ErrAdministratorBootstrapNoCredential` | sentinel | `repository/errors.go` | added | mapper/repository | credential test |
| 36 | `ErrAdministratorAlreadyExists` | sentinel | `repository/errors.go` | added | mapper/repository | existing-admin test |
| 37 | `ErrCanonicalEmailCollision` | sentinel | `repository/errors.go` | added/modified | auth/bootstrap | collision tests |
| 38 | `PostgresAdministratorBootstrapRepository` | type | `admin_bootstrap_repository.go:51` | added/modified | CLI/tests | repository tests |
| 39 | `NewPostgresAdministratorBootstrapRepository` | constructor | `admin_bootstrap_repository.go:60` | added/modified | CLI | constructor tests |
| 40 | `BootstrapAdministrator` | transaction method | `admin_bootstrap_repository.go:66` | added/modified | service | lifecycle/race/rollback |
| 41 | `administratorBootstrapTarget` | projection | `admin_bootstrap_repository.go:112` | added/modified | loader/transaction | DB fixtures |
| 42 | `loadAdministratorBootstrapTarget` | loader | `admin_bootstrap_repository.go:121` | added/modified | transaction | selector/reindex tests |
| 43 | `scanAdministratorBootstrapTarget` | scanner | `admin_bootstrap_repository.go` | added | loader | credential matrix |
| 44 | `validateAdministratorBootstrapSelector` | validator | `admin_bootstrap_repository.go:140` | added/modified | transaction | validation table |
| 45 | `adminBootstrapLockSQL` | embedded SQL | `sql/admin_bootstrap_lock.sql` | added | transaction | concurrency |
| 46 | `adminBootstrapTargetByDigestSQL` | embedded SQL | `sql/admin_bootstrap_target_by_digest.sql` | added/modified | loader | mixed-case/credential DB tests |
| 47 | `adminBootstrapTargetByIDSQL` | embedded SQL | `sql/admin_bootstrap_target_by_id.sql` | added/modified | loader | UUID/credential DB tests |
| 48 | `adminBootstrapExistingSQL` | embedded SQL | `sql/admin_bootstrap_existing.sql` | added | invariant check | lifecycle/race |
| 49 | `adminBootstrapPromoteSQL` | embedded SQL | `sql/admin_bootstrap_promote.sql` | added | role mutation | verification immutability |
| 50 | `adminBootstrapAuditSQL` | embedded SQL | `sql/admin_bootstrap_audit.sql` | added | operator audit | audit/rollback |
| 51 | migration `000029` up | migration unit | `database/migrations/000029_admin_bootstrap_actor.up.sql` | added | migration runner | PostgreSQL reset |
| 52 | migration `000029` down | migration unit | `database/migrations/000029_admin_bootstrap_actor.down.sql` | added | migration runner | downgrade inspection |
| 53 | development administrator removal | seed unit | `backend/internal/seed/development.sql:140` | modified | seed runner | seed test |
| 54 | `TestRunReadsEmailInteractivelyAndEmitsOnlySafeMetadata` | test | `main_test.go:21` | added | `run` | self |
| 55 | `TestRunSupportsUUIDAndSafeFailures` | test | `main_test.go:47` | added | `run` | self |
| 56 | `TestRunRejectsMissingOrMalformedArguments` | test | `main_test.go:79` | added | `run` | self |
| 57 | `TestSafeBootstrapErrorsAndLookupKeyConfiguration` | test | `main_test.go:96` | added | mapper/key loader | self |
| 58 | `TestExecuteBootstrapReturnsOnlySafeCompositionFailures` | test | `main_test.go:131` | added | composition | self |
| 59 | `TestExecuteBootstrapComposesSuccessfulRuntime` | integration test | `main_test.go:150` | added | composition | self |
| 60 | `TestExecuteBootstrapReindexesLegacyMixedCaseEmail` | integration test | `main_test.go` | added | composition | PostgreSQL |
| 61 | `TestExecuteBootstrapRefusesCanonicalEmailCollision` | integration test | `main_test.go` | added | composition | PostgreSQL |
| 62 | `TestExecuteBootstrapRejectsMalformedPasswordCredential` | integration test | `main_test.go` | added | credential validator | PostgreSQL matrix |
| 63 | `ioDiscard` | test type | `main_test.go:195` | added | argument tests | self |
| 64 | `ioDiscard.Write` | test method | `main_test.go:197` | added | `ioDiscard` | self |
| 65 | `testKeys` | fixture type | `adminbootstrap/service_test.go` | added | service tests | self |
| 66 | `testKeys.ActiveLookupKey` | fixture method | `service_test.go` | added | digest service | self |
| 67 | `testKeys.LookupKey` | fixture method | `service_test.go` | added | digest service | self |
| 68 | `recordingRepository` | fixture type | `service_test.go` | added | service tests | self |
| 69 | `recordingRepository.BootstrapAdministrator` | fixture method | `service_test.go` | added | `Bootstrap` | self |
| 70 | `TestBootstrapValidatesEnvironmentAndNormalizesEmail` | test | `service_test.go:34` | added | service | self |
| 71 | `TestBootstrapSupportsOnlyOneUUIDSelector` | test | `service_test.go:59` | added | service | self |
| 72 | `failingKeys` | fixture type | `service_test.go` | added | error test | self |
| 73 | `failingKeys.ActiveLookupKey` | fixture method | `service_test.go` | added | digest service | self |
| 74 | `failingKeys.LookupKey` | fixture method | `service_test.go` | added | digest service | self |
| 75 | `TestBootstrapHidesLookupFailureAndPropagatesRepositoryOutcome` | test | `service_test.go:83` | added | service | self |
| 76 | `bootstrapSelector` | DB test helper | `admin_bootstrap_repository_test.go:17` | added | repository tests | self |
| 77 | `TestPostgresAdministratorBootstrapLifecycle` | integration test | `admin_bootstrap_repository_test.go:23` | added | repository | self |
| 78 | `TestPostgresAdministratorBootstrapRejectsInvalidTargets` | integration test | `admin_bootstrap_repository_test.go:66` | added | repository | self |
| 79 | `TestPostgresAdministratorBootstrapAcceptsEncryptedOAuthCredential` | integration test | `admin_bootstrap_repository_test.go:96` | added | repository | self |
| 80 | `TestPostgresAdministratorBootstrapSerializesConcurrentAttempts` | integration test | `admin_bootstrap_repository_test.go:120` | added | advisory lock | race run |
| 81 | `TestPostgresAdministratorBootstrapRollsBackWhenAuditFails` | integration test | `admin_bootstrap_repository_test.go:160` | added | transaction | self |
| 82 | `TestPostgresAdministratorBootstrapValidationAndDatabaseFailures` | test | `admin_bootstrap_repository_test.go:185` | added | error mapping | self |
| 83 | `TestPostgresAdministratorBootstrapReindexesLegacyEmailDigest` | integration test | `admin_bootstrap_repository_test.go` | added | loader | PostgreSQL |
| 84 | `TestPostgresAdministratorBootstrapRejectsCanonicalEmailCollision` | integration test | `admin_bootstrap_repository_test.go` | added | loader | PostgreSQL |
| 85 | `createBootstrapUserOnDB` | DB fixture helper | `admin_bootstrap_repository_test.go:227` | added | integration tests | self |
| 86 | `bootstrapTestHash` | fixture helper | `admin_bootstrap_repository_test.go` | added | credential tests | self |
| 87 | `bootstrapRequestID` | fixture helper | `admin_bootstrap_repository_test.go` | added | request tests | self |
| 88 | `validateBootstrapTestCredential` | fixture validator | `admin_bootstrap_repository_test.go` | added | repository constructor | self |
| 89 | `TestRunIsIdempotentAndSeedsRepositoryFixtures` | seed integration test | `seed_test.go:74` | modified | seed runner | self |
| 90 | `TestAdministratorBootstrapDoesNotChangeExistingSessionClaims` | auth regression test | `token_test.go:69` | added | token service | self |
| 91 | `TestIsUsablePasswordCredentialMatchesAuthenticationParser` | parser test | `password_test.go` | added | eligibility | malformed matrix |
| 92 | `TestPasswordHasherHashesAndVerifies` | password test | `password_test.go` | modified | parser | self |
| 93 | `TestCoreAuthServiceReindexesLegacyMixedCaseEmailWithoutMergingCollision` | auth integration test | `service_test.go` | added | lookup helper | PostgreSQL |
| 94 | registration canonical persistence assertions | auth test unit | `registration_test.go` | modified | registration | self |
| 95 | OAuth canonical persistence assertions | auth test unit | `oauth_coverage_test.go` | modified | OAuth | self |
| 96 | `oauthUnsupportedIdentityRepository.ReindexUserEmailDigest` | fixture method | `oauth_coverage_test.go` | added | OAuth interface | self |
| 97 | `memoryIdentityRepository.ReindexUserEmailDigest` | fixture method | `service_test.go` | added | auth interface | self |
| 98 | `fakeRegistrationRepository.CreateUserWithConsent` | fixture method | `registration_test.go` | modified | registration interface | self |
| 99 | `TestNormalizeInput` | security test | `security_test.go` | modified | normalizer | mixed-case assertion |
| 100 | `TestLookupExactEmailNormalizesDigestAndAuditsEntity` | user-admin test | `useradmin/service_test.go` | modified | user administration | canonical assertion |
| 101 | `repository.AdminAuditEntry` | impacted audit type | `repository/types.go:525` | pre-existing dependency | list/persist readers | no operator-list test |
| 102 | `adminAuditListForEntitySQL` | impacted SQL | `sql/admin_audit_list_for_entity.sql` | modified | `ListAuditForEntity` | operator readback test |
| 103 | `ListAuditForEntity` | impacted reader | `compliance_repository.go:427` | modified | audit consumers | bootstrap readback |
| 104 | `scanAdminAuditEntry` | impacted scanner | `compliance_repository.go:482` | modified | `ListAuditForEntity` | nullable actor test |
| 105 | `validateAdminAuditEntry` | impacted validator | `compliance_repository.go:540` | modified | shared audit insert | actor invariant tests |
| 106 | `adminAuditInsertSQL` | impacted SQL | `sql/admin_audit_insert.sql` | modified | shared audit persistence | administrator tests |
| 107 | `useradmin.Service.lookupRequest` | impacted lookup | `useradmin/service.go:199` | modified | user-admin `Lookup` | canonical/legacy tests |
| 108 | `LookupAdminUsers` | impacted repository method | `admin_user_repository.go:49` | dependency consumer | resolver adapter | PostgreSQL lookup tests |
| 109 | `adminUserGetByDigestSQL` | impacted SQL | `sql/admin_user_get_by_digest.sql` | dependency primitive | resolver adapter | exact digest tests |
| 110 | `AdminAuditActorKind` | behavioral type | `repository/types.go:525` | added | audit model/validation | audit tests |
| 111 | `AdminAuditActorAdministrator` | actor constant | `repository/types.go:530` | added | administrator callers | audit tests |
| 112 | `AdminAuditActorOperator` | actor constant | `repository/types.go:532` | added | bootstrap audit/readback | lifecycle test |
| 113 | `ResolveCanonicalEmailIdentity` | generic resolver | `canonical_email_resolver.go:11` | added | auth and user-admin | resolver unit matrix |
| 114 | `PostgresAdminUserRepository.ReindexUserEmailDigest` | repository method | `admin_user_repository.go:80` | added | user-admin resolver | PostgreSQL reindex/collision test |
| 115 | `useradmin.Service.Lookup` | service method | `useradmin/service.go:114` | modified | user-admin controller | mixed-case/collision tests |
| 116 | `useradmin.Service.lookupUserByEmailDigest` | adapter method | `useradmin/service.go:240` | added | canonical resolver | user-admin service tests |
| 117 | `AdminUserRepository` | interface | `repository/types.go:978` | modified | user-admin service/repository | compile contract |
| 118 | `httpapi.AdminController.transactionalMutation` | handler | `httpapi/admin_controller.go:227` | modified | admin routes | HTTP audit assertions |
| 119 | `dataimporter.confirm` | integration helper | `dataimporter/integration_test.go:370` | modified | importer tests | compile/integration suite |
| 120 | `TestResolveCanonicalEmailIdentity` | unit test | `canonical_email_resolver_test.go:17` | added | resolver | branch matrix |
| 121 | `TestPostgresAdminUserReindexesLegacyDigestAndRejectsCollision` | integration test | `admin_user_repository_test.go:58` | added | user-admin repository | PostgreSQL |
| 122 | `TestLookupExactEmailReindexesLegacyMixedCaseDigest` | service test | `useradmin/service_test.go:160` | added | user-admin service | legacy reindex |
| 123 | `TestLookupExactEmailRefusesCanonicalLegacyCollision` | service test | `useradmin/service_test.go:177` | added | user-admin service | collision refusal |
| 124 | `memoryAdminUsers.LookupAdminUsers` | test method | `useradmin/service_test.go:34` | modified | resolver adapter | service tests |
| 125 | `memoryAdminUsers.ReindexUserEmailDigest` | test method | `useradmin/service_test.go:42` | added | resolver adapter | service tests |
| 126 | `recordingDigester.DigestForWrite` | test method | `useradmin/service_test.go:80` | modified | lookupRequest | canonical/legacy assertions |
| 127 | `TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData` | audit security test | `admin_audit_security_test.go:14` | modified | audit validator | safety suite |
| 128 | `TestPostgresComplianceAndAdminRepositories` | integration test | `postgres_repository_test.go:2261` | modified | audit read/write | administrator actor readback |
| 129 | `TestPostgresComplianceAndAdminRepositoryValidationAndErrors` | test | `postgres_repository_test.go:2594` | modified | audit validation | pointer/actor matrix |
| 130 | `TestAdminAuditSnapshotValidationRollsBackTransaction` | audit test | `admin_audit_security_test.go:99` | modified | `WithMutationAudit` | rollback |
| 131 | `TestAdminAuditPersistenceErrorPreservesCause` | audit test | `admin_audit_security_test.go:115` | modified | persistence | error propagation |
| 132 | `TestAdminMutationAuditSuccessfulCommitPath` | audit test | `admin_audit_security_test.go:130` | modified | persistence | commit path |
| 133 | `TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit` | audit test | `admin_audit_security_test.go:144` | modified | replay path | no duplicate |
| 134 | `TestAdminMutationControlOrderAtomicAuditAndSanitizedEnvelopes` | HTTP test | `admin_controller_test.go:102` | modified | transactional handler | administrator actor assertion |
| 135 | `TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit` | HTTP test | `user_admin_controller_test.go:126` | modified | user-admin controller | administrator actor assertion |
| 136 | `maxStoredPasswordHashBytes` | bound constant | `auth/password.go:30` | added | parser/hasher contract | oversized hash test |
| 137 | `TestIsUsablePasswordCredentialMatchesAuthenticationParser` | parser test | `auth/password_test.go:66` | modified | credential eligibility | oversized hash case |
| 138 | `NewPasswordHasher` | impacted constructor | `auth/password.go:47` | dependency caller | `HashPassword` | missing overbound configuration test |
| 139 | `PasswordHasher.HashPassword` | impacted producer | `auth/password.go:66` | dependency caller | registration/reset | missing overbound configuration test |
| 140 | `TestPasswordHasherKeyLengthMatchesStoredParserBoundary` | password boundary test | `auth/password_test.go:112` | added | `NewPasswordHasher`, `HashPassword`, `VerifyPassword`, parser | 64-byte generation/verification and 65-byte rejection |

```yaml
inventory_source_count: 140
audited_symbol_count: 140
inventory_complete: true
generated_groupings:
  - "None. Every inventory row has one corresponding audit row; impacted dependencies are explicitly included rather than hidden behind changed-file rows."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/concurrency | Security boundary | I/O/performance | API/idioms | Tests/gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `main` | Delegates to `run` and exits with its status. | Process-only boundary. | OS lifetime only. | No direct input handling. | One call. | Minimal entry point. | Wrapper not directly callable. | PASS |
| `bootstrapExecutor` | Injectable runtime signature. | Type only. | Context is explicit. | Keeps composition isolated. | No allocation. | Narrow seam. | Used by CLI tests. | PASS |
| `run` | Parses one selector and safe flags. | Handles stdin EOF, bad UUID, missing values, replay, and closed errors. | Bounded scanner; no shared mutable state. | Does not print selector or internal errors. | One bounded input path. | Clear CLI boundary. | Captured-output matrix. | PASS |
| `executeBootstrap` | Composes configuration, DB, key, service, and repository. | Config/key/DB failures are closed. | Defers DB close; context reaches service. | Key and URL details remain hidden. | One DB connection. | Direct composition. | Success and failure tests. | PASS |
| `safeBootstrapError` | Maps only approved operator-safe outcomes. | Wrapped sentinels and unknown errors handled. | Pure function. | Prevents PII and secret leakage. | Constant strings. | Idiomatic `errors.Is`. | Exhaustive mapper test. | PASS |
| `lookupKeyLoader` | Holds validated lookup key material. | Type has no error path. | Process-scoped immutable bytes by convention. | Production key is required and never emitted. | Small fixed key. | Matches existing digest interface. | Configuration tests. | PASS |
| `newLookupKeyLoader` | Requires supported version and sufficient key length. | Missing and short production keys denied. | No resource lifetime. | No key disclosure. | Setup only. | Small constructor. | Dev/prod tests; unknown version branch limited. | PASS |
| `lookupKeyLoader.ActiveLookupKey` | Returns active version and key. | Validated constructor makes success expected. | Read-only. | Trusted digest boundary. | Memory only. | Interface-compatible. | Config test. | PASS |
| `lookupKeyLoader.LookupKey` | Resolves configured version. | Unsupported version behavior is not exercised. | Read-only. | Avoids accidental key fallback. | Memory only. | Interface-compatible. | Unknown-version test gap is non-blocking. | PASS |
| `Request` | Carries explicit environment, confirmation, selector, and request ID. | Value validation is delegated to service. | Context separate from request. | Email is transient and not output. | Small value. | Clear service boundary. | Service tests. | PASS |
| `Service` | Holds repository and key dependencies. | Nil dependencies fail at call boundaries. | No shared mutable state. | Internal operator-only package. | Two references. | Narrow service. | Constructor/use tests. | PASS |
| `NewService` | Constructs service with required dependencies. | No resource error. | No acquisition. | Does not expose HTTP. | One allocation. | Idiomatic. | Indirect coverage. | PASS |
| `Service.Bootstrap` | Validates environment and exactly one selector, then delegates. | Email normalization, digest lookup, UUID, production confirmation, and key failures handled. | Context propagated; repository owns transaction. | Canonical email and optional legacy digest are pseudonymous. | At most two digest calculations. | Simple orchestration. | Mixed-case, collision, and selector tests. | PASS |
| `OAuthIdentityStore` | Requires provider lookup/link/create and digest reindex operations. | Interface contract only. | Context-aware methods. | OAuth path remains internal. | No direct I/O. | Explicit migration seam. | OAuth fixtures compile. | PASS |
| `EncryptedIdentityRepository` | Supports canonical lookup and digest reindex. | Reindex conflict is mapped fail-closed. | Repository update is atomic per row. | HMAC values only. | Indexed update. | Interface extension is used. | Auth fixtures updated. | PASS |
| `emailDigestIdentityRepository` | Narrows lookup helper to lookup and reindex operations. | Missing legacy result remains normal. | Context passed to both queries. | Avoids plaintext identity. | Two bounded queries. | Small internal interface. | Memory fixture. | PASS |
| `CoreAuthService.emailLookupDigests` | Produces canonical digest and exact trimmed legacy digest only when different. | Invalid email or key error stops lookup. | No mutation. | Never logs email or digest. | At most two HMAC calls. | Reusable helper. | Mixed-case auth tests. | PASS |
| `lookupAndReindexUserByEmail` | Resolves canonical first, then exact legacy; refuses different IDs. | Not-found, reindex conflict, and canonical collision are closed. | Reindex is guarded by unique index; no merge. | Prevents identity confusion. | Two indexed queries plus rare update. | Shared auth helper. | Auth mixed-case/collision test. | PASS |
| `CoreAuthService.Login` | Uses canonical and legacy lookup before password verification. | Existing legacy rows are reindexed; collision fails. | Lockout/auth flow remains context-bound. | No email output. | Extra lookup only during migration. | Uses shared helper. | Full auth suite. | PASS |
| `CoreAuthService.RequestPasswordReset` | Uses same identity resolver as login. | Missing and collision paths remain closed. | No state before target resolution. | Avoids reset to ambiguous account. | Bounded lookup. | Shared path prevents drift. | Full auth suite; direct legacy reset assertion limited. | PASS |
| `CoreAuthService.CompleteOAuth` | Normalizes profile and matches existing identity by canonical plus legacy digest. | Existing link, collision, and new-user branches are distinct. | Reindex occurs before link decision. | No account merge on collision. | One or two indexed lookups. | Reuses helper. | OAuth suite and interface fixtures. | PASS |
| `normalizeOAuthProfile` | Applies typed email normalization to provider profile. | Invalid provider email is rejected. | Pure profile value. | Prevents case-based duplicate identities. | One normalization. | Existing profile boundary. | Canonical OAuth assertion. | PASS |
| `PasswordHasher.VerifyPassword` | Accepts only the parser-approved credential format. | Malformed hash, salt, parameters, and weak cost return false. | Argon2 work is bounded by parsed stored values and the decoded hash bound. | Constant-time comparison after parse. | Argon2 cost dominates. | Shared parser is correct. | Password and malformed matrix. | PASS |
| `IsUsablePasswordCredential` | Exactly reuses authentication parser and minimum parameters without plaintext. | Every parser failure returns false. | Pure, no DB. | No secret material in result/error. | Parse plus Base64 only. | Correct eligibility seam. | All requested malformed cases. | PASS |
| `parseStoredPasswordCredential` | Requires Argon2id v19, nonzero parameters, minimum memory, hash and salt lengths. | Invalid encoding, duplicate/unknown/missing params, Base64, and short values fail. | Pure parser. | Rejects malformed stored material before promotion. | Bounded by stored text; hash length bound should be documented for G115. | Single source of truth. | Parser matrix. | PASS |
| `parseEncodedHash` | Parses exact four-part format and records hash length. | Bad algorithm/version/encoding/short hash fail. | Pure. | No secret in errors. | Base64 allocation. | Simple parser. | Invalid hash and short hash tests. | PASS |
| `parseHashParams` | Accepts each required parameter once with valid nonzero values and p <= 255. | Duplicate, unknown, missing, zero, malformed, and overflow values fail. | Pure. | Prevents cost/parser ambiguity. | Small map allocation. | Readable validation. | Adversarial parameter tests. | PASS |
| `security.normalizeEmail` | Trims, validates, and lowercases canonical email. | Invalid address fails; changed flag records normalization. | Pure. | Establishes identity boundary. | One parse and lowercase. | Shared typed normalizer. | Mixed-case security test. | PASS |
| `PasswordCredentialValidator` | Repository cannot silently select its own weaker policy. | Nil validator rejected at construction/call. | Called while target row is locked. | Credential decision is shared with auth. | One parser invocation. | Dependency inversion is appropriate. | Constructor and DB tests. | PASS |
| `AdministratorBootstrapSelector` | Exactly one UUID or digest selector. | Nil/empty/both invalid. | Value passed into transaction. | Digest is pseudonymous. | Small. | Clear API. | Service/repository tests. | PASS |
| `AdministratorBootstrapResult` | Reports target, created/replayed state, and optional audit ID. | Replay has nil audit ID, not zero UUID. | Immutable result. | No PII. | Small. | Truthful nullable API. | Replay/CLI tests. | PASS |
| `AdministratorBootstrapRepository` | Exposes one atomic operation only. | Interface type. | Context boundary explicit. | No public route. | No I/O itself. | Minimal contract. | Compile/use tests. | PASS |
| `ErrAdministratorBootstrapTargetNotFound` | Closed target resolution failure. | No row maps here. | No state. | Does not reveal identity. | Constant. | Sentinel. | DB/CLI tests. | PASS |
| `ErrAdministratorBootstrapUnverified` | Closed verification denial. | Verified false maps here. | No mutation. | Does not disclose account details. | Constant. | Sentinel. | DB test. | PASS |
| `ErrAdministratorBootstrapNoCredential` | Closed real-credential denial. | No OAuth and parser false maps here. | No mutation. | Does not expose hash/salt. | Constant. | Sentinel. | DB and malformed tests. | PASS |
| `ErrAdministratorAlreadyExists` | Closed refusal for different administrator. | Same target is replay, different target denied. | Advisory lock protects decision. | Avoids account identity disclosure. | One bounded query. | Sentinel. | Lifecycle/race tests. | PASS |
| `ErrCanonicalEmailCollision` | Closed ambiguity failure for two identities. | Canonical and legacy IDs differ, no reindex. | Unique index remains intact. | Prevents account merge/auth confusion. | Two indexed queries. | Shared error contract. | Auth/bootstrap collision tests. | PASS |
| `PostgresAdministratorBootstrapRepository` | Owns DB executor, validator, and transaction setup. | Nil DB/validator denied. | Transaction is local to operation. | SQL is embedded and parameterized. | One dependency object. | Correct repository placement. | Constructor/error tests. | PASS |
| `NewPostgresAdministratorBootstrapRepository` | Binds DB and parser validator. | Invalid dependencies fail closed later or at construction. | No DB acquisition. | Parser cannot be bypassed. | One allocation. | Idiomatic. | Repository tests. | PASS |
| `BootstrapAdministrator` | Advisory lock, target lock, eligibility, admin invariant, role update, audit insert, commit. | All failures roll back; replay returns nil audit ID. | Transaction and lock serialize attempts. | Role and truthful operator audit are atomic. | Indexed reads plus one update/insert. | Clear transaction sequence. | Lifecycle/race/rollback plus shared operator audit readback. | PASS |
| `administratorBootstrapTarget` | Holds locked target and credential material for validator. | Nullable password fields and OAuth boolean are represented. | Row is locked before decision. | No plaintext email. | Small projection. | Minimal. | PostgreSQL matrix. | PASS |
| `loadAdministratorBootstrapTarget` | Resolves exact UUID or digest and reindexes legacy within transaction. | Canonical/legacy not-found, collision, reindex conflict, and scan errors handled. | Same transaction/row lock. | No merge and no verification mutation. | Indexed lookups. | Correct placement. | Mixed-case and collision DB tests. | PASS |
| `scanAdministratorBootstrapTarget` | Scans nullable password fields and OAuth existence without parsing in SQL. | NULL pairs and scan errors handled. | No state. | Credential bytes stay in process only. | One row scan. | Parser is injected separately. | Malformed credential tests. | PASS |
| `validateAdministratorBootstrapSelector` | Requires one nonzero UUID or nonempty digest. | Invalid request returns validation error. | Pure. | Boundary prevents broad selection. | Constant work. | Idiomatic. | Table-driven validation test. | PASS |
| `adminBootstrapLockSQL` | Serializes bootstrap attempts. | DB lock error aborts. | Transaction-scoped advisory lock. | Prevents race-based second admin. | One constant lock. | Correct primitive. | Concurrent PostgreSQL test. | PASS |
| `adminBootstrapTargetByDigestSQL` | Selects locked email target and raw credential fields. | No row is target-not-found. | `FOR UPDATE`; legacy reindex is transaction-local. | Parameterized digest. | Indexed lookup plus OAuth `EXISTS`. | SQL colocated/embedded. | Mixed-case and malformed credential DB tests. | PASS |
| `adminBootstrapTargetByIDSQL` | Selects exact UUID target and raw credential fields. | No row is target-not-found. | `FOR UPDATE`. | UUID parameterized. | Primary-key lookup. | SQL colocated. | UUID and malformed credential tests. | PASS |
| `adminBootstrapExistingSQL` | Finds one existing administrator under transaction. | None means creation; same ID means replay. | Lock ordering is consistent. | No identity output. | `LIMIT 1`. | Straightforward. | Lifecycle/race. | PASS |
| `adminBootstrapPromoteSQL` | Updates only role and timestamp. | No row is unexpected failure. | Same transaction as audit. | Email verification stays unchanged. | One update. | Minimal. | Lifecycle/inspection. | PASS |
| `adminBootstrapAuditSQL` | Inserts operator actor with NULL admin user and returns metadata. | Constraint/insert failure rolls back. | Same transaction as role update. | Truthful actor at write time. | One insert. | Explicit fields. | Insert/rollback and shared reader readback. | PASS |
| migration `000029` up | Allows administrator or operator actor shapes. | DDL failure aborts migration. | Migration runner controls ordering. | Constraint prevents false admin attribution. | Small DDL. | Explicit schema transition. | PostgreSQL reset and psql NULL probe. | PASS |
| migration `000029` down | Removes operator rows before restoring old non-null constraint. | Downgrade is intentionally destructive to operator rows. | Schema rollback ordering is valid. | Old schema restored. | Small DDL. | Tradeoff documented. | No preservation test. | PASS |
| development administrator removal | Deletes unusable seeded administrator and related audit. | Idempotent keyed deletes. | Seed reruns deterministically. | Prevents known unusable account. | Two deletes. | Surgical. | Seed integration test. | PASS |
| `TestRunReadsEmailInteractivelyAndEmitsOnlySafeMetadata` | Proves stdin and safe output. | Successful result path. | Buffer-only. | Checks supplied PII and secret classes absent. | Small. | Strong adversarial CLI test. | No writer-error branch. | PASS |
| `TestRunSupportsUUIDAndSafeFailures` | Covers UUID selection and safe errors. | Bad UUID, both selectors, existing admin, unknown error. | Injected executor. | Sensitive internal strings excluded. | Table-driven. | Idiomatic. | Good negative coverage. | PASS |
| `TestRunRejectsMissingOrMalformedArguments` | Denies incomplete CLI input. | Missing env/value/positional/stdin. | No bootstrap invocation. | Parser text does not expose secrets. | Bounded scan. | Clear. | Oversized scanner path not explicit. | PASS |
| `TestSafeBootstrapErrorsAndLookupKeyConfiguration` | Covers closed mapper and key policy. | Development fallback and production key cases. | `t.Setenv` scoped. | No key output. | Small. | Good configuration test. | Unknown version gap is non-blocking. | PASS |
| `TestExecuteBootstrapReturnsOnlySafeCompositionFailures` | Covers generic config/key/DB composition failures. | Each failure is sanitized. | No successful DB mutation. | Internal details suppressed. | One attempt per case. | Useful boundary. | No canceled-context assertion. | PASS |
| `TestExecuteBootstrapComposesSuccessfulRuntime` | Exercises initial create and replay through real composition. | Both results safe. | Isolated DB. | No secret output. | Two calls. | Valuable integration. | Replay nil audit is asserted. | PASS |
| `TestExecuteBootstrapReindexesLegacyMixedCaseEmail` | Proves CLI path can migrate exact trimmed legacy case. | Legacy-only target succeeds. | Reindex and promotion share transaction. | Collision-safe path. | Two lookups/update. | Direct regression. | PostgreSQL-backed. | PASS |
| `TestExecuteBootstrapRefusesCanonicalEmailCollision` | Proves CLI fails closed on two identities. | No role or reindex mutation. | Transaction rolls back. | Prevents account merge. | Bounded. | Correct negative test. | PostgreSQL-backed. | PASS |
| `TestExecuteBootstrapRejectsMalformedPasswordCredential` | Proves CLI rejects parser-invalid material. | Params, hash/salt Base64, short hash/salt cases. | No promotion. | Test does not print material. | Parser cost only. | Adversarial coverage. | Valid generated credential also covered. | PASS |
| `ioDiscard` | Test writer implements `io.Writer`. | Type only. | No state. | Test-only. | No allocation. | Minimal. | Argument tests. | PASS |
| `ioDiscard.Write` | Returns full write count. | Always succeeds. | Pure. | Test-only. | Constant. | Idiomatic. | Argument tests. | PASS |
| `testKeys` | Deterministic service digest fixture. | Valid fixture only. | Immutable test bytes. | No production secrets. | Small. | Minimal. | Service tests. | PASS |
| `testKeys.ActiveLookupKey` | Returns fixture active key. | Always succeeds. | Pure. | Test-only. | Fixed. | Simple. | Service tests. | PASS |
| `testKeys.LookupKey` | Returns fixture key by version. | Version behavior is fixture-only. | Pure. | Test-only. | Fixed. | Interface seam. | Service tests. | PASS |
| `recordingRepository` | Records service selector and returns configured result. | Success/error branch. | Test-local mutable state. | No production boundary. | No I/O. | Small mock. | Service tests. | PASS |
| `recordingRepository.BootstrapAdministrator` | Implements repository seam. | Configured error propagated. | No transaction. | Test-only. | Constant. | Idiomatic fixture. | Service tests. | PASS |
| `TestBootstrapValidatesEnvironmentAndNormalizesEmail` | Covers environment and canonical digest construction. | Mismatch, production, invalid email, success. | Mock only. | No output. | HMAC fixture. | Focused. | Persisted mixed-case behavior covered at DB layer. | PASS |
| `TestBootstrapSupportsOnlyOneUUIDSelector` | Covers UUID alternative and exclusivity. | UUID success and both-selector denial. | Mock. | No email output. | Small. | Clear. | Service tests. | PASS |
| `failingKeys` | Failing digest fixture. | All key calls error. | No state. | Test-only. | Constant. | Minimal. | Failure test. | PASS |
| `failingKeys.ActiveLookupKey` | Fails active key lookup. | Error propagated. | Pure. | Test-only. | Constant. | Simple. | Failure test. | PASS |
| `failingKeys.LookupKey` | Fails versioned key lookup. | Error propagated. | Pure. | Test-only. | Constant. | Simple. | Failure test. | PASS |
| `TestBootstrapHidesLookupFailureAndPropagatesRepositoryOutcome` | Confirms closed lookup and repository errors. | Both dependency failures. | Mock. | Key detail suppressed. | Small. | Good mapping test. | No cancellation branch. | PASS |
| `bootstrapSelector` | Builds pseudonymous DB selector fixture. | Valid fixture. | Test-local. | Avoids PII. | Small. | Minimal. | All DB tests. | PASS |
| `TestPostgresAdministratorBootstrapLifecycle` | Covers create, audit, replay, and different-admin refusal. | Normal state transitions. | Real transaction. | Operator actor asserted on insert. | Small DB workload. | Strong lifecycle. | Nullable replay asserted. | PASS |
| `TestPostgresAdministratorBootstrapRejectsInvalidTargets` | Covers missing, unverified, no credential, and invalid target. | Closed errors. | Real DB. | No secret details. | Table-driven. | Good negative coverage. | Malformed cases are in separate test. | PASS |
| `TestPostgresAdministratorBootstrapAcceptsEncryptedOAuthCredential` | Allows verified OAuth Login Method without password. | OAuth-only target succeeds. | Real transaction. | Envelope projection only. | One identity. | Correct eligibility test. | Nonempty envelope semantics. | PASS |
| `TestPostgresAdministratorBootstrapSerializesConcurrentAttempts` | Proves one create and one replay. | Both calls complete safely. | Advisory lock under two goroutines. | One audit effect. | Two transactions. | Appropriate concurrency test. | Race command passed. | PASS |
| `TestPostgresAdministratorBootstrapRollsBackWhenAuditFails` | Proves role rollback on audit error. | Triggered insert failure. | Transaction abort. | Fail-closed mutation. | One transaction. | Strong rollback test. | Audit count remains controlled. | PASS |
| `TestPostgresAdministratorBootstrapValidationAndDatabaseFailures` | Covers validation, lock, target, existing, promote, audit, commit errors. | Error categories mapped. | Fake transaction paths. | No output. | Table-driven. | Broad failure coverage. | Exact database categories vary by fake. | PASS |
| `TestPostgresAdministratorBootstrapReindexesLegacyEmailDigest` | Proves transaction-local exact legacy reindex. | Legacy digest becomes canonical. | Row lock and unique index. | No merge. | Lookup plus update. | Direct regression. | PostgreSQL-backed. | PASS |
| `TestPostgresAdministratorBootstrapRejectsCanonicalEmailCollision` | Proves collision is rejected. | Neither account promoted. | Transaction remains safe. | Fail-closed identity boundary. | Two lookups. | Direct regression. | PostgreSQL-backed. | PASS |
| `createBootstrapUserOnDB` | Creates controlled target fixtures. | Verified, credential, and digest variants. | Isolated DB writes. | Test material only. | One insert. | Reusable helper. | Supports malformed matrix. | PASS |
| `bootstrapTestHash` | Produces valid parser-compatible fixture. | Valid generated hash path. | Test-only. | No plaintext output. | Argon2 test cost. | Central fixture. | Valid credential case. | PASS |
| `bootstrapRequestID` | Produces stable nonzero request UUID. | Test-only value. | No state. | No identity. | Constant. | Minimal helper. | Repository tests. | PASS |
| `validateBootstrapTestCredential` | Injects real parser validator into DB repository. | Invalid material denied. | Called in transaction. | Matches auth rules. | Parse only. | Correct test seam. | Parser matrix. | PASS |
| `TestRunIsIdempotentAndSeedsRepositoryFixtures` | Confirms seed reruns and no administrator. | Repeated seed is stable. | Isolated DB. | Removes unusable account. | Existing seed workload. | Appropriate seed regression. | Audit absence indirectly covered. | PASS |
| `TestAdministratorBootstrapDoesNotChangeExistingSessionClaims` | Old token retains old role; new token sees new role. | Reauthentication distinction. | JWT is immutable. | No session privilege escalation. | Small crypto test. | Focused. | Persistence/session integration not needed for CLI. | PASS |
| `TestIsUsablePasswordCredentialMatchesAuthenticationParser` | Eligibility and verification parser agree. | All malformed and valid cases. | Pure. | No credential disclosure. | Parser cost only. | Strong shared-contract test. | Requested matrix complete. | PASS |
| `TestPasswordHasherHashesAndVerifies` | Hash output remains accepted by new parser. | Valid and invalid passwords. | Random salt. | Constant-time verify. | Argon2 workload. | Regression-safe. | Existing password suite. | PASS |
| `TestCoreAuthServiceReindexesLegacyMixedCaseEmailWithoutMergingCollision` | Auth resolves exact legacy case and reindexes safely. | Canonical-only, legacy-only, collision branches. | Unique index protects update. | No account merge. | Two lookups rare path. | Shared resolver regression. | PostgreSQL-backed. | PASS |
| registration canonical persistence assertions | Registration encrypts/digests canonical email. | Mixed-case input accepted. | New row only. | Identity uniqueness consistent. | One HMAC/encryption. | Uses normalizer. | Registration tests. | PASS |
| OAuth canonical persistence assertions | New OAuth identity persists canonical email. | Mixed-case provider input. | New identity transaction. | Prevents duplicate identity. | One HMAC/encryption. | Uses normalizer. | OAuth tests. | PASS |
| `oauthUnsupportedIdentityRepository.ReindexUserEmailDigest` | Keeps OAuth test double aligned with interface. | Fixture method only. | No state beyond test. | Test-only. | No I/O. | Compile seam. | OAuth coverage. | PASS |
| `memoryIdentityRepository.ReindexUserEmailDigest` | Keeps auth memory repository able to migrate digest. | Updates fixture record. | Test-local mutation. | Test-only. | Constant. | Interface compatibility. | Auth tests. | PASS |
| `fakeRegistrationRepository.CreateUserWithConsent` | Keeps registration fake aligned with canonical contract. | Fixture success path. | Test-local. | Test-only. | No I/O. | Interface compatibility. | Registration tests. | PASS |
| `TestNormalizeInput` | Asserts lower-case canonical email and changed metadata. | Mixed-case and invalid cases. | Pure. | Shared identity boundary. | Small. | Focused. | Security tests. | PASS |
| `TestLookupExactEmailNormalizesDigestAndAuditsEntity` | Asserts user-admin query uses canonical digest and retains the audit entity contract. | Canonical, exact legacy fallback, and collision paths are covered by the service suite. | Reindex is constrained by the unique digest index. | No plaintext output. | One canonical query and rare legacy fallback. | Shared resolver prevents drift. | Mixed-case reindex and collision tests pass. | PASS |
| `repository.AdminAuditEntry` | Represents administrator and operator actors with an explicit kind and nullable administrator actor. | Administrator rows require a nonzero ID; operator rows require nil. | Shared readers consume nullable state safely. | Actor truthfulness is a security/audit boundary. | Small struct. | Explicit enum plus pointer is idiomatic. | PostgreSQL operator readback and administrator compatibility tests. | PASS |
| `adminAuditListForEntitySQL` | Returns actor kind and nullable actor ID in the scanner’s declared order. | Administrator and operator rows use the same list path. | Ordered read only. | Preserves truthful origin. | One indexed entity query. | SQL remains colocated and embedded. | PostgreSQL operator readback. | PASS |
| `ListAuditForEntity` | Lists every audit row after bootstrap without inventing actor identity. | Operator NULL actor and administrator UUID rows both scan successfully. | Rows closed/deferred correctly. | Audit availability and attribution remain intact. | Bounded entity query. | Existing API is reused. | Bootstrap lifecycle reads the inserted operator row. | PASS |
| `scanAdminAuditEntry` | Scans actor kind and nullable administrator actor into the shared model. | NULL operator actor is accepted; invalid row data returns an error. | Pure row conversion. | Does not fabricate an administrator. | One row scan. | Pointer field matches PostgreSQL nullability. | B-1 PostgreSQL readback. | PASS |
| `validateAdminAuditEntry` | Preserves administrator validation and supports operator entries. | Administrator requires a nonzero admin ID; operator requires nil ID and valid kind. | Used by shared writes. | Actor kind drives attribution invariants. | Constant work. | Closed enum validation. | Security and repository validation matrix. | PASS |
| `adminAuditInsertSQL` | Shared administrator audit insert preserves administrator actor semantics. | Explicit actor kind and ID are inserted for administrator mutations. | Same transaction semantics. | No false operator attribution. | One insert. | Explicit column list. | Existing administrator audit tests. | PASS |
| `useradmin.Service.lookupRequest` | User administration derives canonical and exact trimmed legacy digests. | Empty normalized input and digest errors remain closed. | No mutation in request construction. | Uses pseudonymous HMAC inputs only. | At most two digest calculations. | Shared request boundary. | Canonical/legacy service tests. | PASS |
| `LookupAdminUsers` | Returns exact canonical or legacy target through the resolver adapter. | Legacy result is reindexed; different identities fail closed. | Repository update is unique-index protected. | Identity ambiguity is not silently merged. | Indexed single-digest queries. | Primitive stays narrow and parameterized. | PostgreSQL reindex/collision tests. | PASS |
| `adminUserGetByDigestSQL` | Supports each selected digest from the shared resolver. | Canonical miss permits exact legacy fallback above it. | Read only; reindex is separate. | No plaintext email or broad matching. | Indexed `LIMIT 1` query. | Correct single-digest primitive. | User-admin mixed-case tests. | PASS |
| `AdminAuditActorKind` | Explicitly distinguishes administrator and operator audit origins. | Unknown values are rejected by validation. | Immutable value. | Prevents truthful-origin loss. | Small string enum. | Go named string type. | Audit validation/readback tests. | PASS |
| `AdminAuditActorAdministrator` | Encodes administrator-origin audit rows. | Administrator rows require a nonzero actor ID. | No mutable state. | Keeps existing actor semantics. | Constant. | Typed constant. | Administrator audit integration tests. | PASS |
| `AdminAuditActorOperator` | Encodes bootstrap operator-origin audit rows. | Operator rows require a nil administrator actor ID. | No mutable state. | Preserves operator attribution. | Constant. | Typed constant. | Bootstrap list readback test. | PASS |
| `ResolveCanonicalEmailIdentity` | Resolves canonical first, then exact legacy, and rejects different identities. | Not-found fallback, same-target replay, lookup errors, reindex errors, and collisions are handled. | Reindex occurs only for one legacy target and unique-index conflicts propagate. | Prevents ambiguous account selection. | At most two indexed lookups and one rare update. | Generic shared helper. | Full branch matrix. | PASS |
| `PostgresAdminUserRepository.ReindexUserEmailDigest` | Reindexes one identified user to the canonical digest. | Invalid ID/digest and unique collisions fail closed. | Single parameterized update. | Cannot update an arbitrary row without exact ID. | Indexed primary-key update. | Repository interface implementation. | PostgreSQL mixed-case and collision test. | PASS |
| `useradmin.Service.Lookup` | Applies authorization and shared canonical/legacy exact lookup. | Not-found remains an empty exact page; collisions and repository errors propagate safely. | Reindex is performed only after exact identity resolution. | No broad email matching or identity merge. | One normal query, rare fallback/update. | Existing service API preserved. | Mixed-case and collision tests. | PASS |
| `useradmin.Service.lookupUserByEmailDigest` | Adapts the repository’s one-digest lookup to the generic resolver. | Zero rows map to not-found; multiple rows remain repository-controlled. | Context passed through. | Digest only. | One indexed query. | Small adapter. | User-admin service tests. | PASS |
| `AdminUserRepository` | Adds the collision-safe digest reindex capability to the user-admin boundary. | Implementations must preserve lookup and update errors. | Context-aware interface. | Keeps identity migration internal. | No direct I/O. | Explicit interface contract. | All fakes and PostgreSQL implementation compile/test. | PASS |
| `httpapi.AdminController.transactionalMutation` | Preserves administrator actor kind and pointer ID in shared audit calls. | Existing admin mutations still persist valid administrator audits. | Transaction and audit sequencing unchanged. | No operator identity is fabricated for HTTP mutations. | Existing transaction cost. | Pointer contract adapted consistently. | HTTP administrator audit assertions. | PASS |
| `dataimporter.confirm` | Keeps integration helper aligned with nullable audit actor contract. | Test confirmation remains deterministic. | Test-local transaction path. | No production behavior change. | Constant. | Fixture compatibility. | Full dataimporter suite. | PASS |
| `TestResolveCanonicalEmailIdentity` | Covers canonical hit, legacy reindex, same identity, collision, lookup errors, and reindex errors. | All resolver branches assert exact outcomes. | Mocked callbacks expose ordering. | Collision fails closed. | Small unit matrix. | Table-driven. | Self. | PASS |
| `TestPostgresAdminUserReindexesLegacyDigestAndRejectsCollision` | Proves legacy user-admin lookup can reindex and refuses unique collision. | Both historical and ambiguous states are covered. | Real PostgreSQL unique index. | No account merge. | Small isolated DB workload. | Integration regression. | Self. | PASS |
| `TestLookupExactEmailReindexesLegacyMixedCaseDigest` | Proves user-admin exact lookup finds and migrates a legacy mixed-case digest. | Legacy-only target returns once and canonical digest is persisted. | Reindex is observable after lookup. | Exact matching only. | Two digest paths. | Focused service test. | Self. | PASS |
| `TestLookupExactEmailRefusesCanonicalLegacyCollision` | Proves user-admin refuses two identities for canonical and legacy digests. | No result is returned and no reindex occurs. | Unique identity state remains unchanged. | Fail-closed collision behavior. | Bounded lookups. | Focused negative test. | Self. | PASS |
| `memoryAdminUsers.LookupAdminUsers` | Test repository supports exact digest lookup for both resolver branches. | Missing digest returns empty page. | Test-local map state. | Test-only. | Constant. | Fixture seam. | User-admin tests. | PASS |
| `memoryAdminUsers.ReindexUserEmailDigest` | Test repository models canonical migration and collision. | Existing digest collision returns the repository error. | Test-local update. | Test-only. | Constant. | Fixture seam. | User-admin tests. | PASS |
| `recordingDigester.DigestForWrite` | Records canonical and legacy digest inputs for user-admin assertions. | Both exact inputs remain inspectable without storing plaintext output. | Test-local slice. | Test-only. | Constant. | Fixture helper. | Lookup request tests. | PASS |
| `TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData` | Verifies shared audit validation remains safe after actor-model expansion. | Unsafe and unbounded fields are rejected. | No persistent mutation on invalid input. | Prevents audit abuse. | Unit-level. | Existing security test extended. | Self. | PASS |
| `TestPostgresComplianceAndAdminRepositories` | Reads administrator audit rows with explicit actor kind and nonnil actor ID. | Existing audit behavior remains compatible. | Real PostgreSQL rows. | Attribution remains truthful. | Small integration workload. | Repository contract test. | Self. | PASS |
| `TestPostgresComplianceAndAdminRepositoryValidationAndErrors` | Covers actor pointer and kind validation/error paths. | Invalid combinations fail closed. | Transaction behavior remains bounded. | No invalid actor attribution. | Unit/integration matrix. | Existing validation test extended. | Self. | PASS |
| `TestAdminAuditSnapshotValidationRollsBackTransaction` | Confirms invalid audit snapshots roll back mutation after actor-model changes. | Validation failure leaves state unchanged. | Transaction abort. | Audit cannot be bypassed. | One transaction. | Regression test. | Self. | PASS |
| `TestAdminAuditPersistenceErrorPreservesCause` | Confirms persistence errors still preserve the underlying cause. | Error identity remains available to callers. | No partial state. | Safe closed failure. | One path. | Go error wrapping. | Self. | PASS |
| `TestAdminMutationAuditSuccessfulCommitPath` | Confirms administrator mutation and audit commit together. | Valid pointer actor is persisted. | Commit path. | Administrator attribution remains valid. | One transaction. | Existing behavior retained. | Self. | PASS |
| `TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit` | Confirms replay commits without creating a duplicate audit. | Replay returns nil audit ID. | No duplicate row. | Idempotent mutation semantics. | One transaction. | Nullable result contract. | Self. | PASS |
| `TestAdminMutationControlOrderAtomicAuditAndSanitizedEnvelopes` | Confirms HTTP admin mutations preserve audit control order and safe response envelopes. | Success and failure ordering remain asserted. | Atomic transaction. | No sensitive response expansion. | Existing request cost. | Controller regression. | Self. | PASS |
| `TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit` | Confirms user-admin retry still requires authorization/CSRF and commits a valid administrator audit. | Unauthorized paths do not mutate. | Transaction is atomic. | Security controls remain intact. | Existing HTTP workload. | Controller regression. | Self. | PASS |
| `maxStoredPasswordHashBytes` | Defines the supported decoded stored-hash upper bound. | Parser rejects values above 64 bytes and below the minimum. | Constant. | Limits parser allocation and conversion range. | No I/O. | Named constant documents policy. | Oversized parser test. | PASS |
| `TestIsUsablePasswordCredentialMatchesAuthenticationParser` | Confirms eligibility and verification use the same parser, including oversized hash rejection. | Malformed Argon2, Base64, short hash/salt, and valid cases are covered. | Pure parser test. | No credential disclosure. | Bounded parsing. | Shared-contract regression. | Self. | PASS |
| `NewPasswordHasher` | Accepts only Argon2 key lengths from 16 through `maxStoredPasswordHashBytes` (64), matching the parser’s supported decoded hash range. | Rejects below-minimum and above-maximum values; valid 64-byte configuration constructs successfully. | Stores only bounded configuration; rejection occurs before Argon2 work. | Prevents creation of credentials the authentication parser cannot consume. | Constructor check is constant-time and avoids oversized allocation. | Explicit shared bound is simple and idiomatic. | Boundary test covers 64 accepted and 65 rejected. | PASS |
| `PasswordHasher.HashPassword` | Generates credentials accepted by the parser for every constructor-accepted key length. | 64-byte output is parsed, usable for bootstrap, and verifies; 65-byte generation is unreachable because construction rejects it. | Uses validated immutable parameters and per-call random salt. | No unusable generated credential within accepted configuration. | Argon2 output is bounded to 64 bytes. | Producer/parser contract is consistent. | Boundary test verifies generation, parser eligibility, and authentication. | PASS |
| `TestPasswordHasherKeyLengthMatchesStoredParserBoundary` | Proves the lower/upper constructor boundaries and producer/parser agreement. | Accepts 64, verifies generated credential, checks parsed key length, and rejects 65. | Pure test-local hasher; no external resources. | Prevents future credential availability regression. | One bounded 64-byte Argon2 operation. | Focused table-free boundary regression. | Covers the prior I-2 reproduction directly. | PASS |

## 7. Findings

All findings from prior review cycles are closed: B-1 shared audit readback, I-1 user-admin canonical/legacy resolution, O-1 bounded parser conversions with focused `gosec`, and I-2 producer/parser key-length consistency.

| ID | Severity | File:line | Symbol | Finding and reproduction | Exact repair instructions |
|---|---|---|---|---|---|
| None | N/A | N/A | No unresolved finding | I-2’s prior reproduction is now denied at construction; 64-byte generation, parser eligibility, and verification all pass. | No further repair required. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
```

## 8. Commands Run

All commands ran on 2026-07-27. Paths are relative to the repository unless stated.

| Command | Working directory | Exit | Result |
|---|---|---:|---|
| Read review checklist completely | repository | 0 | PASS |
| Read `code-review-skill/SKILL.md` exactly once and Go guide completely | repository | 0 | PASS |
| Read repository-required `golang-security` guidance | repository | 0 | PASS; focused Go security review applied |
| `git status --short`, baseline diff/name reconstruction, `rg` symbol/caller searches | repository | 0 | PASS |
| `gofmt -d $(rg --files cmd internal -g '*.go')` and `git -C .. diff --check f9a646ffede5c8a63ad787a07eaba7efc0433677` | `backend/` | 0 | PASS; no formatting or whitespace errors |
| `go test -count=1 ./internal/auth -run 'TestPasswordHasherKeyLengthMatchesStoredParserBoundary|TestPasswordHasherHashesAndVerifies|TestIsUsablePasswordCredentialMatchesAuthenticationParser'` | `backend/` | 0 | PASS; 64-byte generation verifies and 65-byte constructor configuration is rejected |
| `go test -count=1 -race ./internal/auth ./cmd/admin-bootstrap` | `backend/` | 0 | PASS; no race report |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/security ./internal/repository ./internal/seed ./internal/useradmin ./internal/httpapi ./internal/dataimporter` | `backend/` | 0 | PASS; no race report |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | `backend/` | 0 | PASS; full backend suite |
| `go test -count=1 -coverprofile=/tmp/task-276-third-rereview.cover ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/security ./internal/repository ./internal/seed ./internal/useradmin ./internal/httpapi ./internal/dataimporter` | `backend/` | 0 | PASS; CLI 98.5%, bootstrap 96.0%, auth 98.8%, security 99.7%, repository 86.6%, seed 100%, useradmin 93.1%, httpapi 87.4%, dataimporter 88.3% |
| `go tool cover -func=/tmp/task-276-third-rereview.cover` filtered to changed/task functions | `backend/` | 0 | PASS; `NewPasswordHasher`, `HashPassword`, parser, and verifier are 100%; only process `main` is 0% |
| `go vet ./...` | `backend/` | 0 | PASS |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS; no called-code vulnerabilities |
| `python3 scripts/check.py --quick` | repository | 0 | PASS; existing OpenAPI OAuth 302-only warning retained |
| `python3 scripts/validate-traceability.py && python3 scripts/validate-task-list.py && python3 scripts/validate-phase07-go-doc.py` | repository | 0 | PASS; Task 276 remains PREPARED and exported Go Doc is valid |
| PostgreSQL lifecycle/list-reader, mixed-case reindex/collision, replay, concurrent, rollback, credential, and administrator-audit compatibility tests | isolated test DB | 0 | PASS; B-1 and I-1 coverage included |
| `go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-dir=.go-mod-cache ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/repository ./internal/security ./internal/seed ./internal/useradmin` | `backend/` | 0 | PASS; 41 files, 9,493 lines, 0 issues; one bound-backed `#nosec G115` |
| `sha256sum` over every changed/untracked worktree path except this evidence file | repository | 0 | PASS; third-cycle hashes recorded in Section 9 |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-276-review.md` | repository | 0 | PASS; structural validator accepts the refreshed PASSED evidence |

Commands not applicable: frontend/browser checks because Task 276 adds no frontend surface. No implementation or task-list repair was performed.

## 9. Files Inspected and Staleness Fingerprints

SHA-256 values below were recomputed from the current worktree after the repaired implementation was inspected. The review evidence file itself is intentionally not included in its own hash manifest. Impacted pre-existing dependency files are included because they are part of the two findings.

| File | Purpose | SHA-256 |
|---|---|---|
| `backend/cmd/admin-bootstrap/main.go` | CLI | `cb553935d676c585ebb3fe7c7bde4209136f5f35b35ab224a1ef3c25917bbd1f` |
| `backend/cmd/admin-bootstrap/main_test.go` | CLI tests | `187bdb6c02879fdf9d40300bdef3f89b074e17ca9680323bb54debefe6248a77` |
| `backend/internal/adminbootstrap/service.go` | bootstrap service | `acd57107a90dc9f2448b30a68cf924a1188c623806a2ce8fd2b913b0b6180758` |
| `backend/internal/adminbootstrap/service_test.go` | service tests | `55084b83e0734e916d26aba401d0b9fac57b3edb8fc6e3c3b1cf0037f5447465` |
| `backend/internal/auth/oauth.go` | OAuth lookup | `356d260cd5f48af5d396bf3cb439a9a723a7dc18d8d37add37493e828e3d2fa6` |
| `backend/internal/auth/oauth_coverage_test.go` | OAuth tests | `dbeddff2ac8f89f1c6dc0ae815f33f45dcee46cefc85441cbd29a5201cc49887` |
| `backend/internal/auth/password.go` | credential parser | `18104445448c2622f3639057e5d240a83ea1d0447204c862ee76fb1fadc8678f` |
| `backend/internal/auth/password_test.go` | parser and boundary tests | `4a0ededdf1db03a5a723797df8879f3a9133692223e589921b07cfcfd5c31fd9` |
| `backend/internal/auth/registration_test.go` | registration tests | `385957bdf3dbf225c01bef72d23c2bac2e5c604fb4263a96d036e1e7110b7b7d` |
| `backend/internal/auth/service.go` | auth lookup/reindex | `ff5e75a6367a3131bec072909da3bdd5a2617dd95c85a1a79ddd0af8f52dc1c8` |
| `backend/internal/auth/service_test.go` | auth tests | `52cd1949b6f8837e1534b0c5165dcae4a4ff7e40c5ad6825776705527223f762` |
| `backend/internal/auth/token_test.go` | reauthentication test | `bb0f6afff8cae8fa0e51367d9ee980d90235102a015df03a82f1e97fb67fbb5f` |
| `backend/internal/repository/admin_bootstrap_repository.go` | bootstrap transaction | `52bb12469a52b8254c6b57a36360e138f931e11037005631455562347513fb9c` |
| `backend/internal/repository/admin_bootstrap_repository_test.go` | PostgreSQL tests | `a22c6b02481b06bb180ed7da0e6c4fe5140fc148065283d0ff87640ba28e2737` |
| `backend/internal/repository/errors.go` | bootstrap errors | `37ff496bbaf6e5c3a407533b240de56b7f962f9c8655b071fbbf0c9b75911f50` |
| `backend/internal/repository/types.go` | contracts and result | `d52db09e9dd3d8caa266f52c0bc718e9d707ffc267760e30c2f0d83cee5b02af` |
| `backend/internal/repository/sql/admin_bootstrap_audit.sql` | operator audit insert | `93ddc3728a0f342bfa109fe2c2626212e59c78dd8dabce23a35e597ae4db560b` |
| `backend/internal/repository/sql/admin_bootstrap_existing.sql` | admin invariant | `225768fd1506380ce1bb68c2539b7c07a4b23089c4a1cdb9f0ab0998bcc6e17a` |
| `backend/internal/repository/sql/admin_bootstrap_lock.sql` | advisory lock | `ace047c7d5fbac6ac40ca8d850e82fb10c8c8b17afde8c8ff49233e18591a139` |
| `backend/internal/repository/sql/admin_bootstrap_promote.sql` | role update | `6f18e34fa4974c39d0b753c56776e7d91febc5a76ca3ba2cc616074dcbe499ff` |
| `backend/internal/repository/sql/admin_bootstrap_target_by_digest.sql` | digest target | `4880c7edd566b76a446ad71d73dc826b21f58ab6e0882a5dafa986505f9dc0e5` |
| `backend/internal/repository/sql/admin_bootstrap_target_by_id.sql` | UUID target | `4aca82ccbc2834c258e6c2eb90e4c5c4cfccfefb5c9071a770464643a3837063` |
| `backend/internal/security/normalizer.go` | email canonicalization | `746b279252d939cf204a13a0d8cccf7152be650e8c0ebbae14693a92e1caeaf3` |
| `backend/internal/security/security_test.go` | normalizer tests | `ced1770a027f325a0437859e8baf6d87b58ee4b4590915c1c001bcdbdd820488` |
| `backend/internal/seed/development.sql` | seed removal | `d64ce7970576eee75899d290b792d36258f9a9d102cecf1014feacf14bd450e3` |
| `backend/internal/seed/seed_test.go` | seed test | `c1779635ed57c6fa310c5a0e3bd53768d9520b11fef1039295229c6030f99fae` |
| `backend/internal/useradmin/service_test.go` | user-admin test | `8c584a3915947c314c649051075fb27a1cf26115a371b7c5309e16d5335ae44d` |
| `database/migrations/000029_admin_bootstrap_actor.down.sql` | migration down | `7beffab6246d5a42ed64def5a6ccf9de6a6b61d90458f127c297dd23286fd57d` |
| `database/migrations/000029_admin_bootstrap_actor.up.sql` | migration up | `a978a20c60f903936f8f5f560987cbb9906aece93961dc3ae5b56cc6d7a0ce21` |
| `docs/design/DESIGN-006.md` | password contract | `6b9b7b0e33e47f15006df7a7a1c80c40b592adcb6721d9c17921f1d43dd888f6` |
| `docs/design/DESIGN-009.md` | admin design | `be8e978538a24db6d8e216fae4a5a5c8b5b5a942c29f04c2c7d6fda756c8ee94` |
| `docs/design/DESIGN-013.md` | identity contract | `fd0fa9430d96123a888cc1d22ebdd49de805d372a46111d241124236fa1b78b3` |
| `docs/implementation/02_TASK_LIST.md` | authoritative task row | `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2` |
| `docs/implementation/evidence/task-276-preparation.md` | updated preparation evidence | `23798897d48150a3e37ab7ae3c0c56fd646770d4ac5b2c7beb35f1253a02fa26` |
| `docs/operations/administrator-bootstrap.md` | operator procedure | `44cbdf1886a48460ebf435f3ccf1b7f6888f607be471929810d09c3e1a3ac1ef` |
| `backend/internal/repository/compliance_repository.go` | impacted audit reader | `4b08638b87bce3c31b149bba4e6b5269537f1e84fb253c5042261c7010ed49a3` |
| `backend/internal/repository/sql/admin_audit_list_for_entity.sql` | impacted audit list SQL | `696c0d053489d90b9fe51b9f471de8023321d06558a417be90a768625ec50e78` |
| `backend/internal/repository/sql/admin_audit_insert.sql` | impacted audit insert SQL | `5e7943af3cdfeaca439b63c55cdb9d781d36831a4743e5c476a633834fd8e71b` |
| `backend/internal/repository/encrypted_identity_repository.go` | legacy reindex dependency | `cb9afe281b4fcd52ebeb5ebd774f58171c3760928acda5e3451f97552ea00761` |
| `backend/internal/repository/sql/encrypted_user_update_digest.sql` | legacy reindex SQL | `8fd494557e5e8a015db410aa1787e1e17879c0297668b3992b91001e7dcce95a` |
| `backend/internal/useradmin/service.go` | impacted user-admin lookup | `b91b3e899412e63fd0b333f52a8814543b4529037eeae21dfee286144d51732a` |
| `backend/internal/repository/admin_user_repository.go` | impacted user-admin repository | `cdff6df7c907b7396e1c19f99ae46e0d96c71e10e213149effefe9d8cc85f525` |
| `backend/internal/repository/sql/admin_user_get_by_digest.sql` | impacted user-admin SQL | `0125bd5d03e287fbfab9c7fb9bde42a978463d8b9f0d35a60448d8fa838bee14` |
| `backend/internal/dataimporter/integration_test.go` | audit fixture dependency | `f7a9d70ab6f32cad7a72de770f5de133797781396f680200f830949e3b58e623` |
| `backend/internal/httpapi/admin_controller.go` | administrator audit caller | `de38a041d80a01e091d38060eb11ef72644c9c65f4a94c0e1904c98f9f839129` |
| `backend/internal/httpapi/admin_controller_test.go` | HTTP audit tests | `208b917f6feb2634d3b18bd9e31505ab262c48cec0bcd3f3717af4de61216452` |
| `backend/internal/httpapi/user_admin_controller_test.go` | user-admin audit tests | `ee6696a012d57e619a516a2e3a66cc783394c18a41a6ad563ae253c8e6b5ab77` |
| `backend/internal/repository/admin_audit_security_test.go` | audit security tests | `b72356fdf78db783ce668651827be393e5d59232f1010409f142a0e7fc960d66` |
| `backend/internal/repository/admin_user_repository_test.go` | user-admin PostgreSQL tests | `d514da5256916f02be3cb5d1d30244290216f4f6bd44068c9e4f85e691f114d7` |
| `backend/internal/repository/canonical_email_resolver.go` | shared resolver | `3cedddc8cbcd425d3ce65c601f4644a79907e5cb46f697c3df1c0e68f3ac5dae` |
| `backend/internal/repository/canonical_email_resolver_test.go` | resolver tests | `2384b95dd02f6691353b0a258d07fd90db1056fdda96b4bbcd000976ce7aa17c` |
| `backend/internal/repository/classification_admin_repository_test.go` | audit caller test | `0e6bb6520fe53be0f99cdebbc513111d5dd13580cf77390c78ac790d297be8e7` |
| `backend/internal/repository/manual_food_repository_test.go` | audit caller test | `e83642d13869e416dbc0c8681412322e43d10938d3543137185e39735583184d` |
| `backend/internal/repository/postgres_repository_test.go` | audit compatibility tests | `97685f5f4db8d6820ea8761d207983b360b92d47fa06e57f2c05d09a5365e241` |

```yaml
all_reviewed_files_hashed: true
hash_file_count: 55
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The second-cycle review evidence was stale for auth/password.go, auth/password_test.go, and the preparation evidence after I-2 repair; those hashes and affected symbols were re-reviewed here."
hash_scope: "Every changed or untracked task-surface implementation, test, SQL, migration, seed, design, task-row, preparation, and operator file, plus the directly inspected pre-existing encrypted identity SQL/repository dependencies; this review evidence file is excluded from its own manifest."
```

## 10. Coverage and Exceptions

- [x] Focused race command ran across CLI, bootstrap, auth, security, repository, seed, user-admin, HTTP API, and data-importer packages.
- [x] Full backend tests ran.
- [x] Focused coverage ran and every inventoried callable path was inspected.
- [x] PostgreSQL acceptance and adversarial tests ran, including B-1 list readback and I-1 reindex/collision cases.
- [x] The only process-only coverage exception is the `main` wrapper; it delegates to covered code and calls `os.Exit`.
- [x] Constructor boundary coverage accepts 64 bytes, rejects 65 bytes, and verifies the generated 64-byte credential through both authentication and bootstrap parsing.

Observed package coverage from `/tmp/task-276-third-rereview.cover`: CLI 98.5%, adminbootstrap 96.0%, auth 98.8%, security 99.7%, repository 86.6%, seed 100%, useradmin 93.1%, httpapi 87.4%, and dataimporter 88.3%. Filtered task callable functions are covered except process-only `main`; the I-2 boundary test covers the repaired constructor and producer/parser contract.

```yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-276-third-rereview.cover"
observed_line_coverage: "Task-owned callable paths covered except process-only main; package totals include pre-existing surfaces."
coverage_passed: true
```

## 11. Negative and Regression Checks

- [x] B-1 fixed: shared audit list SQL reads `actor_kind`; `AdminAuditEntry` uses `*uuid.UUID`; scanner and validator enforce administrator/operator invariants; PostgreSQL coverage reads operator `NULL` actor back.
- [x] I-1 fixed: authentication and user administration share canonical-first/exact-legacy resolution; legacy reindexing is unique-index and collision safe; mixed-case PostgreSQL tests pass.
- [x] O-1 fixed: decoded hashes are bounded to 16–64 bytes and focused `gosec` reports zero source issues.
- [x] I-2 fixed: `NewPasswordHasher` enforces 16–64 bytes; 64-byte output is parser- and verifier-compatible; 65-byte configuration is rejected before generation.
- [x] Replay audit ID is nullable and safe: replay returns nil and the CLI emits `audit_id=none`.
- [x] SQL is parameterized; no public promotion route or client role input was added.
- [x] Verification state is read-only during promotion; reauthentication documentation and JWT regression are present.
- [x] Seeded unusable administrator is removed.
- [x] Output tests exclude email, credential, URL, key, hash, salt, token, and cookie classes.
- [x] Concurrency and rollback checks pass.
- [x] No unresolved constructor/parser bound inconsistency remains.

## 12. Decision

A task may be PASSED only when every acceptance criterion and symbol audit passes, every reviewed file is hashed, and no blocking or important finding remains. B-1, I-1, O-1, replay-ID, and I-2 checks pass; the current evidence is complete and current.

```yaml
decision: "PASSED"
reason: "All acceptance criteria and 140 audited symbols pass; the third-cycle I-2 repair aligns NewPasswordHasher with the 16–64 byte parser bound and boundary tests prove generated credentials remain verifiable."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None. Preserve the PREPARED task-list row and retain this evidence with the refreshed hashes."
```

## 13. Repair Context

### Closed Findings From Prior Review

The prior mixed-case normalization blocker is closed. `security.normalizeEmail` now lowercases the typed email contract, registration persists canonical data, login/reset/OAuth/bootstrap derive canonical plus exact trimmed legacy digests, different-account collisions fail closed, and legacy rows are reindexed through the unique digest path. Mixed-case PostgreSQL tests cover bootstrap and auth behavior.

The prior credential-eligibility blocker is closed. Bootstrap no longer relies on SQL prefix matching. It receives `auth.IsUsablePasswordCredential`, which delegates to the parser used by `VerifyPassword`; malformed parameters, invalid hash/salt Base64, short hash/salt, weak parameters, and valid generated credentials are covered.

Replay ambiguity is closed. `AdministratorBootstrapResult.AuditID` is a pointer, replay returns nil, and CLI output says `audit_id=none`.

### Third-Cycle Repair Closure

I-2 is closed. `NewPasswordHasher` now enforces the shared 16–64 byte key-length contract. `TestPasswordHasherKeyLengthMatchesStoredParserBoundary` accepts and verifies a generated 64-byte credential, checks parser eligibility and parsed key length, and rejects a 65-byte constructor configuration before Argon2 generation.

All prior B-1, I-1, O-1, and replay-ID repairs remain verified by the complete 140-symbol audit and current focused/full test lanes.

### Scope Preservation

Do not add a public promotion endpoint, accept client role input, mark email verified, print PII or secrets, bypass encrypted/HMAC lookup, remove the advisory lock, weaken atomic audit persistence, or make existing sessions gain fresh claims without sign-out/sign-in. Do not edit the task-list status as part of the repair.
