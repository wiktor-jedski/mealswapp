# Trace: frontend/tsconfig.json

`frontend/tsconfig.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/DESIGN-001.md`: SearchView TypeScript/Svelte source compilation.
- `docs/design/DESIGN-016.md`: ComponentStyles and ThemeProvider TypeScript support for frontend state and styling modules.

## Implemented Surface

- Enables TypeScript checking for `frontend/src/**/*.ts`, `frontend/src/**/*.svelte`, browser fixtures, Playwright config, and Vite config.
- Includes Bun types for frontend unit tests, browser tests, and bootstrap scripts.
- Uses bundler module resolution for the Svelte/Vite application.
