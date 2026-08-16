# Task 276 preparation evidence

## Outcome

Task 276, Phase 08.02 Safe Administrator Bootstrap, is implemented and repaired after independent review. This repair did not edit `docs/implementation/02_TASK_LIST.md`; the independent workflow had already changed Task 276 from `OPEN` to `PREPARED`, and that external hunk was preserved unchanged.

The repaired implementation adds a dedicated operator-only Go CLI and no HTTP role-promotion route. One trim/validate/lower-case email contract now governs registration, login, password reset, OAuth account matching/linking, user administration, and bootstrap. Exact pre-canonical mixed-case digests are reindexed to the canonical digest only after unambiguous resolution; different-account canonical collisions fail closed. Password eligibility now calls the same parser as authentication rather than interpreting hash prefixes in SQL. Replay has no audit UUID instead of exposing an optional zero UUID.

## Independent review repair

### Rejected findings reproduced

The `diagnose` loop added two PostgreSQL regressions before the repair and observed both exact failures:

- `TestExecuteBootstrapReindexesLegacyMixedCaseEmail`: failed with `administrator bootstrap target not found` because registration/auth digested case-preserved email while bootstrap digested lower-case email.
- `TestExecuteBootstrapRejectsMalformedPasswordCredential`: returned success for malformed Argon2 parameters because SQL accepted a prefix and non-empty salt.

### Repair baseline

- Repair baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- Preparation evidence SHA-256 before repair: `e1b1f8d1e5053942db02293f87183663602c925ee4179a239cb5f6c8cecf4b74`.
- Task-list SHA-256 before repair: `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2`.
- Repair-start status contained the prior Task 276 implementation plus an independently modified task-list row (`OPEN` to `PREPARED`). The repair preserved that row and all unrelated state.

### Repair behavior

- `security.normalizeEmail` now owns the canonical identity contract: trim, RFC-style validation, then lower-case.
- Registration encryption/digesting, login, password reset, OAuth profile/account matching and linking, user-admin lookup, and bootstrap all consume that same normalized value.
- Login, password reset, OAuth matching, and bootstrap derive both canonical and exact trimmed legacy digests when needed. They compare both results before reindexing; different users produce `ErrCanonicalEmailCollision`. PostgreSQL's unique digest index remains the concurrent collision guard.
- Bootstrap target SQL returns password hash/salt as data and only computes structurally populated encrypted OAuth eligibility. It no longer parses password syntax.
- `auth.IsUsablePasswordCredential` and `PasswordHasher.VerifyPassword` share `parseStoredPasswordCredential`. The parser rejects missing/duplicated/malformed/weak parameters, invalid Base64, hashes shorter than 16 bytes, and salts shorter than 16 bytes.
- `AdministratorBootstrapResult.AuditID` is nullable. Created results carry the real audit UUID; replay carries `nil`, rendered safely as `audit_id=none`.

## Baseline and reference confidence

- Worktree: `/home/wiktor/Work/mealswapp`
- Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Original implementation `git status --short`: clean. Repair status is described above.
- Authoritative task row: Task 276 in `docs/implementation/02_TASK_LIST.md`.
- Dependency: Task 247 is `PASSED`.
- Reference confidence: **HIGH**. The implementation was derived from the complete Task 276 row, `docs/design/DESIGN-009.md`, Task 247 preparation/review evidence, the existing encrypted identity/digest/auth/session repositories, admin audit transaction boundary, migrations, seed behavior, and existing CLI/configuration patterns.
- This repair did not edit task-list status. The externally supplied `OPEN` to `PREPARED` hunk is excluded from repair attribution.

## Design and security decisions

- Authorization boundary: bootstrap exists only under `backend/cmd/admin-bootstrap`; no public controller, route, OpenAPI operation, or general role-management API was added.
- Target selection: DESIGN-013 now defines one trim/validate/lower-case email contract for every identity flow. Email and UUID cannot be combined. Exact legacy digest fallback is bounded to the supplied trimmed spelling and reindexes only after collision-safe dual lookup.
- Environment boundary: `--environment` is mandatory and must exactly match `MEALSWAPP_ENV`; production additionally requires `--confirm-production`.
- Credential boundary: eligibility requires the account-level verified Login Method projection and either password material accepted by the exact authentication parser or a populated encrypted OAuth identity. SQL does not interpret password hashes.
- Concurrency: one constant PostgreSQL transaction advisory lock serializes all bootstrap attempts across processes.
- First-administrator invariant: under the lock, the transaction checks the selected target and existing administrator. Same-target replay commits no mutation and no duplicate audit; any different existing administrator is refused.
- Atomicity: role update and audit insert share one transaction. Audit failure rolls the role back.
- Truthful actor: migration 29 adds `actor_kind`; bootstrap writes `actor_kind = 'operator'`, `admin_user_id = NULL`, and the promoted user as `entity_id`. Existing administrator audits default to and require `actor_kind = 'administrator'`.
- Privacy: request IDs must be UUIDs; SQL is parameterized; internal/database/key errors are mapped to a closed CLI vocabulary; stdout/stderr never include supplied email, credentials, database URL, key material, hashes, salts, tokens, or cookies.
- Reauthentication: bootstrap does not mutate sessions. Existing JWTs retain their signed `user` role; the user must sign out and sign in to receive fresh role and verification claims.
- `golang-security` repair review: **PASS**. Canonical identity ambiguity, digest migration races/collisions, parser equivalence, malformed credential denial, SQL/command injection, secrets, output, transaction rollback, concurrency, actor attribution, and session claims were reviewed. No shell construction, SQL hash parsing, silent account merge, public authorization path, or PII logging remains.

## Task-owned changed paths

### Independent-review repair paths

- `backend/cmd/admin-bootstrap/main.go`
- `backend/cmd/admin-bootstrap/main_test.go`
- `backend/internal/adminbootstrap/service.go`
- `backend/internal/adminbootstrap/service_test.go`
- `backend/internal/auth/oauth.go`
- `backend/internal/auth/oauth_coverage_test.go`
- `backend/internal/auth/password.go`
- `backend/internal/auth/password_test.go`
- `backend/internal/auth/registration_test.go`
- `backend/internal/auth/service.go`
- `backend/internal/auth/service_test.go`
- `backend/internal/repository/admin_bootstrap_repository.go`
- `backend/internal/repository/admin_bootstrap_repository_test.go`
- `backend/internal/repository/errors.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/sql/admin_bootstrap_target_by_digest.sql`
- `backend/internal/repository/sql/admin_bootstrap_target_by_id.sql`
- `backend/internal/security/normalizer.go`
- `backend/internal/security/security_test.go`
- `backend/internal/useradmin/service_test.go`
- `docs/design/DESIGN-006.md`
- `docs/design/DESIGN-009.md`
- `docs/design/DESIGN-013.md`
- `docs/operations/administrator-bootstrap.md`
- `docs/implementation/evidence/task-276-preparation.md`

### Production and persistence

- `backend/cmd/admin-bootstrap/main.go`
- `backend/internal/adminbootstrap/service.go`
- `backend/internal/auth/oauth.go`
- `backend/internal/auth/password.go`
- `backend/internal/auth/service.go`
- `backend/internal/repository/admin_bootstrap_repository.go`
- `backend/internal/repository/errors.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/sql/admin_bootstrap_audit.sql`
- `backend/internal/repository/sql/admin_bootstrap_existing.sql`
- `backend/internal/repository/sql/admin_bootstrap_lock.sql`
- `backend/internal/repository/sql/admin_bootstrap_promote.sql`
- `backend/internal/repository/sql/admin_bootstrap_target_by_digest.sql`
- `backend/internal/repository/sql/admin_bootstrap_target_by_id.sql`
- `database/migrations/000029_admin_bootstrap_actor.up.sql`
- `database/migrations/000029_admin_bootstrap_actor.down.sql`
- `backend/internal/seed/development.sql`
- `backend/internal/security/normalizer.go`

### Tests

- `backend/cmd/admin-bootstrap/main_test.go`
- `backend/internal/adminbootstrap/service_test.go`
- `backend/internal/auth/oauth_coverage_test.go`
- `backend/internal/auth/password_test.go`
- `backend/internal/auth/registration_test.go`
- `backend/internal/auth/service_test.go`
- `backend/internal/repository/admin_bootstrap_repository_test.go`
- `backend/internal/auth/token_test.go`
- `backend/internal/security/security_test.go`
- `backend/internal/seed/seed_test.go`
- `backend/internal/useradmin/service_test.go`

### Design, operator, and evidence documentation

- `docs/design/DESIGN-009.md`
- `docs/design/DESIGN-006.md`
- `docs/design/DESIGN-013.md`
- `docs/operations/administrator-bootstrap.md`
- `docs/implementation/evidence/task-276-preparation.md`

## Added or modified symbols

### Production executable symbols added

- `main`
- `run`
- `executeBootstrap`
- `safeBootstrapError`
- `newLookupKeyLoader`
- `lookupKeyLoader.ActiveLookupKey`
- `lookupKeyLoader.LookupKey`
- `adminbootstrap.NewService`
- `adminbootstrap.Service.Bootstrap`
- `repository.NewPostgresAdministratorBootstrapRepository`
- `repository.PostgresAdministratorBootstrapRepository.BootstrapAdministrator`
- `repository.loadAdministratorBootstrapTarget`
- `repository.validateAdministratorBootstrapSelector`

### Repair-added executable symbols

- `auth.IsUsablePasswordCredential`
- `auth.parseStoredPasswordCredential`
- `auth.CoreAuthService.emailLookupDigests`
- `auth.lookupAndReindexUserByEmail`
- `repository.scanAdministratorBootstrapTarget`

### Repair-modified executable symbols

- `security.normalizeEmail`
- `auth.PasswordHasher.VerifyPassword`
- `auth.parseHashParams`
- `auth.CoreAuthService.Login`
- `auth.CoreAuthService.RequestPasswordReset`
- `auth.CoreAuthService.CompleteOAuth`
- `adminbootstrap.Service.Bootstrap`
- `repository.NewPostgresAdministratorBootstrapRepository`
- `repository.PostgresAdministratorBootstrapRepository.BootstrapAdministrator`
- `repository.loadAdministratorBootstrapTarget`
- `repository.validateAdministratorBootstrapSelector`
- CLI `run`, `executeBootstrap`, and `safeBootstrapError`

### Production declarations added

- `bootstrapExecutor`
- `lookupKeyLoader`
- `adminbootstrap.Request`
- `adminbootstrap.Service`
- `repository.PostgresAdministratorBootstrapRepository`
- `repository.administratorBootstrapTarget`
- `repository.AdministratorBootstrapSelector`
- `repository.AdministratorBootstrapResult`
- `repository.AdministratorBootstrapRepository`
- `repository.ErrAdministratorBootstrapTargetNotFound`
- `repository.ErrAdministratorBootstrapUnverified`
- `repository.ErrAdministratorBootstrapNoCredential`
- `repository.ErrAdministratorAlreadyExists`
- `repository.ErrCanonicalEmailCollision`
- `repository.PasswordCredentialValidator`
- Embedded SQL variables: `adminBootstrapLockSQL`, `adminBootstrapTargetByDigestSQL`, `adminBootstrapTargetByIDSQL`, `adminBootstrapExistingSQL`, `adminBootstrapPromoteSQL`, and `adminBootstrapAuditSQL`.

### Test symbols added or modified

- Added CLI tests: `TestRunReadsEmailInteractivelyAndEmitsOnlySafeMetadata`, `TestRunSupportsUUIDAndSafeFailures`, `TestRunRejectsMissingOrMalformedArguments`, `TestSafeBootstrapErrorsAndLookupKeyConfiguration`, `TestExecuteBootstrapReturnsOnlySafeCompositionFailures`, and `TestExecuteBootstrapComposesSuccessfulRuntime`.
- Repair-added CLI/PostgreSQL tests: `TestExecuteBootstrapReindexesLegacyMixedCaseEmail`, `TestExecuteBootstrapRefusesCanonicalEmailCollision`, and `TestExecuteBootstrapRejectsMalformedPasswordCredential`.
- Added CLI helpers: `resetBootstrapCommandDatabase`, `insertBootstrapCommandUser`, and `ioDiscard.Write`.
- Added service test fixtures/methods: `testKeys.ActiveLookupKey`, `testKeys.LookupKey`, `recordingRepository.BootstrapAdministrator`, `failingKeys.ActiveLookupKey`, and `failingKeys.LookupKey`.
- Added service tests: `TestBootstrapValidatesEnvironmentAndNormalizesEmail`, `TestBootstrapSupportsOnlyOneUUIDSelector`, and `TestBootstrapHidesLookupFailureAndPropagatesRepositoryOutcome`.
- Added repository tests: `TestPostgresAdministratorBootstrapLifecycle`, `TestPostgresAdministratorBootstrapRejectsInvalidTargets`, `TestPostgresAdministratorBootstrapAcceptsEncryptedOAuthCredential`, `TestPostgresAdministratorBootstrapSerializesConcurrentAttempts`, `TestPostgresAdministratorBootstrapRollsBackWhenAuditFails`, and `TestPostgresAdministratorBootstrapValidationAndDatabaseFailures`.
- Repair-added repository tests: `TestPostgresAdministratorBootstrapReindexesLegacyEmailDigest` and `TestPostgresAdministratorBootstrapRejectsCanonicalEmailCollision`.
- Added repository test helpers/declarations: `bootstrapTestHash`, `bootstrapRequestID`, `bootstrapSelector`, `validateBootstrapTestCredential`, and `createBootstrapUserOnDB`.
- Added authentication test: `TestAdministratorBootstrapDoesNotChangeExistingSessionClaims`.
- Repair-added authentication/security tests: `TestIsUsablePasswordCredentialMatchesAuthenticationParser` and `TestCoreAuthServiceReindexesLegacyMixedCaseEmailWithoutMergingCollision`; registration and OAuth composition tests now assert canonical mixed-case persistence.
- Repair-modified authentication fixtures/tests: `oauthUnsupportedIdentityRepository.ReindexUserEmailDigest`, `fakeRegistrationRepository.CreateUserWithConsent`, `memoryIdentityRepository.ReindexUserEmailDigest`, `TestPasswordHasherHashesAndVerifies`, `TestRegistrationServiceConsentGate`, `TestCoreAuthServiceOAuthRealTrialTracker`, and `TestCoreAuthServiceOAuthBoundary`.
- Repair-modified normalization test: `TestNormalizeInput`.
- Repair-modified user administration test: `TestLookupExactEmailNormalizesDigestAndAuditsEntity` now asserts the shared lower-case canonical contract.
- Modified `TestRunIsIdempotentAndSeedsRepositoryFixtures` to prove seeding creates no administrator.

## Criteria coverage

| Task 276 criterion | Evidence |
|---|---|
| Dedicated operator-only CLI; no public promotion endpoint | CLI package exists; no router/controller/OpenAPI change. |
| Normalized, case-insensitive configured email lookup | DESIGN-013 and `security.normalizeEmail` define one canonical contract. Registration, login/reset, OAuth, user-admin, and bootstrap tests assert it. PostgreSQL tests prove canonical mixed-case lookup, exact legacy reindex, and collision refusal. |
| Optional exact UUID alternative | Service/CLI tests reject combined selectors and accept UUID; PostgreSQL lifecycle test selects exact UUID. |
| Existing verified Login Method and real credential | Bootstrap SQL returns password material without parsing it; `auth.IsUsablePasswordCredential` shares the authentication parser. Unit and PostgreSQL tests reject malformed/weak parameters, invalid Base64, short hashes/salts, and accept generated credentials plus encrypted OAuth. |
| First administrator only | Lifecycle test creates the first administrator and refuses a different target. |
| Same-target idempotent replay | Lifecycle and CLI runtime tests return unchanged, retain one audit, and expose `AuditID == nil`/`audit_id=none` rather than a zero UUID. |
| Concurrent serialization and one audit effect | Two raced PostgreSQL calls produce one create, one replay, and one bootstrap audit. |
| Atomic role and truthful operator audit | Migration actor constraint, lifecycle audit assertions, and forced-trigger rollback test. |
| Explicit environment and production confirmation | Service tests cover mismatch and absent production confirmation; CLI requires `--environment`. |
| Safe output and interactive input | CLI tests capture stdin/stdout/stderr and assert closed metadata with no supplied email or secret classes. |
| Remove unusable seeded administrator | Development seed deletes the legacy fixture/audit and no longer inserts either; seed test asserts zero administrators. |
| Fresh claims require sign-out/sign-in | JWT test proves an existing user token remains `user` while a newly issued token can contain `admin`; design/operator docs mandate reauthentication. |
| Do not mark email verified | Bootstrap SQL updates only `role` and `updated_at`; verification state is read-only eligibility input. |
| Design/operator semantics | `DESIGN-009.md` and `docs/operations/administrator-bootstrap.md` define authorization, environment, actor, replay, failure, privacy, and reauthentication behavior. |

## Verification commands and results

All commands ran on 2026-07-27 from the stated working directory.

| Command | Result |
|---|---|
| `go test -count=1 -race ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/security ./internal/repository ./internal/seed` (`backend/`) | PASS; no race report after repair. |
| `go test -count=1 ./...` (`backend/`) | PASS across all backend commands and packages. |
| `python3 scripts/check.py --quick` | PASS after repair; changed backend packages, formatting, vet, vulnerability scan, docs/task validation, OpenAPI, generators, and static checks passed. OpenAPI retained its existing OAuth 302-only warning. |
| `go test -count=1 -coverprofile=/tmp/task-276-repair.cover ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/security ./internal/repository` | PASS; CLI 98.5%, adminbootstrap 96.0%, auth 97.4%, security 99.7%, repository 86.2%. Repair-critical password parsing and canonical normalizer functions report 100%; CLI behavior/composition functions report 100% except process-only `main`. |
| `go vet ./...` (`backend/`, through quick gate) | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` (`backend/`, through quick gate) | PASS: no vulnerabilities in called code. One package and 19 module advisories are not called. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-phase07-go-doc.py` | PASS for Phase 07/08 exported Go Doc. |
| `python3 scripts/validate-task-list.py` | PASS; Task 276 is externally `PREPARED`; repair made no task-list edit. |
| `go test -count=1 -race ./internal/repository -run TestPostgresAdministratorBootstrap` (`backend/`) | PASS after final formatting; all isolated PostgreSQL bootstrap regressions passed under the race detector. |
| `git diff --check` | PASS. |

## Coverage note

There is no behavioral coverage exception. Every rejected behavior has unit and isolated PostgreSQL regression coverage. The command package remains below 100% solely because unit coverage cannot enter the process-only `main` wrapper; it delegates to covered behavior and calls `os.Exit`. Package totals include substantial pre-existing surfaces outside Task 276.

## Risks and blockers

- Blockers: none.
- The CLI’s `--email` option can expose an address through shell history or process inspection. Operator documentation therefore makes interactive stdin the preferred path; CLI tests prove that path.
- The lookup key version remains `local-v1`, matching the repository’s current application key loader. A future key-rotation design would need a multi-version operator lookup strategy; that is outside Task 276.
- Credential eligibility validates stored format and security parameters with the authentication parser because bootstrap must not ask for the user's plaintext secret. Actual secret verification remains owned by login.
- A historical mixed-case digest can be reindexed only when canonical lookup or the exact trimmed legacy spelling resolves it; arbitrary unknown historical casing cannot be inferred from HMAC material. UUID remains the safe advanced selector for such an account. Different-account canonical collisions fail closed.
- Migration rollback deletes operator-origin bootstrap audit rows before restoring the historical non-null administrator actor constraint. This is limited to an explicit schema downgrade and is documented by the down migration itself.

## First-review repair hashes

These SHA-256 values were captured after the first independent-review repair and are superseded for files changed by the re-review repair below. The task-list hash remains unchanged.

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2` |
| `backend/cmd/admin-bootstrap/main.go` | `cb553935d676c585ebb3fe7c7bde4209136f5f35b35ab224a1ef3c25917bbd1f` |
| `backend/cmd/admin-bootstrap/main_test.go` | `187bdb6c02879fdf9d40300bdef3f89b074e17ca9680323bb54debefe6248a77` |
| `backend/internal/adminbootstrap/service.go` | `acd57107a90dc9f2448b30a68cf924a1188c623806a2ce8fd2b913b0b6180758` |
| `backend/internal/adminbootstrap/service_test.go` | `55084b83e0734e916d26aba401d0b9fac57b3edb8fc6e3c3b1cf0037f5447465` |
| `backend/internal/auth/oauth.go` | `356d260cd5f48af5d396bf3cb439a9a723a7dc18d8d37add37493e828e3d2fa6` |
| `backend/internal/auth/password.go` | `63740e19a2ec079a5738e7d376786f658459886f3402c382eefdb0c7a6e55d84` |
| `backend/internal/auth/service.go` | `604221210757b29cead30667ecccb676d49358485e4eeb3e886e667256558147` |
| `backend/internal/repository/admin_bootstrap_repository.go` | `52bb12469a52b8254c6b57a36360e138f931e11037005631455562347513fb9c` |
| `backend/internal/repository/admin_bootstrap_repository_test.go` | `d43d7581eebc556d67bf1eaca3b9872d2c46f498db3c2b2cfa267b1a0d231923` |
| `backend/internal/repository/errors.go` | `37ff496bbaf6e5c3a407533b240de56b7f962f9c8655b071fbbf0c9b75911f50` |
| `backend/internal/repository/types.go` | `4c8881145d779134fd5debb22c36f577a00801259f82eaf7561c391616dc3888` |
| `backend/internal/repository/sql/admin_bootstrap_target_by_digest.sql` | `4880c7edd566b76a446ad71d73dc826b21f58ab6e0882a5dafa986505f9dc0e5` |
| `backend/internal/repository/sql/admin_bootstrap_target_by_id.sql` | `4aca82ccbc2834c258e6c2eb90e4c5c4cfccfefb5c9071a770464643a3837063` |
| `backend/internal/security/normalizer.go` | `746b279252d939cf204a13a0d8cccf7152be650e8c0ebbae14693a92e1caeaf3` |
| `docs/design/DESIGN-006.md` | `422d0dd8909eb0fb7f9171123169c296845c39f241f5aab705307ac485a77cf6` |
| `docs/design/DESIGN-009.md` | `89a5a6377f14e83f7e842a62b2f07b036f0d5c54444abe7c941fccfbc2330331` |
| `docs/design/DESIGN-013.md` | `fd0fa9430d96123a888cc1d22ebdd49de805d372a46111d241124236fa1b78b3` |
| `docs/operations/administrator-bootstrap.md` | `2d741dc731a9fc16e1e7672b270df45fa194f0a0a15cdd8011674bc53fad4086` |
| `database/migrations/000029_admin_bootstrap_actor.up.sql` | `a978a20c60f903936f8f5f560987cbb9906aece93961dc3ae5b56cc6d7a0ce21` |
| `database/migrations/000029_admin_bootstrap_actor.down.sql` | `7beffab6246d5a42ed64def5a6ccf9de6a6b61d90458f127c297dd23286fd57d` |

## Re-review repair

### Reproduction and repair baseline

- Re-review repair baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- Preparation evidence SHA-256 before this repair: `8ef18ed262bd857db637cc25ae5f9478b0eb34efd1b834617741cc006cb03f0b`.
- Task-list SHA-256 before this repair: `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2`.
- The worktree still contained the existing Task 276 implementation and the independently changed `PREPARED` row. This repair did not edit or revert `docs/implementation/02_TASK_LIST.md`.
- Before production changes, `TestPostgresAdministratorBootstrapLifecycle` showed that shared PostgreSQL readback represented the operator audit with a zero non-nullable UUID and had no `ActorKind`.
- Before production changes, `TestLookupExactEmailReindexesLegacyMixedCaseDigest` returned an empty page because user administration queried only the canonical digest.
- Before production changes, `TestLookupExactEmailRefusesCanonicalLegacyCollision` returned the canonical account instead of rejecting two differently identified accounts.

### Re-review behavior

- `AdminAuditEntry` now carries `ActorKind` and nullable `AdminUserID`. Shared audit list SQL selects `actor_kind`, the scanner reads both values, and shared persistence validates administrator versus operator actor invariants.
- Bootstrap lifecycle coverage reads its operator audit back through `PostgresAdminImportAuditRepository.ListAuditForEntity` and asserts `ActorKind == operator`, `AdminUserID == nil`, and the bootstrap action. Existing administrator-audit PostgreSQL coverage asserts the complementary non-nil administrator actor.
- `repository.ResolveCanonicalEmailIdentity` is the single collision-safe canonical/exact-legacy resolver. Authentication delegates to it, and user administration uses it for exact email lookup.
- `PostgresAdminUserRepository.ReindexUserEmailDigest` performs the same parameterized, unique-index-protected digest update. User administration maps a final not-found result back to its historical empty exact page while propagating collision and dependency failures.
- User-administration tests cover mixed-case legacy lookup/reindex and canonical-versus-legacy different-account refusal. Repository tests cover resolver branches, real PostgreSQL reindex, and unique-index collision rejection.
- Stored Argon2 hash output is explicitly bounded to 16–64 decoded bytes. Verification uses the already parsed `KeyLength`; the sole remaining checked conversion has a narrow `#nosec G115` justification tied to the enforced 64-byte bound.

### Re-review changed paths

- `backend/internal/auth/password.go`
- `backend/internal/auth/password_test.go`
- `backend/internal/auth/service.go`
- `backend/internal/dataimporter/integration_test.go`
- `backend/internal/httpapi/admin_controller.go`
- `backend/internal/httpapi/admin_controller_test.go`
- `backend/internal/httpapi/user_admin_controller_test.go`
- `backend/internal/repository/admin_audit_security_test.go`
- `backend/internal/repository/admin_bootstrap_repository_test.go`
- `backend/internal/repository/admin_user_repository.go`
- `backend/internal/repository/admin_user_repository_test.go`
- `backend/internal/repository/canonical_email_resolver.go`
- `backend/internal/repository/canonical_email_resolver_test.go`
- `backend/internal/repository/classification_admin_repository_test.go`
- `backend/internal/repository/compliance_repository.go`
- `backend/internal/repository/manual_food_repository_test.go`
- `backend/internal/repository/postgres_repository_test.go`
- `backend/internal/repository/sql/admin_audit_insert.sql`
- `backend/internal/repository/sql/admin_audit_list_for_entity.sql`
- `backend/internal/repository/types.go`
- `backend/internal/useradmin/service.go`
- `backend/internal/useradmin/service_test.go`
- `docs/design/DESIGN-006.md`
- `docs/design/DESIGN-009.md`
- `docs/operations/administrator-bootstrap.md`
- `docs/implementation/evidence/task-276-preparation.md`

The data-importer, HTTP-controller, classification, manual-food, and audit-security test changes above are the minimum compile-time adaptations required by the now-nullable shared `AdminAuditEntry.AdminUserID`; no behavior from a later task was added.

### Re-review added or modified symbols

- Added executable symbols: `repository.ResolveCanonicalEmailIdentity`, `repository.PostgresAdminUserRepository.ReindexUserEmailDigest`, and `useradmin.Service.lookupUserByEmailDigest`.
- Modified executable symbols: `auth.PasswordHasher.VerifyPassword`, `auth.parseEncodedHash`, `auth.lookupAndReindexUserByEmail`, `repository.PostgresAdminImportAuditRepository.PersistAuditEntry`, `repository.scanAdminAuditEntry`, `repository.validateAdminAuditEntry`, `useradmin.Service.Lookup`, `useradmin.Service.lookupRequest`, and `httpapi.AdminController.transactionalMutation`.
- Added declarations: `auth.maxStoredPasswordHashBytes`, `repository.AdminAuditActorKind`, `repository.AdminAuditActorAdministrator`, and `repository.AdminAuditActorOperator`.
- Modified declarations/contracts: `repository.AdminAuditEntry` and `repository.AdminUserRepository`.
- Added tests and test symbols: `TestResolveCanonicalEmailIdentity`, `TestPostgresAdminUserReindexesLegacyDigestAndRejectsCollision`, `TestLookupExactEmailReindexesLegacyMixedCaseDigest`, `TestLookupExactEmailRefusesCanonicalLegacyCollision`, and `resolverIdentity`.
- Modified tests and fixtures: `TestPostgresAdministratorBootstrapLifecycle`, `TestPostgresComplianceAndAdminRepositories`, `TestPostgresComplianceAndAdminRepositoryValidationAndErrors`, `TestIsUsablePasswordCredentialMatchesAuthenticationParser`, `TestLookupProjectsOnlyApprovedFieldsWithBoundedPaginationAndAudit`, `TestLookupExactEmailNormalizesDigestAndAuditsEntity`, `memoryAdminUsers.LookupAdminUsers`, `memoryAdminUsers.ReindexUserEmailDigest`, and `recordingDigester.DigestForWrite`.
- Mechanical nullable-actor adaptations modified the existing audit tests/callers in the exact changed paths above; their executable behavior is otherwise unchanged.

### Re-review verification

| Command | Result |
|---|---|
| Pre-fix focused PostgreSQL audit readback test | FAIL as expected: shared readback exposed zero `AdminUserID` and no actor-kind field. |
| Pre-fix focused user-administration mixed-case/collision tests | FAIL as expected: legacy result was missed and collision returned one account. |
| `go test -count=1 ./internal/repository -run 'TestResolveCanonicalEmailIdentity\|TestPostgresAdminUserReindexesLegacyDigestAndRejectsCollision\|TestPostgresAdministratorBootstrapLifecycle'` (`backend/`) | PASS. |
| `go test -count=1 ./...` (`backend/`) | PASS across every backend command and package after the re-review repair. |
| `go test -count=1 -race ./internal/auth ./internal/repository ./internal/useradmin ./internal/httpapi` (`backend/`) | PASS with no race report. |
| `go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-dir=.go-mod-cache ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/repository ./internal/security ./internal/seed ./internal/useradmin` (`backend/`) | PASS: 41 files, 9,492 lines, one narrowly justified suppression, zero issues. Both reported G115 findings are resolved. |
| `go test -count=1 -coverprofile=/tmp/task-276-rereview.cover ./internal/auth ./internal/repository ./internal/useradmin` (`backend/`) | PASS: auth 98.8%, repository 85.3%, useradmin 93.1%. Password parser/verification and auth resolver report 100%; package totals include pre-existing non-Task-276 surfaces. |
| `python3 scripts/check.py --quick` | Initial re-review run found one missing adjacent traceability comment on the new actor-kind constant block; the comment was added. Final run passed all lanes with the existing OpenAPI OAuth 302-only warning. |
| `python3 scripts/validate-traceability.py` | PASS after the traceability correction. |
| `python3 scripts/validate-phase07-go-doc.py` | PASS for Phase 07/08 exported Go Doc. |
| `python3 scripts/validate-task-list.py` | PASS; Task 276 remains externally `PREPARED`, and this repair made no task-list edit. |
| `git diff --check` | PASS. |

### Re-review risks and blockers

- Blockers: none.
- The shared model now distinguishes authenticated administrators from infrastructure operators on both write and read. Existing callers explicitly supply administrator actor kind and a non-nil UUID; bootstrap remains the only operator-origin writer.
- Exact legacy recovery is necessarily bounded to the operator/administrator-supplied trimmed spelling because an HMAC digest cannot reveal unknown historical casing. Canonical/legacy ambiguity and concurrent unique-index conflicts fail closed.
- The 64-byte hash maximum accepts current generated 32-byte Argon2id credentials and prevents oversized stored text from reaching Argon2 allocation or an unchecked integer conversion.

### Re-review final hashes

SHA-256 values were captured after final formatting. The task-list hash exactly matches both repair baselines.

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2` |
| `backend/internal/auth/password.go` | `279db34ee32ee8568cf543400d10b8458e761740866d392cf4a6c09761a0f9d4` |
| `backend/internal/auth/password_test.go` | `7746d5a1925b3b44f46e6ad11086bd29c355621bed262677d6046627317c1161` |
| `backend/internal/auth/service.go` | `ff5e75a6367a3131bec072909da3bdd5a2617dd95c85a1a79ddd0af8f52dc1c8` |
| `backend/internal/dataimporter/integration_test.go` | `f7a9d70ab6f32cad7a72de770f5de133797781396f680200f830949e3b58e623` |
| `backend/internal/httpapi/admin_controller.go` | `de38a041d80a01e091d38060eb11ef72644c9c65f4a94c0e1904c98f9f839129` |
| `backend/internal/httpapi/admin_controller_test.go` | `208b917f6feb2634d3b18bd9e31505ab262c48cec0bcd3f3717af4de61216452` |
| `backend/internal/httpapi/user_admin_controller_test.go` | `ee6696a012d57e619a516a2e3a66cc783394c18a41a6ad563ae253c8e6b5ab77` |
| `backend/internal/repository/admin_audit_security_test.go` | `b72356fdf78db783ce668651827be393e5d59232f1010409f142a0e7fc960d66` |
| `backend/internal/repository/admin_bootstrap_repository_test.go` | `a22c6b02481b06bb180ed7da0e6c4fe5140fc148065283d0ff87640ba28e2737` |
| `backend/internal/repository/admin_user_repository.go` | `cdff6df7c907b7396e1c19f99ae46e0d96c71e10e213149effefe9d8cc85f525` |
| `backend/internal/repository/admin_user_repository_test.go` | `d514da5256916f02be3cb5d1d30244290216f4f6bd44068c9e4f85e691f114d7` |
| `backend/internal/repository/canonical_email_resolver.go` | `3cedddc8cbcd425d3ce65c601f4644a79907e5cb46f697c3df1c0e68f3ac5dae` |
| `backend/internal/repository/canonical_email_resolver_test.go` | `2384b95dd02f6691353b0a258d07fd90db1056fdda96b4bbcd000976ce7aa17c` |
| `backend/internal/repository/classification_admin_repository_test.go` | `0e6bb6520fe53be0f99cdebbc513111d5dd13580cf77390c78ac790d297be8e7` |
| `backend/internal/repository/compliance_repository.go` | `4b08638b87bce3c31b149bba4e6b5269537f1e84fb253c5042261c7010ed49a3` |
| `backend/internal/repository/manual_food_repository_test.go` | `e83642d13869e416dbc0c8681412322e43d10938d3543137185e39735583184d` |
| `backend/internal/repository/postgres_repository_test.go` | `97685f5f4db8d6820ea8761d207983b360b92d47fa06e57f2c05d09a5365e241` |
| `backend/internal/repository/sql/admin_audit_insert.sql` | `5e7943af3cdfeaca439b63c55cdb9d781d36831a4743e5c476a633834fd8e71b` |
| `backend/internal/repository/sql/admin_audit_list_for_entity.sql` | `696c0d053489d90b9fe51b9f471de8023321d06558a417be90a768625ec50e78` |
| `backend/internal/repository/types.go` | `d52db09e9dd3d8caa266f52c0bc718e9d707ffc267760e30c2f0d83cee5b02af` |
| `backend/internal/useradmin/service.go` | `b91b3e899412e63fd0b333f52a8814543b4529037eeae21dfee286144d51732a` |
| `backend/internal/useradmin/service_test.go` | `8c584a3915947c314c649051075fb27a1cf26115a371b7c5309e16d5335ae44d` |
| `docs/design/DESIGN-006.md` | `6b9b7b0e33e47f15006df7a7a1c80c40b592adcb6721d9c17921f1d43dd888f6` |
| `docs/design/DESIGN-009.md` | `be8e978538a24db6d8e216fae4a5a5c8b5b5a942c29f04c2c7d6fda756c8ee94` |
| `docs/operations/administrator-bootstrap.md` | `44cbdf1886a48460ebf435f3ccf1b7f6888f607be471929810d09c3e1a3ac1ef` |

## I-2 producer/parser boundary repair

### Baseline and reproduction

- I-2 repair baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- Preparation evidence SHA-256 before I-2 repair: `9412d6e7d7186e7c5ea98f86c01f6581ed0a44c2286ff0c3098485fd2e5eeeaf`.
- Task-list SHA-256 before I-2 repair: `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2`.
- The repair preserved the existing Task 276 worktree and independently supplied `PREPARED` row. It did not edit `docs/implementation/02_TASK_LIST.md`.
- The test-first reproduction configured `KeyLength = 64`, generated and verified a valid credential, then configured `KeyLength = 65`. Before the production fix, `TestPasswordHasherKeyLengthMatchesStoredParserBoundary` failed with `NewPasswordHasher() accepted 65-byte key length`, reproducing the producer/parser inconsistency exactly.

### Repair behavior

- `NewPasswordHasher` now accepts only key lengths from 16 through `maxStoredPasswordHashBytes` (64), matching `parseEncodedHash`.
- The accepted 64-byte boundary generates an Argon2id credential that `parseStoredPasswordCredential`, `IsUsablePasswordCredential`, and `VerifyPassword` all accept.
- The rejected 65-byte boundary fails during hasher construction, before Argon2 allocation or unusable credential generation.
- No bootstrap, repository, SQL, audit, identity, HTTP, or task-list behavior changed.

### Exact changed paths and symbols

- `backend/internal/auth/password.go`
  - Modified executable symbol: `auth.NewPasswordHasher`.
- `backend/internal/auth/password_test.go`
  - Added executable test symbol: `auth.TestPasswordHasherKeyLengthMatchesStoredParserBoundary`.
- `docs/implementation/evidence/task-276-preparation.md`
  - Added this I-2 repair record.

### Verification

| Command | Result |
|---|---|
| `go test -count=1 ./internal/auth -run TestPasswordHasherKeyLengthMatchesStoredParserBoundary` (`backend/`, before fix) | FAIL as expected: the 65-byte configuration was accepted. |
| `go test -count=1 ./internal/auth ./cmd/admin-bootstrap` (`backend/`) | PASS; generated credential, authentication parser, and bootstrap eligibility composition remain compatible. |
| `go test -count=1 -race ./internal/auth ./cmd/admin-bootstrap` (`backend/`) | PASS with no race report. |
| `go test -count=1 ./...` (`backend/`) | PASS across every backend command and package. |
| `go test -count=1 -coverprofile=/tmp/task-276-i2.cover ./internal/auth` (`backend/`) | PASS at 98.8% package coverage; `NewPasswordHasher`, `HashPassword`, `VerifyPassword`, `parseStoredPasswordCredential`, and `parseEncodedHash` each report 100%. |
| `go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-dir=.go-mod-cache ./cmd/admin-bootstrap ./internal/adminbootstrap ./internal/auth ./internal/repository ./internal/security ./internal/seed ./internal/useradmin` (`backend/`) | PASS: 41 files, 9,493 lines, one narrowly justified existing suppression, zero issues. |
| `python3 scripts/check.py --quick` | PASS across static and changed-area lanes; the existing OpenAPI OAuth 302-only warning remains unchanged. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-phase07-go-doc.py` | PASS for Phase 07/08 exported Go Doc. |
| `python3 scripts/validate-task-list.py` | PASS; task-list contents and hash remain unchanged during I-2 repair. |
| `git diff --check` | PASS. |

### Risk and blocker disposition

- Blockers: none.
- Every configuration accepted by `NewPasswordHasher` now emits a decoded hash length accepted by the shared authentication/bootstrap parser.
- Existing generated credentials use 32 bytes and remain unaffected. The 64-byte upper boundary is verified end to end; 65 bytes is rejected before generation.

### I-2 final hashes

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `0f7b4c2dbb92d841fce440dd0152a7c1f8ad7d6c822f7fcfbf7dc5e331c613c2` |
| `backend/internal/auth/password.go` | `18104445448c2622f3639057e5d240a83ea1d0447204c862ee76fb1fadc8678f` |
| `backend/internal/auth/password_test.go` | `4a0ededdf1db03a5a723797df8879f3a9133692223e589921b07cfcfd5c31fd9` |
