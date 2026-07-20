# Task 235 Preparation Evidence

## Scope and preservation

- Task: **235 — Phase 07.01 Coverage and Aggregate Quality Gate**.
- Baseline and current `HEAD`: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Preparation date: 2026-07-18 (Europe/Warsaw).
- Task status remained `OPEN`; this work did not edit `docs/implementation/02_TASK_LIST.md`.
- The initial worktree contained cumulative Phase 07.01 implementation, tests, reviews, and unrelated user-owned changes. Nothing was cleaned, reverted, staged, or rewritten. This rejected-finding repair changes only `scripts/check.py`, `docs/implementation/04_OPEN.md`, and this evidence file; Task 235 status remains untouched.
- Read inputs: the Task 235 row; Phase 07.01 plan and repository validation instructions; `docs/implementation/04_OPEN.md`; all preparation and independent review evidence for Tasks 213-234; `scripts/check.py`; the contract, traceability, backend, frontend, browser, static, race, security, coverage, integration, and observability/capacity commands referenced by those sources. The phase-completion skill's requested `tools.md` is absent from this repository, so the repository commands in `AGENTS.md` and `scripts/check.py` are authoritative.

## Aggregate outcome

The first aggregate run reached the coverage gate after every preceding check passed, then failed because the Task 212 Phase 07 coverage percentages no longer matched the remediated source. No production failure was found. Task 235 refreshed the accepted deviations from current measurements and reran the full gate successfully.

The final aggregate result is **PASS**:

- requirements/design traceability, task-list integrity, and Phase 07 Go Doc validation passed;
- OpenAPI lint passed with the already accepted OAuth callback `302`-only warning; all 16 deliberate API-generation drift tests and generated output checks passed;
- local PostgreSQL/Redis migrations, API/worker health/readiness, Phase 02/03 compatibility UAT, and focused Phase 07 repository/API/queue/worker/CLP integration passed;
- backend formatting, full tests, race detector, `go vet`, and `govulncheck` passed;
- frontend API drift, typecheck, production build, 438 unit tests, and coverage passed;
- deterministic Chromium verification produced desktop/mobile screenshots; focused browser suites passed 70 and 28 tests; the complete Playwright/axe suite passed 231 tests with 3 suite-defined intentional skips;
- the dedicated Phase 07.01 observability/capacity gate passed its 10 Python tests and all selected normal/race Go tests, including real Redis restart, bounded cleanup, durable publication, privacy-safe telemetry, and real child-process timeout checks;
- all 22 independent review evidence files for Tasks 213-234 passed structural validation.

## Review-action audit

The source command against commit `a4e31367485b03269e90b5607f2057c9568bb5b1` found exactly 50 Phase 07 `OPEN REVIEW ACTION` entries. The current Phase 07 section contains zero `OPEN REVIEW ACTION` entries and exactly 50 dated, evidenced `IMPLEMENTED` dispositions. Every entry preserves its original owner and now cites its implementing task preparation plus independent PASSED review.

| # | Review action | Disposition / owner | Evidence |
|---:|---|---|---|
| 1 | Daily Diet/optimization response-status audit | IMPLEMENTED 2026-07-18 — API/frontend | Task 213 preparation and PASSED review |
| 2 | Remove duplicate saved-diet repository forwarding API | IMPLEMENTED 2026-07-18 — backend | Task 214 preparation and PASSED review |
| 3 | Durable, concurrent, lookup-efficient Daily Diet create | IMPLEMENTED 2026-07-18 — backend | Task 216 preparation and PASSED review |
| 4 | Canonical quantity-unit boundaries | IMPLEMENTED 2026-07-18 — backend/design | Task 215 preparation and PASSED review |
| 5 | Typed final optimization submission outcomes | IMPLEMENTED 2026-07-18 — backend/observability | Task 223 preparation and PASSED review |
| 6 | `maps.Copy` label cloning | IMPLEMENTED 2026-07-18 — backend/observability | Task 223 preparation and PASSED review |
| 7 | Remove obsolete CLP aliases | IMPLEMENTED 2026-07-18 — backend | Task 217 preparation and PASSED review |
| 8 | Enforce CLP process deadline/cleanup/output authority | IMPLEMENTED 2026-07-18 — backend | Task 217 preparation and PASSED review |
| 9 | Bounded deterministic LP serialization | IMPLEMENTED 2026-07-18 — backend | Task 217 preparation and PASSED review |
| 10 | Go 1.25 iterator CLP version parsing | IMPLEMENTED 2026-07-18 — backend | Task 217 preparation and PASSED review |
| 11 | Exact iterator-based CLP solution grammar | IMPLEMENTED 2026-07-18 — backend | Task 217 preparation and PASSED review |
| 12 | Canonical constraints and real meal-set distinctness | IMPLEMENTED 2026-07-18 — backend/design | Task 218 preparation and PASSED review |
| 13 | Eligible-meal and nutrition-basis boundary | IMPLEMENTED 2026-07-18 — backend/data | Task 218 preparation and PASSED review |
| 14 | Remove ambiguous solver-domain state and unsafe quantity default | IMPLEMENTED 2026-07-18 — backend/design | Task 218 preparation and PASSED review |
| 15 | Calorie-primary, diversity-secondary objective | IMPLEMENTED 2026-07-18 — optimization/design | Task 219 preparation and PASSED review |
| 16 | Objective contract and zero-information candidates | IMPLEMENTED 2026-07-18 — optimization/data | Task 219 preparation and PASSED review |
| 17 | Canonical generation before deduplication | IMPLEMENTED 2026-07-18 — optimization | Task 220 preparation and PASSED review |
| 18 | One authoritative validation/publication pipeline | IMPLEMENTED 2026-07-18 — optimization | Task 220 preparation and PASSED review |
| 19 | Closed optimization failure vocabulary | IMPLEMENTED 2026-07-18 — backend/API | Task 221 preparation and PASSED review |
| 20 | Authoritative non-zero similarity score | IMPLEMENTED 2026-07-18 — product/optimization/frontend | Task 221 preparation and PASSED review |
| 21 | Remove controller-wide submission lock | IMPLEMENTED 2026-07-18 — backend | Task 222 preparation and PASSED review |
| 22 | Canonical normalized optimization request hash | IMPLEMENTED 2026-07-18 — API/backend | Task 222 preparation and PASSED review |
| 23 | Separate exact replay from failed-publication repair | IMPLEMENTED 2026-07-18 — API/backend | Task 222 preparation and PASSED review |
| 24 | Admission errors aligned with `AppError` | IMPLEMENTED 2026-07-18 — API/frontend | Task 222 preparation and PASSED review |
| 25 | Remove ignored replay acknowledgement parameter | IMPLEMENTED 2026-07-18 — backend | Task 222 preparation and PASSED review |
| 26 | Bounded observable admission cleanup | IMPLEMENTED 2026-07-18 — backend/observability | Task 223 preparation and PASSED review |
| 27 | Remove one-use controller validation/fallback helpers | IMPLEMENTED 2026-07-18 — backend | Task 222 preparation and PASSED review |
| 28 | Built-in bounded `Retry-After` clamp | IMPLEMENTED 2026-07-18 — backend | Task 222 preparation and PASSED review |
| 29 | Explicit queue reservation cardinality | IMPLEMENTED 2026-07-18 — backend/queue | Task 224 preparation and PASSED review |
| 30 | Ownership-first atomic attempt counting | IMPLEMENTED 2026-07-18 — backend/queue | Task 224 preparation and PASSED review |
| 31 | Canonical queue UUID validation and malformed cleanup | IMPLEMENTED 2026-07-18 — backend/queue | Task 224 preparation and PASSED review |
| 32 | Coherent queue timing/TTL contract | IMPLEMENTED 2026-07-18 — backend/queue | Task 224 preparation and PASSED review |
| 33 | Explicit fail-safe terminal publication and cleanup | IMPLEMENTED 2026-07-18 — backend/queue | Task 225 preparation and PASSED review |
| 34 | Remove dead queue branches | IMPLEMENTED 2026-07-18 — backend/queue | Task 225 preparation and PASSED review |
| 35 | Correct waiting/pending queue ages | IMPLEMENTED 2026-07-18 — backend/observability | Task 226 preparation and PASSED review |
| 36 | Embedded cached Lua and approved Redis topology | IMPLEMENTED 2026-07-18 — backend/platform | Task 225 preparation and PASSED review |
| 37 | Live stream/group loss recovery | IMPLEMENTED 2026-07-18 — backend/queue | Task 225 preparation and PASSED review |
| 38 | Typed Daily Diet mutation-idempotency persistence | IMPLEMENTED 2026-07-18 — backend/repository | Task 216 preparation and PASSED review |
| 39 | Strict exact-status Daily Diet decoder | IMPLEMENTED 2026-07-18 — frontend/API | Task 228 preparation and PASSED review |
| 40 | Retry-stable Daily Diet create key | IMPLEMENTED 2026-07-18 — frontend | Task 228 preparation and PASSED review |
| 41 | Shared runtime-safe client error mapper | IMPLEMENTED 2026-07-18 — frontend/architecture | Task 227 preparation and PASSED review |
| 42 | Canonical simplified Daily Diet client surface | IMPLEMENTED 2026-07-18 — frontend | Task 228 preparation and PASSED review |
| 43 | Strict optimization union decoder and statuses | IMPLEMENTED 2026-07-18 — frontend/API | Task 230 preparation and PASSED review |
| 44 | Caller-owned secure optimization key | IMPLEMENTED 2026-07-18 — frontend | Task 230 preparation and PASSED review |
| 45 | Cancellable coordinated Daily Diet operation lifecycle | IMPLEMENTED 2026-07-18 — frontend/state | Task 229 preparation and PASSED review |
| 46 | Remove stale-macro optimistic replacement | IMPLEMENTED 2026-07-18 — frontend/product | Task 229 preparation and PASSED review |
| 47 | One authoritative selected-diet source | IMPLEMENTED 2026-07-18 — frontend/state | Task 229 preparation and PASSED review |
| 48 | Explicit current-input optimization retry policy | IMPLEMENTED 2026-07-18 — frontend/state | Task 231 preparation and PASSED review |
| 49 | Resumable identity-safe optimization lifecycle | IMPLEMENTED 2026-07-18 — frontend/state | Task 231 preparation and PASSED review |
| 50 | Bounded leak-free polling configuration/delay | IMPLEMENTED 2026-07-18 — frontend | Task 231 preparation and PASSED review |

The exact action text, original owner, date, and evidence paths remain in `docs/implementation/04_OPEN.md`; this table is the compact audit index.

## Coverage disposition

The repository's 100% goal is not reached. Task 235 therefore replaced the stale Task 212 figures with precise current accepted exceptions under Phase 07 testing coverage deviations.

### Backend

- Aggregate internal coverage: **88.3%**.
- Dedicated Phase 07 packages: `dailydiet 80.1%`, `optimization 84.1%`, `queue 76.1%`, `worker 67.4%`.
- `go tool cover -func=coverage.out` reports **106** below-100 function rows across those packages (**104** unique file/function pairs). `docs/implementation/04_OPEN.md` records every row with the file, declaration line, function, exact measured percentage, zero-count statement ranges, owner/date/evidence, and one explicit rationale. Declaration lines distinguish the two `validator.go` methods both rendered as `UnmarshalJSON`.
- `scripts/check.py` now derives this below-100 inventory from the current aggregate profile and requires an exact `` `path:line function` | `percentage` `` marker for every row. A newly uncovered function or changed measurement therefore fails the gate; package percentages cannot hide missing required coverage.
- The repaired nominal, replay, replacement/deletion, concurrency, cancellation, CLP parser/model, queue ownership/finalization/recovery, timeout, observability, and durable publication paths have focused tests plus PostgreSQL/Redis/CLP integration and full race evidence.

### Frontend

- Aggregate coverage: **94.01% functions / 94.86% lines**, 438 tests.
- The complete executable Phase 07 runtime inventory is ten rows: Daily Diet client `95.74%` functions / `95.22%` lines; error mapper `100.00% / 100.00%`; generated API `100.00% / 98.93%`; optimization client `97.78% / 95.00%`; search client `100.00% / 100.00%`; Daily Diet store `98.31% / 99.55%`; optimization store `98.00% / 100.00%`; search store `84.48% / 94.72%`; selected-diet store `100.00% / 100.00%`; and shared units `50.00% / 76.19%`.
- `src/lib/units.ts` is now enforced by `PHASE07_FRONTEND_SOURCES`. Its exact uncovered lines are `54,63-64,88,106-107,116-119`; all six related `src/lib/units.test.ts` cases pass, including metric/imperial defaults and round-trip tolerances.
- The inventory also adds the previously unenforced `error-message-mapper.ts`, `search-client.ts`, and `selected-daily-diet.ts` rows. Svelte components and type-only `search-state.types.ts` do not produce Bun coverage rows and are explicitly dispositioned to colocated component tests and Task 233 Playwright/axe evidence.
- `docs/implementation/04_OPEN.md` records exact function/line values and uncovered line ranges where Bun provides stable ranges. Accepted gaps are defensive bounded-body/abort/fallback, generated unreachable fallback, callback instrumentation, impossible-state search projection, and unsupported/cross-basis conversion guards.
- Strict decoder, idempotency, retry/lifecycle, authoritative selection, malformed response, responsive/theme/keyboard, axe, and real-browser workflows pass.

No product behavior was waived. Coverage exceptions are measurement/testability dispositions only and require an aggregate rerun after Phase 07 production changes.

## Commands and exact results

| Command | Result |
|---|---|
| `python3 scripts/check.py` (first run) | FAIL only at stale coverage documentation after all prior gates passed: backend `88.3%`; Phase 07 packages `80.1%`, `84.1%`, `76.1%`, `67.4%` were not present in `04_OPEN.md`. |
| `cd backend && go tool cover -func=coverage.out` filtered to Phase 07 packages | PASS; produced the exact below-100 function inventory now recorded in `04_OPEN.md`. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | PASS; 438 tests, 1,998 expectations, `94.01%` functions / `94.86%` lines. |
| Exact backend coverage-inventory audit | PASS; 106 below-100 rows checked, zero missing exact file/line/function/percentage markers. |
| Phase 07 frontend inventory validator | PASS; all ten runtime rows present, including `src/lib/units.ts`; all below-100 measurements match `04_OPEN.md`. |
| `python3 scripts/check.py` (final rerun) | PASS, exit 0. Includes traceability/task-list/Go Doc, OpenAPI, capacity unit, vet, vulnerability, local stack/UAT, focused integrations, formatting, full backend normal/race/coverage, API drift/typecheck/build/unit/coverage, focused browser, and complete Playwright/axe gates. |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS; 16 tests in 0.472s. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types are current. |
| `python3 scripts/verify-phase0701-observability-capacity.py` | PASS; 10 Python tests and every required selected Go test in normal and race modes. Injected Redis connection-refused lines are expected bounded cleanup/restart evidence. |
| `for n in $(seq 213 234); do python3 .../validate_review_evidence.py docs/implementation/reviews/task-$n-review.md; done` | PASS; all 22 review files structurally valid. |
| Historical/current action-count audit | PASS; baseline `50`, current open `0`, current evidenced IMPLEMENTED `50`. |
| `git diff --check` | PASS. |

Aggregate browser details: auth/subscription/search focused suite `70 passed`; Daily Diet/Phase 07 acceptance suite `28 passed`; complete suite `231 passed, 3 skipped`. The skips are maintained suite-defined environmental/visual cases and are not hidden failures; all Task 233 required browser cases passed in the focused 28-test run.

The OpenAPI lint emits one accepted warning: OAuth callback has intentional `302` and no `2XX`. `govulncheck` reports no called vulnerabilities; it reports 18 vulnerabilities in required modules that are not reached by application code.

## Final hashes

| Artifact | SHA-256 |
|---|---|
| `docs/implementation/04_OPEN.md` | `f92ef7b4dfb9ed8d6b43d08e0897d9bcc6b838f485022b2c909248a48ae8be19` |
| `scripts/check.py` | `1e7c89d4eaf5272c816eb8284eeab1dd09fa27fc962bf1e1a0a8a6ff3f963119` |
| `docs/implementation/02_TASK_LIST.md` | `7544d5d761481992c790288e8a4d49fcf20fa89aa8945298e1c6188614f1e833` |
| `api/openapi.yaml` | `392a3d531301a937b001bc7561b6e5cdef76a6a786d2073d739ab81cd1161c4a` |
| `frontend/src/lib/api/generated.ts` | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| Ordered SHA-256 manifest of Task 213-234 preparation and review files | `51ec29ce252c0b8e405f55fb2097d84699f834abbfb12a4486df4ac949191b59` |
| `backend/coverage.out` (generated, ignored) | `26f72b2424e49262d0c9f50b55ebd23c1be296bd9a61a0204d32e39996d7c09b` |
| Final desktop verification screenshot (temporary) | `5e53c2313bd3ee25e9136451d5909fcb58823589191220f299069cc98bf5f875` |
| Final mobile verification screenshot (temporary) | `f77d1561526a35e86b63404af157b037b40e908a92e08f7ce72038f9ddf943d2` |

`backend/coverage.out` and `/tmp/mealswapp-frontend-verifier/*` remain generated local evidence and are not added to the repository.

## Preparation decision

Task 235's verification contract is satisfied with precise accepted coverage exceptions. All 50 cited review actions are dispositioned and evidenced, the full aggregate gate passes, and no required validation remains unrun. The task status intentionally remains `OPEN` for the external status owner.
