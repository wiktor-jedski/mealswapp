# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- `CONTEXT.md` at the repo root when it exists.
- Relevant architecture files under `docs/architecture/`.

If `CONTEXT.md` does not exist, proceed silently.

## Architectural decisions

Architectural decisions belong under `docs/architecture/`. Follow the existing repository convention and add new `ARCH-xxx.md` files when a new decision needs to be recorded. Do not create or use `docs/adr/`.

## Use the glossary's vocabulary

Use terms defined in `CONTEXT.md` when it exists. If a required concept is missing, reconsider the terminology or note the gap for documentation work.

## Flag architecture conflicts

If proposed work contradicts an existing `docs/architecture/ARCH-xxx.md` file, surface the conflict explicitly instead of silently overriding it.
