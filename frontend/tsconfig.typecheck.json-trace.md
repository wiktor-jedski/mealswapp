# Trace: frontend/tsconfig.typecheck.json

`frontend/tsconfig.typecheck.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/DESIGN-018.md`: AuthApiClient generated DTO, envelope, request-helper, and browser fetch compatibility.
- `docs/design/DESIGN-017.md`: ErrorMessageMapper shared envelope and error-contract compilation.

## Implemented Surface

- Extends the frontend TypeScript configuration while disabling composite output for a read-only validation command.
- Includes production TypeScript modules under `src/` and excludes `*.test.ts` fixtures.
- Provides a focused compile gate for generated contracts and their real API-client consumers without expanding this repair into unrelated test-fixture or Vite configuration typing.
