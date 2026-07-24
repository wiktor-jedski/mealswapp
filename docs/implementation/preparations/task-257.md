# Task 257 preparation — Dynamic Substitution Filter UI

## Outcome

Task 257 (`DESIGN-001: SearchView`) is repaired and remains `PREPARED`. No edit was made to `docs/implementation/02_TASK_LIST.md`; its concurrent worktree diff changed from `19/19` to `20/20` while this repair was in progress and is preserved as unrelated work.

The Substitution filter client now treats `api/openapi.yaml` and generated `SearchFilterKind` as a closed runtime contract. `food_object_type` is rejected both as an option kind and in nested exclusion references. Successful payloads must have exact envelope, data, option, and reference keys; all required fields and booleans must be present and correctly typed; status, mode, and filter kinds must be allowed enum values; IDs, labels, and optional label keys must contain 1–200 Unicode code points; options and exclusions are capped at 1000 and 20 entries respectively. Unknown ordering fields such as `order` and `sortOrder`, other extra fields, and malformed nested values are rejected.

Response decoding is capped at 32 MiB from both declared content length and streamed byte count before JSON parsing. Fetch-time and body-read aborts remain `AbortError`. Any invalid successful response enters the same fixed, recoverable UI error state as an unavailable endpoint; selected-item Food Category and Culinary Role options remain usable and no policy fallback is invented.

Direct keyboard activation of rendered filter options now handles Enter and Space. Backend order, exact backend labels, operation permissions, selected-classification deduplication, ID-only search requests, focus refresh, retry/empty/error states, and monotonic stale-response protection remain intact.

## Review findings addressed

| Finding | Disposition and evidence |
| --- | --- |
| Task status gate | Resolved by concurrent orchestration: row 257 is `PREPARED`. This repair did not edit the task list. |
| F-257-001 unsupported `food_object_type` | Removed from `isSearchFilterKind`; adversarial option-kind and nested-reference tests reject it. |
| F-257-002 fail-open/unbounded schema | Exact-key guards, required type checks, enum checks, 1–200 text bounds, 20/1000 array bounds, and a 32 MiB pre-parse stream cap reject all reviewed mutants. Tests cover missing/blank/oversized fields, extra envelope/data/option/reference fields, invalid ordering fields, malformed nested values, and oversized bodies. |
| F-257-003 body abort mapping | `fetchSubstitutionFilterOptions` preserves abort reasons while reading a response body; fetch-time and body-read tests both assert `AbortError`. |
| F-257-004 direct option keyboard path | `onFilterOptionKeydown` activates option buttons with Enter or Space; the desktop/mobile browser scenario focuses an option, presses Enter, and observes the selected chip and ID-only request. |
| Cross-workflow invalidation evidence | Task 257 browser coverage proves focus refresh consumes the renamed inventory without stale overwrite. Dependency Task 241's `TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration` was rerun and passes, proving administration invalidation at the backend-owned source. The broader Administration Panel-to-Search browser gate remains explicitly owned by dependent task 259. |

## Design and contract sources

- `docs/design/DESIGN-001.md`: `SearchView` filter composition, generated-request orchestration, recoverable states, and accessibility.
- `docs/design/DESIGN-002.md`: closed `SearchFilterKind`, filter identity, and include/exclude request semantics.
- `docs/design/DESIGN-009.md`: backend ownership of classification policy and invalidation.
- `api/openapi.yaml`: exact `FilterOptionReference`, `FilterOption`, and `FilterOptionsEnvelope` schemas and bounds.
- `frontend/src/lib/api/generated.ts`: generated `SearchFilterKind`, `FilterOption`, `FilterOptionReference`, and `FilterOptionsEnvelope` types.
- `backend/internal/search/filter_options.go`: deterministic backend-owned option projection and administration invalidation seam.

## Exact changed symbols and surfaces

| Path | Task 257 symbols and surface |
| --- | --- |
| `frontend/src/lib/api/filter-options-client.ts` | `FILTER_OPTIONS_ENDPOINT`; `MAX_RESPONSE_BYTES`; `MAX_OPTIONS`; `MAX_EXCLUDES`; `MAX_TEXT_LENGTH`; `FilterOptionsClientError`; `fetchSubstitutionFilterOptions`; `isFilterOptionsEnvelope`; `isFilterOption`; `isFilterOptionReference`; `isSearchFilterKind`; `isRecord`; `hasOnlyKeys`; `isBoundedText`; `readBoundedJson`. |
| `frontend/src/lib/api/filter-options-client.test.ts` | `envelope`; `option`; `expectRejected`; six tests covering valid fetch behavior, unsupported kinds, required fields/enums/text bounds, exact nested schemas/array bounds, body bounds/safe errors, and fetch/body abort preservation. |
| `frontend/src/lib/substitution-filter-options.ts` | `SubstitutionFilterOption`; `substitutionFilterOptions`; `projectBackendOption`; `kindLabel`; `optionIdentity`. |
| `frontend/src/lib/substitution-filter-options.test.ts` | Three projection tests for backend order/localized labels/deduplication, operation permissions/ID identity, and selected-only degraded behavior. |
| `frontend/src/lib/components/SubstitutionInputs.svelte` | `backendFilterOptions`; `filterOptionsStatus`; `filterOptionsRequest`; `filterOptionsAbort`; `includeFilterOptions`; `excludeFilterOptions`; `loadFilterOptions`; `onFilterOptionMouseDown`; `onFilterOptionKeydown`; `filterLabel`; loading/empty/error/retry branches; include/exclude option key handlers. |
| `frontend/src/lib/components/SubstitutionInputs.test.ts` | Dynamic-client/projection source assertions; no-hardcoded-inventory assertion; refresh, stale guard, recoverable states, retry, and two keyboard-handler binding assertions. |
| `frontend/tests/dynamic-substitution-filters.spec.ts` | `option`; `fulfill`; `stubApplication`; `addSelectedItem`; well-formed/stale refresh scenario with direct keyboard option activation; unavailable/empty scenario; repeated unsupported/extra-field/out-of-bounds malformed-success scenario with selected-only safe degradation. |

## Verification evidence

| Command | Current result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage src/lib/api/filter-options-client.test.ts src/lib/substitution-filter-options.test.ts src/lib/components/SubstitutionInputs.test.ts` | PASS: 20 tests, 156 assertions; `filter-options-client.ts` and `substitution-filter-options.ts` both 100% functions and lines. |
| `cd frontend && ... bun run typecheck` | PASS. |
| `cd frontend && ... bun run build` | PASS: 217 modules transformed. |
| `cd frontend && ... bunx playwright test tests/dynamic-substitution-filters.spec.ts` | PASS: 6/6 across desktop Chromium and Pixel 5; includes malformed-success degraded UI and direct option Enter activation. |
| `cd frontend && ... bunx playwright test tests/search-workflow.spec.ts --project=desktop-chromium -g "Substitution Input search sends inputs"` | PASS: 1/1. Pre-existing unstubbed local API proxy calls log connection refusals while the controlled scenario passes. |
| `cd frontend && ... bunx playwright test tests/accessibility.spec.ts --project=desktop-chromium -g "keyboard-only Substitution workflow"` | PASS: 1/1. Pre-existing unstubbed local API proxy calls log connection refusals while the controlled scenario passes. |
| `cd frontend && ... bun run check` | PASS: API generated-type drift, typecheck, production build, and all 507 frontend unit/component tests; 2,357 assertions. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search -run '^TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration$' -count=1` | PASS: backend administration-to-filter-option invalidation dependency evidence. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `git diff --check` | PASS. |
| Production-source search for `food_object_type`, former inventories, and fixed substitution IDs | PASS: matches are confined to adversarial/contract fixtures; no production runtime match. |

The phase-wide root `scripts/check.py` gate was not run because this request is limited to PREPARED task 257 and Phase 08 is not complete. Task-specific frontend aggregate, browser, dependency invalidation, traceability, task-list, and whitespace checks all pass.

## Current SHA-256 fingerprints

| Path | SHA-256 |
| --- | --- |
| `frontend/src/lib/api/filter-options-client.ts` | `1d5944fa03e856bcc3324aeeabd205fa0240bc809a57a99e7b586421cffcd593` |
| `frontend/src/lib/api/filter-options-client.test.ts` | `75171bdbe81672c744dd589a42c27156658547e158995bfe496a72e8a3f75a87` |
| `frontend/src/lib/substitution-filter-options.ts` | `6ac6c8fcf2083172f99754fb4922fbfb87c0d802ee62219e4593c4a33e9e58cc` |
| `frontend/src/lib/substitution-filter-options.test.ts` | `47c0aab73f6cfec3917bc7b495af7392091eae08cefa520401ca15b4c10d1771` |
| `frontend/src/lib/components/SubstitutionInputs.svelte` | `c6669695f13eb087e7c0fcbc043d6cefdaa624b0694012f302d7bd2e99e1a434` |
| `frontend/src/lib/components/SubstitutionInputs.test.ts` | `bb28a91215c71aaa3c11deb0d55031d50a7795069d1474c173b5d38d1c40373e` |
| `frontend/tests/dynamic-substitution-filters.spec.ts` | `4b7fa47c3cd998666a23f3739cf9779efeebc0e26d6b36338a223fcb13ec858a` |
| `docs/design/DESIGN-001.md` | `3b61228bdce782567af30197dde5558e33118da5dd72fc78cdbb4834210f75ee` |
| `docs/design/DESIGN-002.md` | `179ff0b7f7226164696fc631615993f4e59e2ee30ad8b87f4a445b9de4f75a2f` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |
| `docs/implementation/reviews/task-257-review.md` | `4c43b10be1d783c327ad6e6b1fcba73e016b9bb9c242ca781d64da026f42bb19` |
| `docs/implementation/02_TASK_LIST.md` | `5ab364cec2962283cf8a9a31087e395ef36216ce37730410de3df577f2d1f2a4` (concurrent work; not Task 257 work) |

The preparation document omits its own self-referential digest. No task-list, backend production, OpenAPI, generated-contract, design, review, or unrelated concurrent source file was edited by this repair.
