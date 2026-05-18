# Repository Guidelines

## Project Structure & Module Organization

This repository is currently organized around requirements, architecture, design specs, and helper scripts. Requirements live in `docs/requirements/`, architecture decisions in `docs/architecture/`, component designs in `docs/design/ARCH-*`, and implementation planning in `docs/implementation/`. Utility scripts are in `scripts/`; keep generated or experimental logs under `logs/`.

When application code is added, follow the documented stack: Svelte frontend code under `src/` or a frontend package, Go/Fiber backend code under a clearly named backend package, and tests colocated with code where the language ecosystem expects them.

## Build, Test, and Development Commands

- `python scripts/check.py`: checks that all software requirement IDs are referenced in `docs/architecture/01_SOFT_ARCH_DESIGN.md`.
- `bash scripts/start-services.sh`: starts local PostgreSQL and Redis services where `service` is available.
- `python scripts/split_arch.py` and `python scripts/phase_splitter.py`: regenerate split documentation from larger planning files when those sources change.

Planned app commands should use `docs/design/01_TECH_STACK.md`: Bun for Svelte (`bun install`, `bun test`, `bun run dev`) and Go tooling for the backend (`go test ./...`, `go run ./cmd/...`) once package manifests exist.

## Coding Style & Naming Conventions

Keep Markdown filenames descriptive and consistent with existing prefixes, for example `ARCH-018.md` or `03_NEW_PLAN.md`. Use requirement IDs exactly as written (`SW-REQ-001`) so scripts can validate traceability. Python helper scripts should stay small, readable, and use snake_case names.

For frontend work, follow `docs/requirements/02_STYLE_GUIDE.md`: Svelte components in `src/lib/components/`, Tailwind utilities, Inter for UI text, Roboto Mono for labels/data, and WCAG AA contrast. For backend Go, use `gofmt` and lower-case package names.

## Testing Guidelines

Run `python scripts/check.py` after editing requirements or architecture docs. Future Go tests should use the standard `testing` package and `Test...` names in `*_test.go` files. Future Svelte tests should use Bun, `@testing-library/svelte`, and Playwright. Add tests near changed behavior, especially around search, auth, subscriptions, and data normalization.

## Commit & Pull Request Guidelines

Recent commits use short summaries such as `scripts update` and `task list update`; keep messages concise and focused on one change. Pull requests should include a brief summary, changed docs or scripts, validation performed, and linked requirements or architecture IDs. Include screenshots for UI changes once the frontend exists.

## Security & Configuration Tips

Do not commit real secrets. Treat `auth.json`, local service credentials, API tokens, Stripe keys, and food data provider keys as local-only configuration. Use GCP Secret Manager for deployed secrets as described in the tech stack.
