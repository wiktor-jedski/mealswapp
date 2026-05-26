# Trace: frontend/tsconfig.node.json

`frontend/tsconfig.node.json` cannot contain inline comments because it must remain valid JSON.

## Design Sources

- `docs/design/DESIGN-001.md`: SearchView frontend compilation baseline.
- `docs/design/DESIGN-016.md`: ComponentStyles build-time TypeScript baseline.

## Implemented Surface

- Defines strict shared TypeScript compiler options for the Phase 00 frontend.
- Provides DOM and ES2022 libraries required by the browser SPA and service-worker registration seam.
