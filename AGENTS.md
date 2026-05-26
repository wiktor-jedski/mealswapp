# Good practices

1. **Think before you commit.** There is often more than one implementation of an idea. Ask when unsure.

2. **Simplicity first.** Keep as easy to create and explain as possible. Derive quality from simple shapes.

3. **Surgical precision.** Only touch what necessary. Delete and add exactly what needed in each step.

4. **Goal-driven.** Translate fuzzy instructions into SMART goals. Make the outcomes testable.

5. **Growth mindset.** There is always an opportunity to learn and grow if facing mistakes. We can create more tools and skills if necessary. Be mindful of opportunity to improve through conversation.

# Repository Guidelines

## Project Structure & Module Organization

This repository is currently organized around requirements, architecture, design specs, and helper scripts. Requirements live in `docs/requirements/`, architecture decisions in `docs/architecture/`, component designs in `docs/design/ARCH-*`, and implementation planning in `docs/implementation/`. Utility scripts are in `scripts/`; keep generated or experimental logs under `logs/`.

When application code is added, follow the documented stack: Svelte frontend code under the `frontend/` package, Go/Fiber backend code under the `backend/` package, and tests colocated with code where the language ecosystem expects them.

## Build, Test, and Development Commands

- `bash scripts/start-services.sh`: starts local PostgreSQL and Redis with Docker Compose when available, falling back to system `service` commands.

Planned app commands should use `docs/design/01_TECH_STACK.md`: Bun for Svelte from `frontend/` (`bun install`, `bun test`, `bun run dev`) and Go tooling from `backend/` (`go test ./...`, `go run ./cmd/...`) once package manifests exist.

## Coding Style & Naming Conventions

Keep Markdown filenames descriptive and consistent with existing prefixes, for example `ARCH-018.md` or `03_NEW_PLAN.md`. Use requirement IDs exactly as written (`SW-REQ-001`) so scripts can validate traceability. Python helper scripts should stay small, readable, and use snake_case names.

For frontend work, follow `docs/requirements/02_STYLE_GUIDE.md`: Svelte components in `frontend/src/lib/components/`, Tailwind utilities, Inter for UI text, Roboto Mono for labels/data, and WCAG AA contrast. For backend Go, use `gofmt` and lower-case package names.

For backend, follow the official Go Doc comments guidelines.
For frontend, follow the TSDoc comment specification.

Additionally, code must include concise comments that identify the exact `docs/design` source being implemented, for example `// Implements DESIGN-010 RouteHandler` or `<!-- Implements DESIGN-001 SearchView -->`. Place the comment near the module, component, function, type, or generated block it applies to, and keep it specific to the relevant design file and static aspect.

For JSON files, do not add inline comments because they make the file invalid. Instead, add a sidecar traceability document named `{filename}-trace.md`, for example `package.json-trace.md`, that lists the relevant `docs/design` source files and the implemented surface.

## Testing Guidelines

- Future Go tests should use the standard `testing` package and `Test...` names in `*_test.go` files. Future Svelte tests should use Bun, `@testing-library/svelte`, and Playwright. Add tests near changed behavior, especially around search, auth, subscriptions, and data normalization.
- Goal - 100% line coverage by the end of each implementation phase. Each deviation needs to be documented in docs/implementation/04_OPEN.md
- For each phase, during task planning, add relevant integration tests for the newly implemented code AND the code that will work with this phase's code.

Testing commands for the current package layout:

- Root aggregate check: `python3 scripts/check.py`
- Root aggregate check with HTML report: `python3 scripts/check.py --output logs/check-report.html`
- Traceability validation: `python3 scripts/validate-traceability.py`
- Local stack verification: `python3 scripts/verify-local-stack.py`
- Frontend UAT/screenshot verification: `python3 scripts/verify-frontend.py`
- Backend unit tests: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...`
- Backend coverage: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -coverprofile=coverage.out && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=coverage.out`
- Frontend install: `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun install`
- Frontend build: `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build`
- Frontend unit tests: `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test`
- Frontend coverage: `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage`
- Local dependencies for integration checks: `bash scripts/start-services.sh`
- Backend migrations: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up`
- Backend API smoke test: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api`, then check `/health` and `/ready`.

`scripts/check.py` runs requirement traceability, design traceability, local stack verification, frontend UAT/screenshot verification, backend formatting/tests/coverage, and frontend build/tests/coverage. The local stack verifier requires Docker Compose. The frontend verifier requires a local Chromium-compatible browser (`chromium`, `chromium-browser`, or `google-chrome`) and writes temporary screenshots under `/tmp/mealswapp-frontend-verifier/`. When `scripts/check.py --output <report>.html` is used, screenshots are copied next to the report under `screenshots/` using the report stem, for example `<report>-desktop.png` and `<report>-mobile.png`.

Each completed phase needs to have user acceptance document in docs/implementation/implemented/{x:02d}_PHASE_UAT.md, where x is the number of the phase. The document is a recap of the changes implemented and suggests relevant acceptance tests.

## Commit & Pull Request Guidelines

Recent commits use short summaries such as `scripts update` and `task list update`; keep messages concise and focused on one change. Pull requests should include a brief summary, changed docs or scripts, validation performed, and linked requirements or architecture IDs. Include screenshots for UI changes once the frontend exists.

## Security & Configuration Tips

Do not commit real secrets. Treat `auth.json`, local service credentials, API tokens, Stripe keys, and food data provider keys as local-only configuration. Use GCP Secret Manager for deployed secrets as described in the tech stack.

Do not commit generated artifacts or local caches, including `frontend/dist/`, root `dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/`, `backend/.go-cache/`, `backend/.go-mod-cache/`, and `backend/coverage.out`.
