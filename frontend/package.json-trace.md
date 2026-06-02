# Trace: frontend/package.json

`frontend/package.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/01_TECH_STACK.md`: Bun, Svelte, Tailwind, and frontend testing toolchain.
- `docs/design/DESIGN-001.md`: SearchView SPA shell dependency on Svelte.
- `docs/design/DESIGN-016.md`: ComponentStyles dependency on Tailwind and frontend build validation.
- `docs/design/DESIGN-017.md`: ErrorMessageMapper shared API error contracts generated from OpenAPI.

## Implemented Surface

- Defines Bun scripts for development, build, test, frontend checks, and OpenAPI contract generation or drift detection.
- Declares Svelte, Vite, Tailwind, TypeScript, Bun test types, and Svelte testing dependencies.
