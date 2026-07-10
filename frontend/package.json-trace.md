# Trace: frontend/package.json

`frontend/package.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/01_TECH_STACK.md`: Bun, Svelte, Tailwind, and frontend testing toolchain (Bun test runner + `@testing-library/svelte` + Playwright; Svelte stores + TanStack Query state management).
- `docs/design/DESIGN-001.md`: SearchView SPA shell dependency on Svelte and TanStack Query server-state orchestration for catalog/substitution/daily-diet search.
- `docs/design/DESIGN-016.md`: ComponentStyles dependency on Tailwind and frontend build validation.
- `docs/design/DESIGN-017.md`: ErrorMessageMapper shared API error contracts generated from OpenAPI.
- `docs/design/DESIGN-018.md`: AuthApiClient generated-contract dependency for auth, session recovery, entitlement refresh, checkout start flows, and production TypeScript compatibility.

## Implemented Surface

- Defines Bun scripts for development, build, preview, test, mocked end-to-end browser tests, opt-in real-stack browser UAT, frontend checks, production TypeScript validation, and OpenAPI contract generation or drift detection.
- Declares Svelte, Vite, TanStack Svelte Query, Tailwind, TypeScript, Bun test types, Svelte testing, Playwright, and axe-core Playwright dependencies.
- `dev` and `preview` serve the SPA for local development and Playwright browser tests respectively; `preview` pins port 4173 with `--strictPort` so the Playwright `webServer` polls a deterministic URL.
- `test` runs deterministic Bun unit/component tests scoped to `src/` via `frontend/bunfig.toml` (`[test] root = "src"`), keeping Playwright specs under `frontend/tests/` out of the Bun runner.
- `test:e2e` runs `playwright test`, executing Chromium desktop and mobile projects defined in `frontend/playwright.config.ts` against the built app served by `bun run build && bun run preview`.
- `test:e2e:real-stack` runs the opt-in `frontend/playwright.real-stack.config.ts` scenario against the Vite dev proxy and a separately running local backend stack, proving real auth cookies and checkout handoff without mocked API routes.
- `typecheck` compiles production TypeScript through `tsconfig.typecheck.json`, excluding test fixtures and Vite configuration so generated DTO, envelope, header, and `fetch` compatibility failures block the frontend gate.
- `check` runs the contract-drift, production typecheck, build, and unit-test gates without requiring Playwright browser binaries.
- `generate:api-types` and `check:api-types` cover the DESIGN-018 auth/session contract helpers consumed by the frontend auth UI.

## Phase 05 Frontend Search Tooling (Task 139)

- Added dependency: `@tanstack/svelte-query` (search-client server-state library for the SearchView query orchestration described in DESIGN-001 step 6).
- Added devDependencies: `@playwright/test` (browser test runner for the Playwright toolchain named in `01_TECH_STACK.md`) and `@axe-core/playwright` (automated axe accessibility scans).
- Added scripts: `preview` (deterministic preview server on port 4173) and `test:e2e` (Playwright command).
- Added `frontend/bunfig.toml` to scope Bun unit/component tests to `src/` so Playwright specs under `frontend/tests/` run only via `test:e2e`.
- Added `frontend/playwright.config.ts` with Chromium desktop and mobile projects and a `webServer` that builds and previews the app.
- Added `frontend/tests/smoke.spec.ts` browser smoke fixture that intercepts `/api/v1/search` and `/api/v1/search/autocomplete` with contract-valid controlled responses (typed against `src/lib/api/generated.ts`) and asserts the SearchView shell renders plus an axe smoke check reports no serious or critical violations.
