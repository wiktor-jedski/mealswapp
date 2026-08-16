# Task 275 Preparation — Phase 08.01 Acceptance Evidence Refresh

## Outcome and task control

- Task: **275 — Phase 08.01 Acceptance Evidence Refresh** (`DESIGN-009: AdminController`).
- Role followed: `/home/wiktor/.codex/agents/developer.toml`.
- Phase-completion workflow followed: `/home/wiktor/.agents/skills/phase-completion/SKILL.md`.
- Dependency 274 was consumed from its complete preparation evidence and current aggregate report/screenshots. Task 275 did not rerun or claim ownership of the aggregate gate.
- `docs/implementation/02_TASK_LIST.md` was read only. Task 275 remains `OPEN`; no task status was edited.
- No other agent was messaged.
- The shared worktree already contained concurrent Phase 08.01 API, backend, frontend, script, architecture, task-list, open-action, report, screenshot, preparation, review, and integration-obligation changes. They were preserved. No reset, checkout, clean, staging, commit, dependency update, generated output rewrite, migration, report regeneration, screenshot overwrite, or unrelated formatting was performed.

## Exact Task 275 changed paths

| Path | Task 275 delta |
|---|---|
| `docs/implementation/implemented/08_PHASE_UAT.md` | Replaced stale Task 262/historical-remediation framing with the current post-review acceptance record; traced Tasks 238-275; distinguished original Tasks 238-263 from Phase 08.01 Tasks 264-275; recorded current Task 274 commands, report/screenshots, coverage, remediation evidence, and 17 unchecked project-owner checks. |
| `docs/implementation/preparations/task-275.md` | Added this task-owned scope, source, command, validator, artifact, coverage, risk, and acceptance evidence. |

Task 274 owns the already-refreshed `08_PHASE_REPORT.html`, its current coverage-summary repair, and the 20 `08_PHASE_REPORT-*.png` artifacts. Task 275 references and integrity-checks those concurrent dependency artifacts without modifying them.

## Sources inspected

- Phase 08 in `docs/implementation/01_PLAN.md`; Tasks 238-275 in `docs/implementation/02_TASK_LIST.md`; Phase 08 assumptions, exact coverage contracts, review-action dispositions, and the Task 275 deferral in `docs/implementation/04_OPEN.md`.
- `/home/wiktor/.codex/agents/developer.toml` and the complete phase-completion skill.
- `docs/architecture/ARCH-009.md`, `docs/architecture/ARCH-012.md`, and the relevant static aspects in `DESIGN-001`, `DESIGN-002`, `DESIGN-005`, `DESIGN-008` through `DESIGN-015`, and `DESIGN-017`.
- `SW-REQ-019`, `SW-REQ-033`, `SW-REQ-043`, `SW-REQ-054` through `SW-REQ-057`, `SW-REQ-072`, `SW-REQ-073`, `SW-REQ-084`, and `SW-REQ-090`.
- Preparations for Tasks 262-274, especially current remediation/gate evidence for Tasks 264-274.
- `docs/testing/integration/ARCH-009-obligations.md` and `ARCH-012-obligations.md`, including remediation obligations `IT-ARCH-009-008` through `IT-ARCH-009-011` and `IT-ARCH-012-004`.
- The prior `08_PHASE_UAT.md`, current Task 274 aggregate report, and all 20 report screenshots.
- `tools.md`, named by the phase-completion workflow, does not exist in this repository. Repository commands were taken from `AGENTS.md`, the task contract, package scripts, and dependency evidence.

## Refreshed acceptance coverage

The UAT now supplies all Task 275-required owner checks while leaving every result unchecked:

1. cross-instance Catalog and Substitution Search visibility after committed manual-item create/rename/delete, with exact replay/failure/rollback non-invalidation;
2. recursive duplicate top-level, `macrosPer100`, and micronutrient JSON rejection on private create/update before dispatch;
3. monotonic import ownership, immutable draft/key snapshots, disabled incompatible controls, stale completion rejection, explicit Keep editing/Discard draft and search, and visible focus restoration;
4. fail-closed Account Export loading/failure, accepted-deletion verification-required feedback, success only after authoritative refresh, and owner-safe retry;
5. destructive-action contrast in light/dark themes, standard 200 ms transitions, reduced-motion opt-out, Surface/Border/Primary form focus styling, and heading-only Bold 700;
6. the original authorization, curation, density, manual/classification administration, private isolation/export/erasure, audit, provider degradation, idempotency, accessibility, and core regression acceptance paths.

The final acceptance decision remains:

`Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations`

No owner name, date, result, deviation, or acceptance claim was inferred.

## Current inherited aggregate evidence

Task 274 ran `python3 scripts/check.py --output docs/implementation/implemented/08_PHASE_REPORT.html` after all approved remediation implementation. Its complete preparation records:

- aggregate **PASS**, exit 0, across static, backend, frontend, and browser lanes;
- requirements `91/91`, task-list/traceability, OpenAPI, all 24 generator tests, generated drift, Go Doc/TSDoc, formatting, vet, vulnerability, race, live PostgreSQL/Redis integration, coverage contracts, typecheck, build, unit tests, Playwright, axe, report, and screenshot evidence;
- frontend 533 tests, 2,789 expectations, and a 219-module production build;
- browser 309/314 with five intentional configured skips and no failures;
- final `python3 scripts/check.py --quick` **PASS** after Task 274 edits.

Task 275 does not present historical Task 262 values as current and does not pretend to rerun Task 274.

## Current coverage disposition

- Backend direct aggregate: `87.5%`.
- Machine-defined Phase 08 backend scope: `4,537/4,849` statements (`93.6%`), with exact below-100 source coordinates and `B1`–`B4` reasons machine-checked in `04_OPEN.md`.
- Frontend aggregate: `95.46%` functions and `96.06%` lines. Current Phase 08 below-100 runtime rows are `admin-workflows.ts`, `account-data-client.ts`, `admin-client.ts`, and generated `generated.ts` line 185 under `F1`–`F3`.
- Svelte component behavior is dispositioned by focused component tests plus desktop/mobile Playwright/axe because Bun emits no component runtime rows.
- Task 275 adds no runtime code, coverage source, exception, waiver, or accepted deviation.

No authorization, ownership, private/global isolation, CSRF, idempotency/replay, duplicate-key rejection, atomic audit rollback, post-commit invalidation, cross-instance visibility, import ownership, Account Export fail-closed state, generated-contract drift, accessibility, or browser workflow is waived.

## Commands actually run for Task 275

Commands ran from the repository root on 2026-07-25. Inspection and hashing were read-only; the only Task 275 writes used `apply_patch` for the two paths listed above.

| Command | Exact result |
|---|---|
| `python3 scripts/validate-task-list.py` | **PASS:** `Task-list validation passed: 275 sequential tasks with ordered dependencies.` |
| `python3 scripts/validate-traceability.py` | **PASS:** `Traceability validation passed.` |
| Focused Task 275 evidence-integrity Python validator shown below | **PASS:** `Task 275 evidence integrity passed: tasks 238-275, 17 unchecked owner checks, unchecked decision, current coverage, report, 20 PNGs, and local links.` |
| `git diff --check -- docs/implementation/implemented/08_PHASE_UAT.md docs/implementation/preparations/task-275.md` | **PASS:** exit 0, no output. |
| `python3 scripts/check.py --quick` | **PASS:** exit 0 after both Task 275 documents existed. Static checks passed requirements, task list, traceability, OpenAPI with the accepted OAuth warning, 24 generator tests, Go Doc/TSDoc, formatting, vet, vulnerability, generated drift, and contract/process tests. Changed-area checks passed 533 frontend tests/2,789 expectations, typecheck, 34 desktop/mobile remediation Playwright cases, and eight changed backend packages; total changed-area runtime `57.3s`. |
| `sha256sum docs/implementation/implemented/08_PHASE_UAT.md docs/implementation/implemented/08_PHASE_REPORT.html` | **PASS:** exact hashes recorded below. |
| Sorted 20-file screenshot SHA-256 manifest piped to `sha256sum` | **PASS:** exact manifest digest recorded below. |
| `stat -c '%n %s bytes'` over the UAT and report | **PASS:** UAT `24,231` bytes; report `778,773` bytes. |
| `rg -n '^\| 275 \|' docs/implementation/02_TASK_LIST.md` | **PASS:** exactly one live Task 275 row, status `OPEN`, dependency `274`; no mutation. |

The exact focused validator was:

```python
from pathlib import Path
import re

uat = Path("docs/implementation/implemented/08_PHASE_UAT.md")
text = uat.read_text()
for task in range(238, 276):
    assert len(re.findall(rf"^\| {task} \|", text, re.MULTILINE)) == 1, task
assert text.count("| ☐ |") == 17
assert "Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations" in text
assert "4,537/4,849" in text and "4,523/4,841" not in text
assert "533 tests and 2,789 expectations" in text
report = uat.parent / "08_PHASE_REPORT.html"
assert report.is_file() and "QUALITY GATE PASSED" in report.read_text()
shots = sorted((uat.parent / "screenshots").glob("08_PHASE_REPORT-*.png"))
assert len(shots) == 20
assert all(path.read_bytes().startswith(b"\x89PNG\r\n\x1a\n") for path in shots)
for target in re.findall(r"\[[^]]+\]\(([^)]+)\)", text):
    assert (uat.parent / target).resolve().exists(), target
print(
    "Task 275 evidence integrity passed: tasks 238-275, 17 unchecked owner checks, "
    "unchecked decision, current coverage, report, 20 PNGs, and local links."
)
```

## Artifact evidence

| Artifact | Exact evidence |
|---|---|
| `docs/implementation/implemented/08_PHASE_UAT.md` | `24,231` bytes; SHA-256 `371361f141ec2fe883e448e21480a63ddf9e8fa1d37910ea7fd9209bf511a9c0` |
| `docs/implementation/implemented/08_PHASE_REPORT.html` | `778,773` bytes; SHA-256 `a64512d4c91da9aa6fcab56a783a2d528c850105d974badaec29f285fa772526`; contains `QUALITY GATE PASSED` and current `4,537/4,849` coverage summary |
| 20 sorted `08_PHASE_REPORT-*.png` SHA-256 lines hashed as one manifest | SHA-256 `7092531b77ca09dbe01dcc137882bad51e5d33bbc29a6212488b8181187fa176` |
| `docs/implementation/02_TASK_LIST.md` (read only) | SHA-256 `df3d5e00cc4bee3c231f078395850fab0b0ec5d377ab71cd2c7c93f2afba8408` |
| `docs/implementation/04_OPEN.md` (read only) | SHA-256 `4cdf7575ed7a2a1616c347482844f15e06755ae64866ee3a96e53af2558b1a99` |
| `docs/testing/integration/ARCH-009-obligations.md` | SHA-256 `d404bc894c499861d3a62c38e5f678d3c4c08cddfbf75ad257542e236d0a9d2a` |
| `docs/testing/integration/ARCH-012-obligations.md` | SHA-256 `01c6213f92f9d931da5aeb2db5801c37b8de4c9dc0cc536a13cf9057d63e562a` |

The preparation file omits its self-referential hash. Artifact hashes were captured immediately after the focused validators. A later concurrent change to a hashed dependency artifact requires the affected evidence to be refreshed.

## Residual actions and readiness

- Project-owner UAT execution remains the sole acceptance action represented by this document. All 17 result cells and the final decision are unchecked.
- The accepted Redocly OAuth callback warning, five intentional browser skips, and exact machine-checked coverage exceptions remain disclosed.
- Phase 09 production/platform actions remain outside Task 275.
- The UAT is ready for project-owner execution. It does not claim that Phase 08 has been accepted.
- No task-list status was edited and no other agent was contacted.
