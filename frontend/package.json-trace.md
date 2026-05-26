# Trace: frontend/package.json

`frontend/package.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/01_TECH_STACK.md`: Bun, Svelte, Tailwind, and frontend testing toolchain.
- `docs/design/DESIGN-001.md`: SearchView SPA shell dependency on Svelte.
- `docs/design/DESIGN-016.md`: ComponentStyles dependency on Tailwind and frontend build validation.

## Implemented Surface

- Defines Bun scripts for development, build, test, and frontend checks.
- Declares Svelte, Vite, Tailwind, TypeScript, Bun test types, and Svelte testing dependencies.
