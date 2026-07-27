# Task 265 preparation — Strict Custom Item JSON Validation

## Baseline and scope

- Task: **265 — Phase 08.01 Strict Custom Item JSON Validation** (`DESIGN-010: RequestValidator`).
- Fixed baseline and current `HEAD`: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- Dependencies inspected in `docs/implementation/02_TASK_LIST.md`: task 239 and task 242 are both `PASSED`.
- `docs/implementation/02_TASK_LIST.md` was read only and no task status was edited.
- The developer role at `/home/wiktor/.codex/agents/developer.toml` was followed: the implementation reuses the existing standard-library recursive JSON token scanner and adds only the private custom-item boundary call and focused regressions.
- The installed `golang-security` skill was applied in coding mode to the hostile authenticated HTTP input boundary.
- Concurrent edits present before or during this preparation were preserved. Task 265 did not edit or revert the OpenAPI, frontend, generator, other backend, Task 270, or browser-test work visible in the shared worktree.

## Exact task-owned changed paths

1. `backend/internal/httpapi/custom_item_controller.go`
2. `backend/internal/httpapi/custom_item_controller_test.go`
3. `docs/implementation/preparations/task-265.md`

No persistence, service, OpenAPI, frontend, migration, task-list, or dependency file was changed for task 265.

## Implementation

`validateCustomItemBody` now calls the existing `rejectDuplicateJSONKeys` scanner before `decodeCustomItemRequest`. Any malformed or duplicate-key result is converted to the existing safe `400 invalid_json` response through `invalidCustomItemBodyError`. The scanner recursively tracks object keys, so the same boundary covers top-level properties, `macrosPer100`, `micros`, objects nested in arrays, and any future nested object without maintaining a field-specific duplicate list.

The placement is deliberately route-specific:

1. router authentication and CSRF policy still run normally;
2. private custom-item create/update validation scans the raw body;
3. only a duplicate-free body reaches typed decoding and domain validation;
4. only an accepted typed request reaches the custom-item service;
5. therefore duplicate bodies cannot reach repository persistence.

No rejected key or value is logged or returned. The response remains the generic `invalid request body` envelope.

## Added or modified executable symbols

| Symbol | Kind | Change |
|---|---|---|
| `validateCustomItemBody` | production function | Modified to invoke recursive duplicate-key rejection before `decodeCustomItemRequest` and map rejection to the stable invalid-JSON error. |
| `(*fakeCustomItemService).Create` | test-double method | Modified to count create dispatches. |
| `(*fakeCustomItemService).Update` | test-double method | Modified to count update dispatches. |
| `TestProfileControllerCustomItemRejectsDuplicateJSONKeysBeforeService` | test function | Added six authenticated route cases: duplicate top-level, macro, and micronutrient keys for both create and update; each asserts safe `400 invalid_json` and zero create/update service calls. |
| `TestProfileControllerCustomItemRejectsEscapedNULProvenanceBeforeService` | test function | Modified its `densitySourceKind` fixture to avoid an unintended duplicate key, retaining the original escaped-NUL `validation_failed` assertion under the new duplicate-key boundary. |

The `fakeCustomItemService` test structure also gained `createCalls` and `updateCalls` counters; these are data fields, not executable symbols.

## Acceptance-criteria evidence

| Criterion | Evidence | Result |
|---|---|---|
| Recursive rejection happens before decode, service, and persistence. | `validateCustomItemBody` invokes `rejectDuplicateJSONKeys(ctx.Body())` before `decodeCustomItemRequest`; route tests assert both fake service counters remain zero. Repository access is behind that service boundary and is therefore unreachable. | **PASS** |
| Duplicate top-level keys are rejected on create and update. | `top-level` subcases duplicate `name` on POST and PUT. | **PASS — 400 `invalid_json`** |
| Duplicate `macrosPer100` keys are rejected on create and update. | `macrosPer100` subcases duplicate `protein` on POST and PUT. | **PASS — 400 `invalid_json`** |
| Duplicate micronutrient keys are rejected on create and update. | `micronutrients` subcases duplicate `Sodium` inside `micros` on POST and PUT. | **PASS — 400 `invalid_json`** |
| Safe envelope and no downstream calls. | Every new subcase decodes the shared envelope, checks status/code, and checks `createCalls == 0` and `updateCalls == 0`. No raw hostile value appears in production logging or errors. | **PASS** |
| Valid requests and CSRF behavior remain unchanged. | `TestProfileControllerCustomItemRoutesRequireAuthenticationAndCSRF` passes, including anonymous denial, missing-CSRF denial, valid create, valid update, and delete. | **PASS** |
| Unknown/client-owned and required/domain validation remain unchanged. | `TestProfileControllerCustomItemRejectsClientOwnershipAndMapsSafeErrors`, `TestProfileControllerCustomItemRejectsEscapedNULProvenanceBeforeService`, and `TestProfileControllerCustomItemRejectsSchemaAndDomainViolationsBeforeService` pass. | **PASS** |
| Ownership and idempotency remain unchanged. | `TestServiceDerivesOwnershipAndKeepsCrossUserItemsNotFound`, `TestServiceCRUDListAndInvalidRequestsHaveExpectedSideEffects`, `TestServiceCreateReplayNormalizesBodyAndRejectsKeyReuse`, and `TestServiceCreatePreservesResourceConflictClassification` pass. | **PASS** |
| Focused tests, vet, and security review pass. | Commands and review below. | **PASS** |

## Commands and results

Commands ran from `backend/` unless stated otherwise.

| Command | Result |
|---|---|
| Initial focused custom-item HTTP command after the first implementation | **FAIL as useful fixture discovery:** the existing escaped-NUL test accidentally supplied `densitySourceKind` twice and now correctly received `invalid_json`. The fixture was repaired to isolate escaped-NUL behavior; no production relaxation was made. |
| `go test ./internal/httpapi -run 'TestProfileControllerCustomItem(RejectsDuplicateJSONKeysBeforeService\|RoutesRequireAuthenticationAndCSRF\|RejectsClientOwnershipAndMapsSafeErrors\|RejectsEscapedNULProvenanceBeforeService\|RejectsSchemaAndDomainViolationsBeforeService)' -count=1` with repository-local Go caches | **PASS:** `ok .../internal/httpapi 0.012s`. |
| `go test ./internal/httpapi -count=1` with repository-local Go caches | **PASS:** final rerun `ok .../internal/httpapi 2.740s`. |
| `go test ./internal/customitem ./internal/httpapi ./internal/repository -count=1` with repository-local Go caches | **PASS:** customitem `0.005s`, httpapi `2.663s`, repository `31.036s`. |
| `go test ./internal/customitem -run 'TestService(CreateReplayNormalizesBodyAndRejectsKeyReuse\|CreatePreservesResourceConflictClassification\|DerivesOwnershipAndKeepsCrossUserItemsNotFound\|CRUDListAndInvalidRequestsHaveExpectedSideEffects)' -count=1` with repository-local Go caches | **PASS:** `ok .../internal/customitem 0.004s` in the final run. |
| `go vet ./...` with repository-local Go caches | **PASS:** exit 0, no findings. It was rerun after the final production placement. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` with repository-local Go caches | **PASS:** no called-code vulnerabilities; zero vulnerabilities affect this code. The tool also reported one vulnerability in imported packages and 18 in required modules that the application does not call. |
| `python3 scripts/validate-task-list.py` from the repository root | **PASS:** `Task-list validation passed: 275 sequential tasks with ordered dependencies.` |
| `git diff --check -- backend/internal/httpapi/custom_item_controller.go backend/internal/httpapi/custom_item_controller_test.go docs/implementation/preparations/task-265.md` from the repository root | **PASS:** exit 0, no whitespace errors. |

`gofmt` was applied to both changed Go files before final verification.

## `golang-security` hostile-JSON review

- **Trust boundary:** attacker-controlled authenticated POST/PUT body after the existing auth and CSRF gates.
- **Tampering/ambiguity:** recursively repeated object members are rejected from raw bytes before map or struct decoding can collapse them.
- **Authorization and ownership:** the change does not read owner identity from the body and does not alter server-derived user IDs, path validation, CSRF, or owner-scoped service/repository behavior.
- **Information disclosure:** duplicate failures use the existing generic `invalid_json` code/message; raw keys, values, decoder detail, and persistence detail are neither logged nor returned.
- **Injection/persistence:** the service is not dispatched for a duplicate body, so no duplicate-derived value can reach SQL. Existing repository SQL remains parameterized and was not changed.
- **Availability:** the scanner performs a single recursive token pass over Fiber's already bounded request body. Work is linear in body tokens and key tracking is scoped to each object. No new goroutine, retry, network, or persistence work is introduced.
- **Dependency security:** `govulncheck` reports no vulnerability reachable by the application.

Review conclusion: **PASS** for the task-265 hostile JSON boundary. No security exception was added.

## Risks and handoff notes

- `rejectDuplicateJSONKeys` is an existing package-level helper also used by curation flows. Task 265 changes neither its algorithm nor those flows; private custom-item validation now composes it in the same way.
- The duplicate scanner intentionally rejects duplicate keys at every nesting depth, including future nested objects. This is stricter than the three named acceptance examples and removes JSON ambiguity consistently.
- The service-counter assertion proves the controller boundary is not crossed. Because repositories are reachable only through that service interface, zero service calls also establishes zero repository calls for these requests.
- A concurrent shared-worktree change separately added duplicate scanning to manual admin items. Task 265 did not author, alter, or rely on that change.
- Full repository aggregate/browser checks were not necessary for this backend-only request-boundary change. Focused preservation tests, the full affected HTTP package, adjacent service/repository packages, vet, and vulnerability scanning passed.
