# Open Points

## Phase XX - Template - do not edit, add next after this one

Note: not all fields need to be filled. Don't add unnecessary information.

### Assumptions
- Write here anything that has been assumed about the implementation AND is missing in the design.

### Clarifications
- Write here anything that needs to be clarified - insert here all of your questions.

### Actions needed
- Immediate problems that need to be solved and will block us in the future.

## Phase 00

No project-owner action is required for Phase 00 at this time.

### Assumptions

- The implementation task list at `docs/implementation/02_TASK_LIST.md` is the authoritative task list. The root `docs/02_TASK_LIST.md` file is a pointer to avoid duplicate task status sources. - resolved, removed pointer docs
- Phase 00 uses development-only local credentials from `.env.example` and `docker-compose.yml`; production secrets remain outside the repository. - OK, accepted
- OpenAPI type generation is documented as deferred until domain request and response contracts exist in later phases. - added to phase 02

### Testing coverage deviations

- Resolved for Phase 00 testable source. `python3 scripts/check.py` now enforces 100% coverage for backend internal packages with `go test ./internal/... -coverprofile=coverage.out` and frontend source with `bun test --coverage`.
- Backend `cmd/*` entrypoints remain covered by build/smoke verification rather than line coverage, because they are process bootstrap commands that bind ports, connect to local services, or run migrations.
