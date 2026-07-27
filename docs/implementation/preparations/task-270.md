# Task 270 preparation — Frontend Exported TSDoc and Generator Gate

## Outcome and baseline

Task 270 is prepared against fixed baseline `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.

The initial worktree already contained restored task-candidate edits in `api/openapi.yaml`, `frontend/src/lib/admin-workflows.ts`, `frontend/src/lib/api/admin-client.ts`, `frontend/src/lib/api/generated.ts`, `scripts/generate-api-types.py`, and `scripts/test_generate_api_types.py`, plus an untracked draft of this preparation document. Those edits were preserved and completed. Concurrent Tasks 264–269 changed backend and administration-component paths while this preparation ran; none of those unrelated changes were reverted or included as Task 270 work.

Dependencies 253–257 were inspected through their task rows, preparation evidence, reviews, generated contracts, and Phase 08 frontend source surfaces. All five dependencies are `PASSED`. Task 270 does not edit `docs/implementation/02_TASK_LIST.md` or any task status.

The completed gate now:

- validates every export in the hand-written Phase 08 TypeScript modules introduced by Tasks 254–257;
- supplies concise TSDoc for every previously undocumented export found by that validator;
- renders five administration-contract TSDoc comments from authoritative OpenAPI schema descriptions during generation;
- proves description changes survive regeneration;
- keeps `frontend/src/lib/api/generated.ts` mechanically generated; and
- runs the focused TSDoc validator in the aggregate static lane.

## Exact Task 270 changed paths

| Path | Task 270 change |
| --- | --- |
| `api/openapi.yaml` | Added authoritative descriptions to `AdminClassificationRequest`, `AdminClassification`, and `AdminUser`. Existing descriptions for `AdminItemRequest` and `AdminItem` remain authoritative and unchanged. |
| `frontend/src/lib/admin-workflows.ts` | Added TSDoc to `AdminItemForm`. |
| `frontend/src/lib/api/account-data-client.ts` | Added TSDoc to the previously undocumented Phase 08 Account Export error, injectable API, and API singleton. |
| `frontend/src/lib/api/admin-client.ts` | Added TSDoc to `ClassificationKind`, `AdminMutationOptions`, `AdminClientError`, `AdminApi`, and `adminApi`. |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | Added TSDoc to the hand-written exported compile-time assertion type. |
| `frontend/src/lib/api/generated.ts` | Regenerated only with `python3 scripts/generate-api-types.py`; received OpenAPI-derived TSDoc for five administration declarations. |
| `frontend/src/lib/substitution-filter-options.ts` | Added TSDoc to `SubstitutionFilterOption`. |
| `scripts/check.py` | Registered the focused validator as traceable source and added it to the aggregate static lane. |
| `scripts/generate-api-types.py` | Added administration-description generation and drift enforcement. |
| `scripts/test_generate_api_types.py` | Extended the existing Phase 08 generator test without increasing the required 24-test inventory. |
| `scripts/validate-phase08-tsdoc.py` | Added the focused hand-written Phase 08 export validator. Generated output is deliberately excluded. |
| `docs/implementation/preparations/task-270.md` | Recorded this preparation and evidence. |

## Added or modified symbol inventory

No frontend runtime body, declaration shape, public name, or value changed. Frontend declaration changes are documentation-only.

| Path | Added or modified symbols |
| --- | --- |
| `frontend/src/lib/admin-workflows.ts` | `AdminItemForm` — TSDoc added. |
| `frontend/src/lib/api/account-data-client.ts` | `AccountDataClientError`, `AccountDataApi`, `accountDataApi` — TSDoc added. Existing documented `loadAccountExport` and `deletePrivateCustomItem` are covered by the validator without modification. |
| `frontend/src/lib/api/admin-client.ts` | `ClassificationKind`, `AdminMutationOptions`, `AdminClientError`, `AdminApi`, `adminApi` — TSDoc added. Existing exported item, classification, lookup, and retry functions are covered by the validator without modification. |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | `Phase08SuccessEnvelopeTypeChecks` — TSDoc added; type-only. |
| `frontend/src/lib/substitution-filter-options.ts` | `SubstitutionFilterOption` — TSDoc added; type-only. Existing `substitutionFilterOptions` remains documented and validator-covered. |
| `frontend/src/lib/api/generated.ts` | Generated `AdminItemRequest`, `AdminItem`, `AdminClassificationRequest`, `AdminClassification`, and `AdminUser` declarations received source-derived TSDoc. No executable symbol was added or modified. |
| `scripts/generate-api-types.py` | Added constant `ADMINISTRATION_DESCRIPTION_SCHEMAS`; modified `GENERATED` administration markers and `generated_contract`; added `administration_schema_description` and `administration_description_mismatches`; modified `main` to fail on description/TSDoc drift. |
| `scripts/test_generate_api_types.py` | Modified `OperationResponseDriftTest.test_phase08_routes_security_statuses_and_safe_dtos_match_generated_types` to verify privacy boundaries, source/output agreement, and changed-description regeneration. |
| `scripts/validate-phase08-tsdoc.py` | Added constants `PHASE08_SOURCES`, `EXPORT`, and `TSDOC`; added functions `validate_file` and `main`. |
| `scripts/check.py` | Modified `TRACEABLE_FILES` and `run_static_lane` to include the validator. |

The validator covers all exports in:

- `admin-access.ts`;
- `admin-workflows.ts`;
- `api/account-data-client.ts`;
- `api/admin-client.ts`;
- `api/external-admin-client.ts`;
- `api/filter-options-client.ts`;
- `api/generated.phase08-typecheck.ts`; and
- `substitution-filter-options.ts`.

This includes the task-named `ClassificationKind`, `AdminMutationOptions`, `AdminClientError`, `AdminApi`, and `AdminItemForm`, plus the additional undocumented Phase 08 exports found during the complete audit. It intentionally does not scan generated output or unrelated pre-Phase-08 modules.

## Verification commands and results

Commands ran from the repository root unless noted.

| Command | Result |
| --- | --- |
| `python3 scripts/validate-phase08-tsdoc.py` | **PASS:** every hand-written Phase 08 export has adjacent concise TSDoc. |
| Temporary undocumented-export probe calling `validate_file` | **PASS:** an undocumented `export interface Missing` is rejected. |
| `python3 scripts/generate-api-types.py` | **PASS:** regenerated `frontend/src/lib/api/generated.ts` from current OpenAPI and generator source. |
| `python3 scripts/test_generate_api_types.py` | **PASS:** 24 tests, 0 failures. The existing Phase 08 test proves a changed OpenAPI `AdminItem` description appears in regenerated TSDoc. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | **PASS:** generated API types are current after regeneration. |
| `npx --no-install redocly lint api/openapi.yaml` | **PASS:** valid OpenAPI; one existing accepted OAuth callback `302`-only warning. |
| `cd frontend && ... bun run typecheck` | **PASS:** no TypeScript errors. |
| `cd frontend && ... bun test` | **PASS:** 528 tests, 0 failures, 2,479 expectations. An earlier run observed two transient failures while concurrent Task 266 was updating its component before its test; the final unchanged Task 270 code passed after that concurrent pair settled. |
| `cd frontend && ... bun test src/lib/admin-access.test.ts src/lib/admin-workflows.test.ts src/lib/api/account-data-client.test.ts src/lib/api/admin-client.test.ts src/lib/api/external-admin-client.test.ts src/lib/api/filter-options-client.test.ts src/lib/substitution-filter-options.test.ts src/lib/api/generated.test.ts` | **PASS:** 54 focused tests, 0 failures, 287 expectations. |
| `cd frontend && ... bun run build` | **PASS:** 219 modules transformed and production output built. Concurrent Task 266 currently emits two non-fatal Svelte warnings in `ExternalImportWorkflow.svelte`; Task 270 did not modify that component. |
| `python3 scripts/validate-traceability.py` | **PASS.** |
| Scoped `git diff --check` over every Task 270 implementation path | **PASS.** |

## Criteria evidence

| Criterion | Evidence |
| --- | --- |
| Every hand-written Phase 08 exported type/interface/class/function/constant has concise valid TSDoc | The focused validator scans every export in the eight Phase 08 hand-written modules and passes; its negative probe rejects a missing comment. Added comments cover every gap found, including the five task-named declarations. |
| Generated administration TSDoc comes from OpenAPI/generator source | `ADMINISTRATION_DESCRIPTION_SCHEMAS` maps five generated declarations to schema descriptions; `administration_schema_description` reads each description from its OpenAPI block; `generated_contract` replaces generator-only markers with those values. |
| Generated descriptions survive regeneration | The 24-test generator suite mutates the authoritative `AdminItem` description and asserts the rendered `AdminItem` TSDoc changes with it. `administration_description_mismatches` rejects missing/drifted output. |
| Generated output is not edited directly | The final generated file was written by `python3 scripts/generate-api-types.py`, then `bun run check:api-types` passed byte-for-byte drift verification. |
| Generator, typecheck, unit, and build gates pass | All 24 generator tests, generated drift, typecheck, 528 unit tests, and production build pass as recorded above. |

## Final fingerprints

| Path | SHA-256 |
| --- | --- |
| `api/openapi.yaml` | `02c6657a15ce63d3f5e7f6ea7c35c84938bce6592b2c06dd8fac37f0b268fd9c` |
| `frontend/src/lib/admin-workflows.ts` | `8cbb52e965d4f05c725b424deba76136e52a3f235dddb2281cfd2a8e2822c0c9` |
| `frontend/src/lib/api/account-data-client.ts` | `a26bb587ac5d60d033a054ab043da59ecb67e1646d297760d06ea5aa4383e375` |
| `frontend/src/lib/api/admin-client.ts` | `7e3b4d109dd08d28a5cb7e384c1cf5f60d206b13be86dcc8ccee00ee328a62dc` |
| `frontend/src/lib/api/generated.phase08-typecheck.ts` | `0164c7e304e7117e4eec4dc99b90755e553cbb4ba69a3a3f0115a2ad6d334b1c` |
| `frontend/src/lib/api/generated.ts` | `e08701b57ede329f92e2437411c7599856b4864ca2ce4e282e9dd3e4f16cb264` |
| `frontend/src/lib/substitution-filter-options.ts` | `e0756c0ba1a166b7f4f401d6ea51ff0d9bc57453c3d099e08c9f871c22c7f74e` |
| `scripts/check.py` | `f843434d561d884cc90fd10829dddbccedca02046165483f5dbf27b80cd0b5ea` |
| `scripts/generate-api-types.py` | `71fa504eedd60e2807483e309684938db829f9a283b7f660076f2d4f4a68278e` |
| `scripts/test_generate_api_types.py` | `f8e1291f1433b0764b5ca9c01644a54d3f044872d3eda3e54def4d5f9312bbc6` |
| `scripts/validate-phase08-tsdoc.py` | `7ea9eb16ed2bbaab7234e078e99d972b4dc6b9c54fb697327f5f3b01c55d41da` |

The preparation document omits its own self-referential digest.

## Risks

- The generator remains a checked-in TypeScript template, but administration descriptions are now read from OpenAPI and checked on every generation. Future administration declaration shapes still require deliberate generator-template work.
- The focused validator uses repository-standard-library parsing and intentionally covers the Phase 08 hand-written modules rather than attempting a repository-wide TypeScript rewrite. New exports added to those files are gated automatically; a new Phase 08 module must be added to `PHASE08_SOURCES`.
- The final build passes with two warnings in concurrently modified `ExternalImportWorkflow.svelte` (`alertdialog` element semantics and a non-reactive button binding). They are outside Task 270 and do not affect TSDoc, generated contracts, typecheck, unit tests, or build success.

No blocking Task 270 risk remains.

## Reviewer repair evidence — administration TSDoc association guard

The independent review finding F-270-001 was reproduced against the current worktree before repair. A synthetic swap of the generated `AdminItemRequest` and `AdminItem` comments retained every authoritative OpenAPI description and export but caused `administration_description_mismatches(source, swapped)` to return `[]`. This proved the guard checked only that each description preceded any export, not that it preceded its matching declaration.

The repair is limited to the generator guard, its existing focused Phase 08 generator test, and this evidence document:

| Path | Repaired symbol | Repair |
| --- | --- | --- |
| `scripts/generate-api-types.py` | `administration_description_mismatches` | Replaced the generic description-before-export substring check with an adjacent regular-expression match requiring the exact `export interface` or `export type` schema symbol. |
| `scripts/test_generate_api_types.py` | `OperationResponseDriftTest.test_phase08_routes_security_statuses_and_safe_dtos_match_generated_types` | Added pairwise comment-swap mutations across all five entries in `ADMINISTRATION_DESCRIPTION_SCHEMAS`; every swap must report both affected schema associations. The suite remains exactly 24 tests. |
| `docs/implementation/preparations/task-270.md` | Reviewer repair evidence | Added reproduction, changed-symbol, command/result, fingerprint, and residual-risk evidence. |

No generated TypeScript, OpenAPI contract, frontend runtime source, validator scope, task-list row, or task status was changed by the repair. Concurrent worktree changes were preserved.

### Repair commands and results

| Command | Result |
| --- | --- |
| Read-only Python probe swapping the `AdminItemRequest` and `AdminItem` generated comments | **REPRODUCED:** the pre-repair guard returned `[]`. |
| `python3 -m unittest scripts/test_generate_api_types.py` after adding the regression and before changing the guard | **EXPECTED FAIL:** 24 tests ran; the sole failure showed expected two schema mismatches but received `[]`. |
| `python3 -m unittest scripts/test_generate_api_types.py` after repair | **PASS:** 24 tests, 0 failures, including all 10 pairwise swaps among the five administration schemas. |
| `python3 scripts/generate-api-types.py --check` | **PASS:** checked-in generated API types are current. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | **PASS:** frontend generator drift check reports current output. |
| `python3 scripts/validate-traceability.py` | **PASS.** |
| `git diff --check -- scripts/generate-api-types.py scripts/test_generate_api_types.py` | **PASS.** |

### Repair risks

- The association guard intentionally accepts only the generated declaration forms currently used by the five schemas: an adjacent `export interface` or `export type` with the exact schema name. A future deliberate change to another declaration form will fail closed until the guard is updated.
- The focused mutation uses unique current OpenAPI descriptions. If two administration schemas deliberately receive identical descriptions in the future, a comment swap is semantically indistinguishable; exact symbol adjacency remains enforced for the shared text.
- No runtime, security, generated-output, or contract-shape behavior changed. The repair adds only stricter validation and regression assertions.
