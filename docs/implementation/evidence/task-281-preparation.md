# Task 281 preparation evidence

prepared_at_utc: 2026-07-28T06:02:54Z
task_id: 281
task_status_observed: PREPARED
component: Phase 08.02 Administrative Authorization Acceptance Scenarios
static_aspect: DESIGN-009 AdminController
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677
baseline_confidence: HIGH
requirement_result: FAIL
test_delivery_result: COMPLETE

## Outcome

Task 281 now owns isolated real-stack SW-REQ-054 browser and API scenarios for
anonymous, ordinary-user, verified-administrator, malformed-session,
stale-session, and spoofed-identity contexts. The suite runs against one
disposable PostgreSQL database, one run-labeled Redis container, the production
API composition, the production Vite frontend, generated frontend clients, and
Chromium desktop/mobile projects. No route interception is used.

The final run executed all nine SW-REQ-054 manifest entries. Seven passed. Two
failed for one product root cause:

- `P08-SWR054-STEP-04` and `P08-SWR054-ACCEPT-04` failed because a browser
  holding ordinary-user access/refresh claims before bootstrap receives an
  administrator refresh projection and reaches `/admin` after bootstrap
  without sign-out and credential reauthentication.
- The old access cookie still receives `403` from the administration API, so
  the elevation occurs specifically through refresh-session claim
  reconstruction.
- The root cause is synchronized as open defect `P08-FIND-281-002`,
  `ROOT-T281-STALE-REFRESH`, in the Phase 08 ledger and mandatory history.

The fresh Task 280 report is
`logs/phase08-acceptance/task281-325cf05bb43ce884b9bb89d4/report.json`.
It records `FAIL`, exit code `1`, nine criteria, seven passes, two failures, one
grouped root cause, linked request IDs, retained desktop/mobile screenshots,
and sanitized backend evidence. This is the required truthful requirement
outcome; no scenario was skipped, weakened, mocked, or converted to pass.

Task 281 is a test-delivery task. Its implementation and synchronization are
complete even though SW-REQ-054 remains unaccepted until
`P08-FIND-281-002` is repaired and retested. The authoritative task-list status
was not edited.

## Review repair

All findings in the latest rejected `task-281-review.md` are repaired while
the earlier evidence, product result, and infrastructure root are preserved:

- Task mode now requires a private, nonce-bound capability manifest generated
  by the current harness. The config verifies the exact loopback base URL,
  Task 281 evidence root, live frontend PID/start token, frontend working
  directory, and strict reserved port before loading tests. Missing capabilities,
  foreign base URLs, non-loopback origins, shared listeners, and mismatched
  evidence roots fail closed.
- Reporter failures before `recordAcceptance`, malformed attachments, missing
  or partial phase files, unknown roots, and malformed result fields normalize
  to `ROOT-T281-ACCEPTANCE-INFRASTRUCTURE`. The combiner always emits exactly
  nine sanitized rows. Task 280 publication is verified, and publication
  failure re-enters the same structured fallback with a nonzero exit.
- F-281-005 is repaired at both producer and consumer boundaries. The reporter
  now rejects malformed values for every attachment field, including
  `evidence[].type=[]`; the runner uses explicit string guards before set
  membership for status, root cause, and evidence type. A focused regression
  proves reporter rejection, injects the malformed phase artifact directly
  to exercise `_safe_evidence`, and verifies Task 280 publishes nine
  synchronized BLOCKED rows with exit `2`.
- The prior browser-attachment repair remains intact.
  PostgreSQL assertions require exactly one classification and one matching
  `classification.create` audit per configured browser project, all audits
  must belong to the exact bootstrapped fixture UUID, and Redis must return
  `PONG`. The persistent backend artifact records the expected count and
  `validated=true`.
- The prior lifecycle repair remains intact: timeout, spawn, setup, bootstrap,
  backend-evidence, and cleanup failures
  synthesize all nine results under grouped root
  `ROOT-T281-ACCEPTANCE-INFRASTRUCTURE`, finalize the requirement report after
  cleanup, and return `1` for FAIL or `2` for BLOCKED. Focused tests exercise
  timeout, bootstrap, count, actor, Redis, producer, and evidence failure
  reports.

The grouped infrastructure root is synchronized as open blocker
`P08-FIND-281-003`; it appears only when trustworthy execution is prevented.
The successful infrastructure path in the fresh run has no infrastructure
root. Product defect `P08-FIND-281-002` remains unchanged and truthfully causes
the same two SW-REQ-054 failures.

## Baseline and preserved worktree

- Repository: `/home/wiktor/Work/mealswapp`
- Branch: `multistep-phase-08`
- HEAD: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Initial worktree: dirty with Tasks 276-280 and other Phase 08 work. All
  unrelated modifications and untracked files were preserved.
- Authoritative row: Task 281, `OPEN`, dependencies 276/279/280 observed
  `PASSED`.
- Sources read completely or at the relevant authoritative sections:
  Task 281 and Task 280 rows, Task 279/280 preparation and review evidence,
  `req_tests.md`, SW-REQ-054, DESIGN-009, the acceptance manifest/finding
  history, Task 279 harness, authentication/session middleware and service,
  administrator route/controller composition, generated admin client,
  administration shell/panel, and existing intercepted Playwright tests.
- Confidence is **HIGH** for the delivered test behavior and reported defect:
  the same stale-refresh elevation reproduced on desktop and mobile in
  repeated isolated runs, while every fresh-login administrator scenario
  passed. Confidence is **HIGH** that fixture cleanup completed because the
  final state is `cleaned` and stale cleanup found zero resources.

## Exact changed paths and executable symbols

### Task 280 requirement-report reuse

- `scripts/phase08_acceptance.py`
  - changed `command_report` to filter a fully validated manifest to one
    requested requirement;
  - changed `build_parser` to add `report --requirement`.
- `scripts/test_phase08_acceptance.py`
  - added
    `test_requirement_report_accepts_one_complete_manifest_scenario`;
  - made history/ledger regressions work with the now non-empty mandatory
    finding history.
- `docs/testing/phase08/acceptance-manifest.json-trace.md`
  - documents requirement-specific report finalization without weakening
    global manifest or finding validation.

### Browser/API acceptance producer

- `frontend/playwright.real-stack.config.ts`
  - adds the Task 281 reporter only for managed Task 281 runs;
  - adds the mobile Chromium project only for Task 281;
  - validates a private harness-generated capability against the exact
    loopback origin, evidence root, and live managed frontend process.
- `frontend/tests/phase08-acceptance-reporter.ts`
  - added `Phase08AcceptanceReporter`;
  - added `splitEnvironment`, `uniqueEvidence`, and `isRecord`;
  - aggregates both projects, preserves failures, blocks missing executions,
    groups synchronized root causes, maps missing/malformed attachments to the
    infrastructure root, validates every runtime attachment field, and writes
    the Task 280 producer schema.
- `frontend/tests/task281-acceptance-helpers.ts`
  - added `SafeEnvelope`, `fixture`, `safeEnvelope`, `signIn`,
    `openSidebarForControl`, `screenshot`, `recordAcceptance`, and
    `staleStatePath`.
- `frontend/tests/task281-prebootstrap.spec.ts`
  - added independently named anonymous and ordinary-user direct-navigation,
    documented-read, mutation, spoofing, DOM-leakage, correlation, and
    pre-bootstrap session-state scenarios.
- `frontend/tests/task281-postbootstrap.spec.ts`
  - added independently named malformed-session, stale-claim,
    documented/undocumented administrator API, spoofed body/header,
    sign-out/fresh-sign-in, generated-client mutation, responsive grid,
    keyboard, theme, and axe scenarios.
- Both Task 281 specs skip in the ordinary mocked Playwright gate and execute
  only under the managed isolated harness; the real-stack run does not skip
  either project.

### Isolated lifecycle and evidence

- `scripts/run-task281-acceptance.py`
  - added `PRE_CRITERIA`, `POST_CRITERIA`, `PROJECTS`,
    `RESOLVED_MOBILE_NAVIGATION_ROOT`;
  - added `Task281Harness`, `Task281Harness.execute`,
    `run_playwright_phase`, `write_backend_evidence`, and `combine_results`;
  - added `parse_args`, `safe_playwright_diagnostics`, and `main`;
  - runs the pre-bootstrap browser/API phase, Task 276 bootstrap, and the
    post-bootstrap phase in that order;
  - retains storage state only in Task 279's disposable raw directory;
  - reads PostgreSQL mutation/audit counts, checks Redis reachability, invokes
    Task 280 reporting, propagates criterion exit code `0/1/2`, and leaves
    cleanup unconditional;
  - writes a private scoped capability for the exact managed listener and
    evidence root, normalizes malformed/partial producers to nine sanitized
    rows with non-throwing type guards, and verifies Task 280 actually
    published the synchronized report.
- `scripts/test_task281_acceptance.py`
  - covers exact nine-criterion output, fail-closed missing phases, foreign
    base rejection, failure before acceptance attachment, lifecycle fallback,
    malformed attachment producer/consumer boundaries, synchronized nonzero
    report publication, bootstrap ordering, and read-only database evidence.
- `scripts/check.py`
  - registers Task 281 contract tests in the Phase 08 acceptance lane;
  - registers the runner/test files for traceability.

### Finding synchronization

- `docs/implementation/04_OPEN.md`
  - added and then closed `P08-FIND-281-001` for the initial mobile sidebar
    test-helper blocker, retaining original and passing evidence;
  - added open product defect `P08-FIND-281-002` for stale-refresh privilege
    elevation.
- `docs/testing/phase08/finding-history.json`
  - retains both finding IDs in sorted append-only history.

No backend production code, OpenAPI contract, migration, application frontend
component, requirement text, later task, or task-list status was changed.

## Criteria coverage

| Manifest criterion | Result | Real-stack evidence |
|---|---|---|
| `P08-SWR054-STEP-01` anonymous `/admin` | PASS | Desktop/mobile direct navigation denies the panel/control; documented read and mutation return 401. |
| `P08-SWR054-STEP-02` ordinary `/admin` | PASS | Desktop/mobile direct navigation denies the panel/control and persists two independent ordinary session contexts for bootstrap checks. |
| `P08-SWR054-STEP-03` representative endpoints | PASS | Documented read succeeds only for fresh admin; documented mutation uses the generated client; undocumented read/mutation return 404. |
| `P08-SWR054-STEP-04` sign in as admin and repeat | FAIL | Fresh sign-in passes, but stale refresh grants admin before required sign-out/sign-in (`ROOT-T281-STALE-REFRESH`). |
| `P08-SWR054-ACCEPT-01` anonymous 401 | PASS | Read and mutation return safe 401 envelopes with server request IDs and no data. |
| `P08-SWR054-ACCEPT-02` ordinary 403 | PASS | Read and mutation return safe 403 envelopes before spoofed body/header identity can help. |
| `P08-SWR054-ACCEPT-03` admin success | PASS | Fresh admin read is 200; generated-client mutations are 201 on both viewports; PostgreSQL has two rows and two matching administrator audits. |
| `P08-SWR054-ACCEPT-04` direct navigation leaks nothing | FAIL | Anonymous/ordinary direct navigation passes, but refresh elevates the stale ordinary browser and reveals the restricted panel. |
| `P08-SWR054-ACCEPT-05` spoofing does not help | PASS | Role/user/request headers do not authorize anonymous/ordinary contexts; unknown role/user body fields are rejected; malformed cookies return 401. |

Additional assertions prove:

- every response correlation ID differs from the client-supplied request ID;
- the documented route allowlist rejects both an undocumented read and
  mutation;
- denied envelopes contain no restricted data and denied DOM contains no
  administration panel/control;
- fresh administrator UI behavior is consistent across one-column mobile and
  three-column desktop layouts;
- navigation is keyboard-operable, light/dark themes persist, and axe reports
  no serious/critical WCAG 2/2.1 A/AA violations;
- PostgreSQL evidence reports and asserts `classificationMutationCount=2`,
  `classificationAuditCount=2`, and two matching audit rows for the exact
  fixture administrator;
- Redis evidence asserts `PONG` and reports `reachable`;
- final harness state is `cleaned`, and age-bounded stale cleanup reports zero
  residual resources.

## Finding history

### `P08-FIND-281-001` — closed test-infrastructure blocker

The first run used desktop-visible authentication/navigation assumptions in
the mobile project. It produced truthful non-pass results rather than skips.
The helper now opens the responsive sidebar before using hidden controls and
waits for authentication completion. The next run passed all pre-bootstrap
desktop/mobile criteria. The finding is closed, retained in history, and
`P08-SWR054-STEP-02` carries the resolved root in the final report.

### `P08-FIND-281-002` — open product defect

The pre-bootstrap access cookie remains ordinary and returns 403. During SPA
startup, however, `POST /api/v1/auth/refresh` reconstructs the session from
current database role state and returns administrator claims. The same browser
then remains on `/admin` and can see the restricted panel without a credential
sign-in. This reproduces on both configured viewports. Fresh explicit
sign-out/sign-in behavior passes, so the defect is isolated to bootstrap-time
refresh claim freshness.

## Security review

The installed `golang-security` guidance was applied to the authentication and
authorization trust boundaries.

- Attacker-controlled headers/body fields are sent deliberately and never used
  as authority. Assertions require server-generated request IDs.
- Browser storage state contains cookies and is written only below Task 279's
  `TemporaryDirectory`; it is deleted before the run returns and is never
  linked into acceptance evidence.
- Fixture email/password values exist only in child-process environment and
  form input. Persistent JSON, HTML, screenshots, logs, and backend evidence
  contain neither.
- Persistent screenshots target only the denied notice or administration
  panel, not the sidebar/profile identity surface.
- Database evidence uses fixed read-only `SELECT count(*)` statements against
  the exact owned `_test` database. No role update, fixture deletion, truncate,
  broad cleanup, or unvalidated target was added.
- Cleanup remains Task 279's exact database-comment, Redis-label, and
  process-token guarded teardown.
- Failed Playwright diagnostics retain only bounded test titles/assertion
  summaries and redact the run ID, credential fixture values, and UUIDs.
- The Task 280 sanitizer, evidence-copy, append-only finding-history, and
  non-pass synchronization checks remain authoritative.

No new production vulnerability was introduced. The observed stale-refresh
elevation is recorded as a functional security defect and was not repaired
inside this test-delivery task.

## Commands and observed results

All commands ran from the repository root on 2026-07-28 unless noted.

| Command | Result |
|---|---|
| `python3 scripts/run-task281-acceptance.py --timeout-seconds 240` | Expected exit 1. Fresh managed-capability run `325cf05bb43ce884b9bb89d4`: pre-bootstrap passed; post-bootstrap ran eight project tests with six passes and the preserved stale-refresh assertion failing on desktop/mobile; exact PostgreSQL/audit and Redis assertions passed; cleanup and atomic report finalization completed. |
| Fresh report `task281-325cf05bb43ce884b9bb89d4` | Expected FAIL/exit 1: nine criteria, 7 PASS, 2 FAIL, 0 BLOCKED, only `ROOT-T281-STALE-REFRESH`; no infrastructure root. |
| Report integrity Python assertion | PASS: status FAIL, exit 1, nine criteria, exact two failures, one root, every evidence link present. |
| Backend/harness assertion | PASS: mutation/audit/fixture-actor counts all 2, expected count 2, Redis reachable, `validated=true`, final state `cleaned`. |
| `python3 scripts/run-real-stack-e2e.py --cleanup-stale --stale-age-seconds 900` | PASS, `resources=0`. |
| `python3 -m unittest -v scripts/test_phase08_acceptance.py scripts/test_task281_acceptance.py` | PASS, 35 tests. The new F-281-005 regression rejects malformed reporter evidence, directly exercises `_safe_evidence` with `evidence[].type=[]`, and publishes nine synchronized BLOCKED rows with exit 2. Prior foreign-base, pre-record, lifecycle, backend, and truthful result tests remain passing. |
| `python3 -m py_compile ...` for Task 280/281/check scripts | PASS. |
| `cd frontend && bun run typecheck` | PASS. |
| `cd frontend && bun run build` | PASS, 219 modules transformed. |
| Focused `go test ./internal/httpapi -run 'TestAdmin\|TestRequireAdmin\|TestAuthSession' -count=1` | PASS. |
| Same focused Go test with `-race` | PASS. |
| `python3 scripts/phase08_acceptance.py validate` | PASS, 12 scenarios and 91 criteria; current ledger/history valid. |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/check.py --quick` | PASS in 78.4 seconds after the F-281-005 repair. Static lane and 35 Phase 08 contract tests passed; 536 frontend tests passed; changed Playwright lane passed with 24 passes and 14 intentional real-stack skips; changed backend packages passed; OpenAPI retained its known redirect-only warning; vulnerability scan found no called vulnerabilities. |
| `git diff --check` | PASS. |

The first quick run failed because the newly added real-stack specs had not yet
declared their managed-harness skip guard. That test-delivery defect was fixed;
the exact quick command then passed. Ordinary Playwright skips are supporting
gate behavior only—the managed real-stack execution above ran every Task 281
scenario on both viewports.

## Current SHA-256 hashes

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `b25eed9787cf8f855ed01ca167d5ae739fb6108740197376cefe88aec2a5d536` |
| `req_tests.md` | `241779579f568eecfe40a8d41b97ec2b6a87e6201bd25b8ea76c4bac5c2a8902` |
| `docs/design/DESIGN-009.md` | `704500175a5e465ee6d10cc1edd421f4c7c6043dfff1cd8c444a28e1d75973ea` |
| `scripts/phase08_acceptance.py` | `b36ea8188e3e6d7e05ae17dc992c84cf493c9ac78a8a85706d51e7e40ca34bdb` |
| `scripts/test_phase08_acceptance.py` | `58d1beec596acb4006174d51c1927297220dbcd9b42a1e7bbdac4521e628ce27` |
| `docs/testing/phase08/acceptance-manifest.json-trace.md` | `a87505099f762aac4432cc00b3516f0758f4ac7b520c95a7cdc0eb2ad1b7c5c3` |
| `frontend/playwright.real-stack.config.ts` | `7f7371279c66f88b6b8ee36b8749edfff8fc1662d896123f3d899eb4fbd19628` |
| `frontend/tests/phase08-acceptance-reporter.ts` | `cc601b14c3dddfb588d7a907ce24e635a20e2985a5c252e069ca04e25f2e30b7` |
| `frontend/tests/task281-acceptance-helpers.ts` | `49877745607f94497a828de43d0c2d86524de84d665db03342f035f7d480594f` |
| `frontend/tests/task281-prebootstrap.spec.ts` | `080441b5fecdc7a4d837680b5e219edc8cd2964e13364e6d25a8fdb719c89048` |
| `frontend/tests/task281-postbootstrap.spec.ts` | `da4ee4c379e02656bdbd6465e9ab903003be4cea9eb092520d7729a4989d9aa7` |
| `scripts/run-task281-acceptance.py` | `30f4a17374209667bfe78988413d261658c0f97518a71e33fdf8405ea32ebdf1` |
| `scripts/test_task281_acceptance.py` | `0499fe3a8385530adea3bfed2deb5ffc03c46b2be429546ac882fb54bf33461f` |
| `scripts/check.py` | `cf338e08b99979d4b7d8ffa08023833203d9a3696d64ba6b185240e2cd634519` |
| `docs/implementation/04_OPEN.md` | `e80965330e09f778545442caa4c216c6b8f58e56480502549ff450a5199029f0` |
| `docs/testing/phase08/finding-history.json` | `c11c95c59a400acdaec0f887583c55b6650e754da5e77bad560e7df9f60e8ef6` |
| Fresh backend evidence | `14b53b5224d0f7beb1b0bec47817b15063890a8c134c47f257ec357ccb36fc67` |
| Final raw criterion result | `bcb8d4b92a52a2ae66bdea932207d285a731eb8c7ccabeb584dd2e676c3dbf6e` |
| Final Task 280 report | `ac113eccdc6914eb7149ac57ff6a2b579637f5a3f84a31512b9b6b3cc72f93af` |

## Residual risks and retest condition

- SW-REQ-054 is not accepted while `P08-FIND-281-002` remains open. Repair must
  preserve signed ordinary claims for existing access and refresh sessions
  across bootstrap, then require explicit credential reauthentication before
  administrator claims are issued.
- The real-stack acceptance suite intentionally returns `1` while that defect
  remains. A green test-delivery review must not reinterpret the requirement
  report as PASS.
- Task 279's listener handoff and `/proc` ownership risks remain as documented
  in Task 279 evidence; Task 281 adds no broader cleanup authority.
- Persistent evidence is sanitized rather than forensic-complete. Raw browser
  traces and cookie state are intentionally destroyed at teardown.
