# Good practices

1. **Think before you commit.** There is often more than one implementation of an idea. Ask when unsure.

2. **Simplicity first.** Keep as easy to create and explain as possible. Derive quality from simple forms and actions.

3. **Surgical precision.** Only touch what necessary. Delete and add exactly what needed in each step.

4. **Goal-driven.** Translate fuzzy instructions into SMART goals. Make the outcomes testable.

5. **Growth mindset.** There is always an opportunity to learn and grow if facing mistakes. We can create more tools and skills if necessary. Be mindful of opportunity to improve through conversation.

# Repository Guidelines

## Project Structure & Module Organization

This repository is organized around:
- **Documentation**: Requirements in `docs/requirements/`, architecture decisions in `docs/architecture/`, component designs in `docs/design/`, and implementation planning in `docs/implementation/`.
- **Frontend**: A Vite + Svelte + TypeScript project under `frontend/` powered by Bun.
- **Backend & Worker**: A Go + Fiber backend API and background worker application under `backend/` with entry points in `backend/cmd/api` and `backend/cmd/worker`.
- **Database**: PostgreSQL migrations under `db/migrations/`.
- **Deployment**: GCP deployment configuration in `deploy/`.
- **Scripts & Logs**: Utility scripts in `scripts/` and logs under `logs/`.

Application code must follow this layout. Put frontend code under `frontend/src/`, Go backend code under `backend/internal/`, and tests colocated with code within their respective subdirectories.

## Build, Test, and Development Commands

- `python scripts/check.py`: the primary local CI check script. It validates documentation traceability, Docker Compose config, backend Go tests, frontend Bun tests/build, and PostgreSQL migrations. Always run this before submitting work.
- `bash scripts/start-services.sh`: starts local PostgreSQL and Redis system services. Alternatively, use `docker compose up -d` in the root directory to run containerized dependencies.
- **Frontend Commands** (run in `frontend/`):
  - `bun install`: installs dependencies.
  - `bun test`: runs Vitest component and unit tests.
  - `bun run dev`: starts the Vite development server (listening on all interfaces, port 5173).
  - `bun run build`: compiles production bundles.
  - `bun run check`: runs svelte-check on the typescript/svelte files.
- **Backend Commands** (run in `backend/`):
  - `go test ./...`: runs all tests with tests colocated in their packages. (Note: use `GOCACHE=backend/.go-cache` if the default build cache is unwritable).
  - `go run ./cmd/api`: runs the Fiber REST API server (listening on port 8080 or `API_ADDR`).
  - `go run ./cmd/worker`: runs the background task runner.
  - `DATABASE_URL=... go run ./cmd/seed`: applies idempotent database seed data.

## Coding Style & Naming Conventions

Keep Markdown filenames descriptive and consistent with existing prefixes, for example `ARCH-018.md` or `03_NEW_PLAN.md`. Use requirement IDs exactly as written (`SW-REQ-001`) to satisfy traceability validations in `scripts/check.py`. Python helper scripts should stay small, readable, and use snake_case names.

For frontend work, follow `docs/requirements/02_STYLE_GUIDE.md`: Svelte components in `frontend/src/lib/components/`, Tailwind utilities, Inter for UI text, Roboto Mono for labels/data, and WCAG AA contrast. For backend Go, use `gofmt` to format code, lowercase package names, and place domain/application logic under `backend/internal/`.

## Testing Guidelines

Run `python scripts/check.py` after editing requirements, architecture docs, database migrations, backend code, or frontend code to guarantee overall repository integrity.
- **Go Backend**: Use Go's standard `testing` package, naming tests `Test...` in `*_test.go` files colocated with the code. For integration tests requiring a database connection, use the `MEALSWAPP_TEST_DATABASE_URL` environment variable.
- **Svelte Frontend**: Use Vitest, Svelte component testing utilities (`@testing-library/svelte`), and TypeScript-based unit tests. Colocate tests inside `frontend/src/`.
- Add comprehensive test coverage near all modified behaviors, especially around search, auth, subscriptions, data normalization, and LP constraints/solver logic.

## Commit & Pull Request Guidelines

Keep commit messages concise, focusing on one logical change at a time (e.g., `external search provider setup`, `frontend gating styling`). Ensure all pull requests contain a summary of accomplishments, manual validation steps, and links to relevant requirement/architecture IDs (`SW-REQ-XXX` or `ARCH-XXX`). Link screenshots or recordings for UI changes in the frontend.

## Security & Configuration Tips

Never commit real secrets or production credentials. Store local configurations in `.env` (ignored by git). Treat session keys, Stripe private keys, USDA and OpenFoodFacts API keys, database credentials, and OAuth secrets as strictly local-only configuration. Use GCP Secret Manager for staging and production secrets.

