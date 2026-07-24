# Task 256 preparation — Manual Item, Classification, and User Admin UI

## Repair outcome

Task 256 (`DESIGN-009: ItemCurator`, `TagManager`, and `UserAdminPanel`) has been repaired against `docs/implementation/reviews/task-256-review.md`. This repair did not edit `docs/implementation/02_TASK_LIST.md`, the OpenAPI source, generated API types, backend code, task-255 external-import work, or task-257 dynamic-filter work.

All review findings are addressed in the task-owned frontend surface:

| Review finding | Repaired behavior | Focused evidence |
| --- | --- | --- |
| F-256-001 destructive target race | `Confirmation` is a frozen ID-bearing snapshot; `confirm` aborts the resource's outstanding operation; the background is `inert`; `deleteItem`, `deleteClassification`, and `retryDeletion` verify the immutable target against current authoritative state before mutation. | Playwright changes item state after confirmation and proves no DELETE occurs; Confirm receives focus; cancel remains covered. |
| F-256-002 replacement data loss | `AdminItemForm`, `applyItem`, and `parseAdminItemForm` round-trip `imageUrl`, preparation time, unit/serving measures, density, provider, source food ID, source kind, micronutrients, and both classification-ID collections. | Unit DTO round-trip plus browser load/edit/PUT assertion for imported dense liquid data. |
| F-256-003 stale/concurrent state | Independent item/classification/user generations and `AbortController`s guard all reads, mutations, recovery reads, classification refreshes, and user lookups. Aborts propagate through the client without becoming visible network errors. | Delayed out-of-order Playwright tests cover item reads, user lookups, classification mutations and follow-up refreshes; client unit test covers abort propagation. |
| F-256-004 weak/bounded decoding | The client now bounds request JSON (64 KiB), success bodies (256 KiB), error bodies (16 KiB), and all documented collections; it validates every nested item, classification, user, deletion, macro, micronutrient, UUID, enum, date, text, measure, provenance, and URL field. Error status/code projection is bounded and allowlisted. | Adversarial unit tests reject wrong nested types, invalid enums/ranges/dates/URLs, oversized classification lists and response/request bodies, hostile error codes, and privacy extras. |
| F-256-005 dense-liquid rejection | The 100-total rule applies only to solid 100 g Macro Profiles. Liquids may exceed 100 on their 100 ml Nutrition Basis while still requiring bounded positive density and valid provenance. | Red reproduction failed before repair; focused unit and browser replacement tests accept carbohydrates `110` with density `1.2`. |
| F-256-006 missing adversarial coverage | Browser fixtures now distinguish mutation projections from authoritative follow-up GETs and count hierarchy reads. | Successful unused classification delete/reload, authoritative projection divergence, target race, field preservation, delayed responses, dense liquid, cancellation, and existing audit/conflict flows pass on desktop and mobile. |
| F-256-R-001 malformed date-time acceptance | `dateTime` now requires the bounded RFC3339 date/time shape, a required `T` separator and timezone, bounded offset fields, and a Gregorian calendar-valid day for the parsed year/month. | The public `lookupAdminUsers` boundary accepts leap days, fractional seconds, `Z`, and signed offsets; it rejects impossible dates, non-contract separators, missing/malformed zones, invalid zone ranges, and an impossible nested deletion date. |
| F-256-R-002 declared oversized response cleanup | `readBoundedText` explicitly cancels an unread response body when its declared `Content-Length` exceeds the endpoint cap. Cancellation failures are contained so the existing bounded success-response error and safe error-response projection remain authoritative. | A focused public-client test observes cancellation for oversized success and error responses, forces one cancellation failure, and verifies that neither transport cleanup detail nor altered error semantics escape. |
| F-256-R-003 wrong-status success cleanup | `json` and `emptyMutation` now explicitly cancel every successful response body whose status differs from the endpoint contract before returning the existing safe malformed-response error. Best-effort cleanup contains cancellation failures. | A focused public-client regression observes cancellation for JSON and empty mutation status mismatches, forces cancellation failure in both paths, and proves the safe status/code/message remain authoritative. |
| F-256-R-004 classification rename parent preservation | Classification edit state retains the decoded authoritative `parentId`; rename PUT sends that ID, or explicit `null` for an authoritative root, before refreshing both hierarchy projections. | A generated-contract API test round-trips `parentId`; desktop/mobile browser coverage renames a nested Food Category and proves the PUT and refreshed child retain the original parent. |

## Preservation and baseline

- Fixed repository reference remains `81ca40ce00cb667ea29243ed2d34068e11229a69` on `multistep-phase-08`.
- The repair began in a heavily dirty concurrent Phase 08 worktree. Existing changes were preserved; no reset, checkout, clean, commit, generated-file edit, backend edit, OpenAPI edit, or task-list edit was performed.
- Review input fingerprint: `docs/implementation/reviews/task-256-review.md` SHA-256 `33286a94ec5a3232cadc4ca726db9eb0575db9932646eddb36ae1b9590cbf137` records the rejected implementation hashes and required repair surface.
- `docs/implementation/02_TASK_LIST.md` was already modified when repair began and remains outside this repair. Its current SHA-256 is `b7ab952df56141c2abe3bcc0f2f66a1ca8f5cd5fbe9845d5e1103994394108ca`.
- Unchanged task dependencies: `docs/design/DESIGN-009.md` SHA-256 `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b`; `api/openapi.yaml` SHA-256 `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46`; `frontend/src/lib/api/generated.ts` SHA-256 `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0`.

## Exact repaired symbols

| Path | Exact task-256 symbols/units |
| --- | --- |
| `frontend/src/lib/api/admin-client.ts` | Existing public `ClassificationKind`, `AdminMutationOptions`, `AdminClientError`, item/classification/user wrappers, `AdminApi`, and `adminApi`; modified `listAdminClassifications`, `mutation`, `emptyMutation`, `request`, `json`, `decodeData`, `decodeItem`, `decodeClassification`, `decodeUser`, `responseError`, `malformed`, and `readBoundedText`; added `invalidRequest`, `cancelResponseBody`, `decodeClassificationSummary`, `macroProfile`, `micronutrients`, `optionalUuidCollection`, `optionalPositive`, `optionalBoundedString`, `finiteBetween`, `nonnegativeInteger`, `boundedString`, `email`, `dateTime`, `safeUriReference`, `safeErrorStatus`, and `isAbort`, plus bounded constants and `SAFE_ERROR_CODES`. Declared oversized and wrong-status successful bodies are explicitly canceled while cleanup failures preserve the primary safe error. |
| `frontend/src/lib/admin-workflows.ts` | Expanded `AdminItemForm`; modified `parseAdminItemForm`; retained `deletionRetryEligible` and `newAdminItemKey`; added `optionalNumber`, `uniqueUuids`, and `safeUriReference`; retained numeric/object helpers. |
| `frontend/src/lib/components/AdminDataManagement.svelte` | Added immutable `Confirmation`; operation generations/controllers; retained authoritative `classificationParentId`; `beginItemOperation`, `beginClassificationOperation`, `beginUserOperation`, three `current*Operation` guards, `aborted`, and `classificationProjection`; modified `refreshClassifications`, `loadItem`, `applyItem`, `saveItem`, `refreshCurrentItem`, `newItem`, `deleteItem`, `saveClassification`, `editClassification`, `deleteClassification`, `lookupUsers`, `retryDeletion`, `confirmAction`, and `confirm`; added `resetItemState`; added teardown cancellation, lossless fields, inert confirmation boundary, and busy action guards. |
| `frontend/src/lib/api/admin-client.test.ts` | Added strict nested/bounded decoder tests, including observable cancellation and cleanup-failure containment for declared oversized and wrong-status success/error streams; added generated-contract nested-parent replacement round-trip; extended route/error tests. |
| `frontend/src/lib/admin-workflows.test.ts` | Added complete field/provenance round-trip, dense-liquid acceptance, and idempotency-key tests; expanded fixture. |
| `frontend/src/lib/components/AdminDataManagement.test.ts` | Replaced stale source assertions with immutable-target, inert-boundary, generation, signal, and authoritative-follow-up assertions. |
| `frontend/tests/admin-data-management.spec.ts` | Extended stateful server fixture with request/read counters, divergent projections, delayed item/user/classification operations, item/classification PUT capture, nested hierarchy state, and DELETE capture; added nested-child rename preservation, unused classification delete/reload, lossless replacement, target race, stale item/user reads, and stale classification mutation/refresh scenarios. |
| `frontend/src/lib/components/AdministrationPanel.svelte` | No repair change; existing task-256 allowed-branch composition remains intact. |

## Behavioral invariants after repair

- A destructive confirmation names and executes exactly one immutable ID. If the displayed authoritative target changes, confirmation fails closed without a mutation.
- While confirmation is open, background controls are inert. Outstanding work for that resource is aborted before the confirmation snapshot is installed.
- Only the latest operation generation may commit data, success/error messages, or busy-state cleanup. This applies across mutation response, authoritative follow-up, and error recovery.
- Item PUT remains replacement-based and lossless for all generated editable fields. A mutation result is never displayed as final; the follow-up GET projection wins.
- Client boundary validation is stricter than TypeScript casts and never renders raw server text, unknown privacy fields, unbounded documents, or unapproved server codes.
- A response rejected from its declared size has its unread body explicitly canceled. Cleanup failure cannot replace or leak through the bounded malformed-response or safe status/code error semantics.
- A successful response rejected for an undocumented status has its body explicitly canceled in JSON and empty mutation paths. Cleanup failure cannot replace or leak through the malformed-response error.
- Classification rename preserves the authoritative hierarchy: a child PUT carries its existing `parentId`, while an authoritative root carries explicit `null`; renaming cannot implicitly reparent either node.
- Admin user `createdAt` and deletion `requestedAt` values cross the decoder boundary only when they have strict RFC3339 syntax and a calendar-valid Gregorian date.
- Solid Macro Profiles retain the `<= 100 g` local rule. Liquid Macro Profiles use a 100 ml basis and require valid density/provenance without the solid-total restriction.

## Verification evidence

| Command / probe | Result |
| --- | --- |
| `cd frontend && ... bun test src/lib/admin-workflows.test.ts` before repair | **EXPECTED FAIL:** dense-liquid regression received no request because carbohydrates `110` triggered the old solid-total rule. This was the deterministic red reproduction. |
| `cd frontend && ... bun test src/lib/api/admin-client.test.ts` before the date guard repair | **EXPECTED FAIL:** 4 passed and the new strict date test failed because `lookupAdminUsers` resolved an impossible date instead of rejecting it. |
| `cd frontend && ... bun test src/lib/api/admin-client.test.ts` before the declared-size cleanup repair | **EXPECTED FAIL:** 5 passed and the focused cancellation test failed with `canceled.success` false, reproducing the unread oversized stream. |
| `cd frontend && ... bun test src/lib/api/admin-client.test.ts` before the wrong-status cleanup repair | **EXPECTED FAIL:** the focused regression observed all four JSON/empty mismatch streams with cancellation false. |
| `cd frontend && ... bunx playwright test tests/admin-data-management.spec.ts --project=desktop-chromium --grep='classification conflicts'` before parent preservation | **EXPECTED FAIL:** nested child rename sent `{ name }` without `parentId`, reproducing implicit reparenting. |
| `cd frontend && ... bun run typecheck` after production repair | **PASS** at the task-256 compile checkpoint. |
| `cd frontend && ... bun test src/lib/api/admin-client.test.ts` after final response/parent repairs | **PASS:** 8 tests, 0 failures, 43 expectations; mismatch and oversized streams were canceled, cleanup failures remained contained, and nested `parentId` round-tripped. |
| `cd frontend && ... bun test src/lib/admin-workflows.test.ts src/lib/api/admin-client.test.ts src/lib/components/AdminDataManagement.test.ts --coverage` | **PASS:** 17 tests, 0 failures, 77 expectations. `admin-client.ts` 100.00% lines; `admin-workflows.ts` 98.51% lines. |
| `cd frontend && ... bunx playwright test tests/admin-data-management.spec.ts --project=desktop-chromium` | **PASS:** 7/7 before the final classification-race addition. |
| `cd frontend && ... bunx playwright test tests/admin-data-management.spec.ts --project=desktop-chromium --grep='older classification'` | **PASS:** delayed older classification mutation/refresh cannot replace the latest projection. |
| `cd frontend && ... bunx playwright test tests/admin-data-management.spec.ts` | **PASS:** 16/16 across desktop Chromium and Pixel 5/mobile Chromium, including nested-child rename preservation and axe light/dark checks. |
| `python3 scripts/verify-frontend.py` | **PASS:** desktop/mobile scenario capture and frontend verification completed. |
| `python3 scripts/validate-traceability.py` | **PASS:** traceability validation passed. |
| `python3 scripts/validate-task-list.py` | **PASS:** 263 sequential tasks with ordered dependencies. The validator is read-only; the repair did not edit the task list. |
| `npx --no-install redocly lint api/openapi.yaml` | **PASS with one pre-existing warning:** OAuth callback has only a 302 response; exit 0. |
| `git diff --check` | **PASS:** no whitespace errors. |
| `[DEBUG-...]` search in task-owned implementation/tests | **PASS:** no temporary diagnosis instrumentation remains. |
| `cd frontend && ... bun run check` final aggregate | **PASS:** generated API types current, TypeScript typecheck and production build passed, and 514 tests passed with 0 failures and 2393 expectations. |

## Final SHA-256 fingerprints

| Path | SHA-256 |
| --- | --- |
| `frontend/src/lib/api/admin-client.ts` | `90a20ce422593fe6593856d34f2a954b8157932de01364815cd6c13ef8aa59bb` |
| `frontend/src/lib/api/admin-client.test.ts` | `931918cd3c0e10eddb823f680385c175b667675eede8cc962a315c831bcda83a` |
| `frontend/src/lib/admin-workflows.ts` | `d6e5e1f1e3d8750dec1c706c1a2a6929ea1e957ff3bf0ee4e3c2e1fbfa413377` |
| `frontend/src/lib/admin-workflows.test.ts` | `68a6f890cba079d16b2b5db1cc2872084b107417762e8809be8af030fcf822b6` |
| `frontend/src/lib/components/AdminDataManagement.svelte` | `3b14eeaa06475db9ff98e71a14bc572e8d0e06023b73850041d1e58d6d591ec1` |
| `frontend/src/lib/components/AdminDataManagement.test.ts` | `79f61408c5793837bb1e2674edb672d9fba5da8fee76f9acc7ecec0e241a693d` |
| `frontend/src/lib/components/AdministrationPanel.svelte` | `cd758ecf302e2b8d722b0be5bd82ac6cb6e457490a3759261bb620769bdd858f` (unchanged shared composition) |
| `frontend/tests/admin-data-management.spec.ts` | `df386b191ec64a0ba3abc36a2a1d7d483c4e2d578692efe11d5dc1062c478c3f` |
| `docs/implementation/02_TASK_LIST.md` | `b7ab952df56141c2abe3bcc0f2f66a1ca8f5cd5fbe9845d5e1103994394108ca` (pre-existing/concurrent modification; untouched by repair) |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |

Task-list status is intentionally unchanged by this repair.
