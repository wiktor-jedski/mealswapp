# Task 253 preparation — Admin and External-Data OpenAPI Contract

## Outcome

Task 253 (`DESIGN-009: AdminController`) has been repaired for `F-253-003` from `docs/implementation/reviews/task-253-review.md`. The task-list row remains `PREPARED`; `docs/implementation/02_TASK_LIST.md` was not edited during this repair.

`phase08_contract_mismatches()` now validates every schema in `PHASE08_SUCCESS_ENVELOPES` at the OpenAPI source boundary. Each of the nine schemas must exist, be a closed object, require exactly `status`, `requestId`, and `data`, expose exactly those top-level properties, and declare `status` as `const: ok`.

The source mutation regression covers all nine envelopes, explicitly including `CustomItemEnvelope` and `AdminClassificationEnvelope`. For every envelope it proves that changing the object type, allowing additional properties, removing required `data`, replacing `status.const: ok`, renaming `data`, or deleting the schema is rejected.

## Repair surface

| Path | Task 253 repair |
| --- | --- |
| `api/openapi.yaml` | Align `CustomItemEnvelope` with the other eight strict success envelopes by adding `additionalProperties: false` and replacing `enum: [ok]` with `const: ok`. Payload schemas and response references are unchanged. |
| `scripts/generate-api-types.py` | Add source-schema assertions for all nine Phase 08 success envelopes and add the previously omitted `CustomItemEnvelope` and `AdminClassificationEnvelope` required markers. |
| `scripts/test_generate_api_types.py` | Assert generated marker coverage through the complete envelope inventory and add six source mutations for each of the nine schemas. |
| `frontend/src/lib/api/generated.ts` | Unchanged; the existing nine `OkEnvelope<TData>` aliases remain current. |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | Unchanged; compile-time strict-envelope assertions remain current. |

The nine guarded schemas are `CustomItemEnvelope`, `FilterOptionsEnvelope`, `ExternalSearchEnvelope`, `CuratedImportEnvelope`, `AdminItemEnvelope`, `AdminClassificationEnvelope`, `AdminClassificationCollectionEnvelope`, `AdminUserPageEnvelope`, and `AdminDeletionRetryEnvelope`.

## Regression evidence

The new mutation test was run before the guard and failed for every mutation. It also identified that `CustomItemEnvelope` was the only source schema not already closed with `const: ok`. After the repair:

| Command | Result |
| --- | --- |
| Focused Phase 08 source, generated-envelope, and route-contract generator tests | PASS (`3/3` test methods; all 54 per-envelope mutation subtests pass). |
| `python3 -m unittest scripts/test_generate_api_types.py` | Task 253 tests PASS; module is `23/24` because of the pre-existing unrelated optimization assertion requiring `Quantity-weighted Jaccard similarity`. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types are current. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS; one known OAuth callback 302-only warning remains. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS, including `generated.phase08-typecheck.ts`. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS; 208 modules transformed. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | PARTIAL/FAIL outside Task 253. Traceability, task-list validation, OpenAPI lint, security checks, local-stack checks, UAT checks, frontend verification, and Task 253-relevant backend packages pass. The aggregate fails in the unrelated `TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`. |

## Preservation checks

- The eight already strict success schemas retain their source definitions and payload shapes.
- `CustomItemEnvelope.data` still references `CustomItem`; only its outer success-envelope strictness changed.
- Generated TypeScript is byte-for-byte unchanged and continues to use strict `OkEnvelope<TData>` for all nine aliases.
- Route/status, authentication, CSRF, idempotency, retry, warning, pagination, ownership, privacy, and error contracts were not changed.
- No backend code, runtime test, architecture file, requirement file, or task-list status was edited for this repair.
- Concurrent shared-worktree changes were preserved and were not reformatted, reverted, staged, or attributed to task 253.

## Current fingerprints

| Path | SHA-256 |
| --- | --- |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | `5a8d415198b975e12dcc003ed537f3007843583e3c3dbf80ca81c0b4d015aee3` |
| `scripts/generate-api-types.py` | `a2d4b6ab3b41862233c531f2738f861c256cdedcab5ed7963915e89cb6384721` |
| `scripts/test_generate_api_types.py` | `f8317f543a0eb730d837d2350d131ba431534c0bd5eb08747dec34631536e3ef` |
| `backend/internal/security/curation_normalizer_test.go` | `28ea79df82b789d677cd5a4f1649afb51311e757cd936782fe3b7b3e1191b749` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/implementation/02_TASK_LIST.md` | `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151` |
