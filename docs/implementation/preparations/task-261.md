# Task 261 preparation — Phase 08 SWE.5 integration verification

## Outcome and repair scope

- Task: 261, `ARCH-009: AdminController`; fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Task status observed: `PREPARED`. `docs/implementation/02_TASK_LIST.md` was not edited during this repair; current SHA-256: `304f2622185cd6c7dcb83e25679866b8de6ba84fca840ef6a2faf7f3be8850ce`.
- Review findings `F-261-01` and `F-261-02` are repaired while all previously documented ARCH-009 and ARCH-012 obligations and evidence remain in place.
- Unrelated dirty-worktree changes were preserved.

## Repaired executable evidence

### F-261-01 — IT-ARCH-009-002 real provider-to-persistence path

`backend/internal/app/task261_external_import_integration_test.go::TestTask261ProviderHTTPImportPostgresFlow` executes this integrated sequence:

1. deterministic USDA and OpenFoodFacts HTTP servers receive real provider-client requests;
2. production USDA/OpenFoodFacts clients feed the real rate-limited `ExternalSearchProxy` and `DataNormalizer`;
3. an authenticated admin invokes the real Fiber external-search controller;
4. the test proves external search is read-only, retains a valid USDA candidate, and emits a bounded partial-provider warning for OpenFoodFacts;
5. an explicit authenticated/CSRF-protected import invokes the real curated-import HTTP controller and `DataImporter`;
6. PostgreSQL atomically contains exactly one food, import, and audit record; and
7. the imported identity is returned by the real catalog HTTP search.

The test uses no provider adapter, proxy, controller, importer, repository, or database mock. Only the two external provider HTTP origins are deterministic boundary servers. Production composition accepts a package-local provider override solely to make this executable boundary deterministic; `NewProduction` behavior is unchanged.

### F-261-02 — IT-ARCH-009-004/-005 generated-client browser path

`scripts/verify-task-261-ui.sh` starts PostgreSQL, Redis, migrations, the real API, Vite, and Chromium, then executes `frontend/tests/task261-real-admin-flow.spec.ts`. The evidence test uses no Playwright route interception. It:

1. authenticates a verified admin against the live API;
2. opens the rendered `AdministrationPanel` and `AdminPrivateData` component;
3. loads an owner-safe account export through generated request builders;
4. confirms private-item deletion through the UI and generated CSRF/deletion client;
5. observes the authoritative export-backed empty state after the real HTTP 204 and PostgreSQL deletion;
6. creates a classification and global item through the rendered Administration Panel; and
7. selects the persisted item in Substitution mode and finds the newly persisted classification through the generated dynamic-filter client.

The old direct-fetch/stub browser case remains only as supporting transport regression evidence and no longer claims IT-ARCH-009-004 coverage.

## Obligation traceability

| Obligation | New primary evidence | Preserved supporting evidence |
| --- | --- | --- |
| IT-ARCH-009-002 | `TestTask261ProviderHTTPImportPostgresFlow` | DataImporter transaction and external-import browser contract suites |
| IT-ARCH-009-004 | real-stack `Admin Panel generated client deletes exported private data and publishes a dynamic filter` | `TestTask240CustomItemErasureIntegration` and owner-safe client regressions |
| IT-ARCH-009-005 | same real-stack Administration Panel/dynamic-filter test | classification HTTP, live Redis generation, PostgreSQL filter-option, and prior browser suites |
| IT-ARCH-012-001 | `TestTask261ProviderHTTPImportPostgresFlow` | provider-client, proxy, normalizer, and browser contract suites |
| IT-ARCH-012-002 | `TestTask261ProviderHTTPImportPostgresFlow` partial-provider branch | outage/rate-limit/normalization warning suites |

IT-ARCH-009-001 and -003 through -007 and IT-ARCH-012-001 through -003 remain documented as PASS in the obligation documents. Their authorization, isolation, replay, conflict, rollback, deletion, invalidation, provider, normalization, UI, and degraded-path evidence was preserved.

## Verification results

| Command | Result |
| --- | --- |
| `cd backend && ... go test -p 1 -count=1 ./internal/externaldata ./internal/httpapi ./internal/dataimporter ./internal/cache ./internal/search ./internal/app ./internal/repository` | PASS; all seven packages, including the real task-261 PostgreSQL flow. `-p 1` prevents concurrent database-reset fixtures from contending. |
| `cd backend && ... go test -race -p 1 -count=1 ./internal/app ./internal/repository` | PASS. Earlier concurrently launched broad validation processes were stopped after database-reset contention and replaced by this successful serial run. |
| `cd backend && ... go vet ./...` | PASS. |
| `cd frontend && ... bun test` | PASS: 526 tests. |
| `cd frontend && ... bun run typecheck` | PASS. |
| `cd frontend && ... bun run build` | PASS. |
| `cd frontend && ... bunx playwright test tests/admin-access-shell.spec.ts tests/external-import-workflow.spec.ts tests/admin-data-management.spec.ts tests/task259-frontend-gate.spec.ts` | PASS: preserved 46/46 desktop/mobile Chromium cases. Expected proxy diagnostics occur only where legacy fixture suites intentionally leave unrelated reads unstubbed. |
| `bash scripts/verify-task-261-ui.sh` | PASS: 1/1 real-stack desktop Chromium case; no route stubs. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated client is current. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the existing OAuth 302/no-2XX warning. |
| `git diff --check` | PASS. |

`python3 scripts/test_generate_api_types.py` has one pre-existing, unrelated failure: the dirty OpenAPI source does not contain the older test's phrase `Quantity-weighted Jaccard similarity`. The generator's executable current-output check passes, and this repair does not alter that optimization wording or its assertion.

## SWE.5 checklist decision

| Area | Result |
| --- | --- |
| Architecture and requirement traceability | PASS — ARCH-009/ARCH-012, DESIGN-009/DESIGN-012, and required SW-REQ links remain explicit. |
| Cross-component integration | PASS — real HTTP, provider clients, proxy, normalization, controller, importer, PostgreSQL, generated clients, Svelte components, Redis, and Chromium are exercised. |
| Test-double boundary | PASS — new primary evidence doubles only external provider HTTP origins and fixture setup; evidence actions use real application components. |
| Observable outcomes | PASS — provider calls, warnings, read-only precondition, HTTP status/envelopes, PostgreSQL counts/audit, local search identity, rendered export state, deletion, and dynamic filter are asserted. |
| Failure and recovery | PASS — partial-provider success, explicit persistence confirmation, preserved rollback/replay suites, and authoritative UI refresh remain covered. |
| Obligation coverage | PASS — no ARCH-009 or ARCH-012 obligation was removed or weakened. |

## Source and evidence hashes

| Path | SHA-256 |
| --- | --- |
| SWE.5 `SKILL.md` | `fd19f5f6b1fddf13364ae89d48d7f6b0fd10f663a81aafb571905c7a3850f2aa` |
| SWE.5 `CHECKLIST.md` | `1f5393a352ed840e78c2541aa85ac056b400a9a03d04be726a4744666a58ca9f` |
| `docs/architecture/ARCH-009.md` | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/architecture/ARCH-012.md` | `8377243ff9409b27ac9c43de556f0ff094c163ca465abe562ee8cec5721ed435` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/design/DESIGN-012.md` | `53ac9bd6a34bd07216666d4beaae6533a0281c905fc2d5c474f48f614746eddf` |
| `docs/requirements/01_SOFT_REQ_SPEC.md` | `80b2f57a8c1caebd8b37cdb949cc7e928f3a128a2b8ed81313637b919cafba8b` |
| `docs/testing/integration/ARCH-009-obligations.md` | `d9e80c26c298f9be72b71ecd3728a584ea92d7f8cc0c2f5816828d68f6837c1c` |
| `docs/testing/integration/ARCH-012-obligations.md` | `c96ea513c1a33bd74999dd2b1c47f755802c0a007117b4693afef6ce5407e85c` |
| `backend/internal/app/app.go` | `4a32fe296885145876d71c01c35a32584d89bc8f52271d2851c0d84ef17281b9` |
| `backend/internal/app/task261_external_import_integration_test.go` | `1a0f313dd77963b376021742133650a027deaae684e6cae14a3d19bb5fe4e963` |
| `frontend/src/lib/api/generated.ts` | `f732d86079c10056959292ad2dea3c0163b83b43185620169cd243e074c7829a` |
| `frontend/src/lib/api/account-data-client.ts` | `57b72c23d05b939dae54f512c0eb011f524547116889b86bfc58dd0ab51423a4` |
| `frontend/src/lib/api/account-data-client.test.ts` | `5bbea3f4225f42a4986f84f08a9fe4eab6519ccef304abfe24b25698e2da6588` |
| `frontend/src/lib/components/AdminPrivateData.svelte` | `0acdf79df7617030b228c8578bc02474b841857f8aeaa7363b41b38de5761719` |
| `frontend/src/lib/components/AdminPrivateData.test.ts` | `eaac9091e7d27e4e238f401fb82d2df20e15f7bf926d5a91c6f549afe106595b` |
| `frontend/src/lib/components/AdministrationPanel.svelte` | `efac4fb695fbfc66013d413ae0fc42b67b3afca9d525944d344265a47d9f72f2` |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | `9c02c68fd5a4254415b05f26b9a23e2c7d564d154646ed24fc900e1ddd8e78fa` |
| `frontend/tests/task261-real-admin-flow.spec.ts` | `f41fdad1a762ea93a42a394c5608e750d11e3ee3c1f77a2b92fb24f18949c556` |
| `frontend/tests/task259-frontend-gate.spec.ts` | `4e7a71528e59c4586484b0bcd9a33779f157f479a3ba17bfb98871fd6495ba84` |
| `scripts/generate-api-types.py` | `b2f7b6faa0fd7fb8c53762e7db5b42403459a3e946878c8425f70ddecc9605ab` |
| `scripts/verify-task-261-ui.sh` | `2c200a1f762f5ade74427c1fa1f32b22b2e9abc404f5d9846e0c9aa057003170` |
