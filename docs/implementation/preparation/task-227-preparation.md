# Task 227 Preparation — Shared Runtime-Safe Client Error Mapping

## Scope and attribution

- Task: `227`, Phase 07.01 Shared Runtime-Safe Client Error Mapping.
- Design source: `docs/design/DESIGN-017.md`, static aspect `ErrorMessageMapper`.
- Review-action source: the Phase 07 frontend/architecture action in `docs/implementation/04_OPEN.md` requiring one runtime-safe mapper for Daily Diet and optimization clients.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- The worktree already contained concurrent Phase 07.01 changes in the OpenAPI contract, generated contracts, both scoped clients/tests, generator, backend, and planning documents. Those changes were preserved. Task 227 changed only the shared mapper boundary, its two client integrations, focused tests, AppError drift enforcement, and this preparation evidence.
- Task 227 remains `OPEN`. No task-list status or other task row was changed.
- Strict success-envelope decoding, Daily Diet operation/idempotency ownership, and optimization response/submission-key ownership remain Tasks 228 and 230.

## Implemented contract

### One unknown-envelope boundary

`mapErrorMessage(scope, status, envelope: unknown)` is the only Daily Diet and optimization API-error parser. Both clients pass raw JSON values to it and no longer cast an unknown error body to generated `Envelope`/`AppError` types or maintain local category, code, message, retryability, and request-ID validators.

The mapper accepts only object-shaped envelopes and errors. An approved mapping requires the exact client scope, HTTP status, code, category, string message field, and boolean `retryable` field. Unknown or malformed values select a fixed status fallback. Server message text is never rendered: approved and fallback paths both return mapper-owned text.

### Fixed approved vocabulary

The shared table covers the current generated-contract policies for validation, authentication, entitlement, dependency, rate limiting, timeout, and bounded generic server failures. Approved server codes are retained only at their documented status/category combination. A syntactically plausible but unapproved code cannot become UI vocabulary.

Daily Diet `403` and `404` bypass source classification and always produce the same `security / daily_diet_unavailable / Saved daily diet is unavailable.` projection. This keeps missing and cross-user resources indistinguishable while retaining a separately validated correlation request ID when one exists.

### Retryability and request IDs

Approved envelope retryability is accepted only when its runtime value is a boolean. A string, number, null, missing value, or other malformed value selects the status policy instead. Every fallback owns an explicit boolean.

Request IDs are optional and limited to 1–120 printable correlation-token characters: ASCII letters, digits, `.`, `_`, `:`, and `-`, with an alphanumeric first character. Whitespace, controls, empty values, and oversized values are discarded. A valid error-level request ID takes precedence over the envelope-level value.

### Generated-contract drift

The API type generator now rejects drift in the OpenAPI `AppError` required fields, exact category enum, boolean `retryable`, and string `requestId` declaration before checking or writing generated output. Python regression tests mutate category and retryability declarations and prove the weakened contracts are rejected.

## Changed Task 227 surfaces

| Path | Task 227 surface |
| --- | --- |
| `frontend/src/lib/api/error-message-mapper.ts` | shared approved-code/status table, unknown-envelope parsing, fixed safe messages, strict boolean retryability, bounded request IDs, and Daily Diet ownership-safe projection |
| `frontend/src/lib/api/error-message-mapper.test.ts` | table-driven approved mappings, malformed fields, stack/SQL/Redis/provider/URL/credential/oversize/control text, request-ID boundaries, unknown statuses, and 403/404 equivalence |
| `frontend/src/lib/api/daily-diet-client.ts` | shared mapper import for response errors, delete status fallback, and propagated CSRF errors; duplicated local policy removed |
| `frontend/src/lib/api/daily-diet-client.test.ts` | fixed-message and oversized-request-ID client-boundary regression, retaining existing Phase 07 response-matrix work |
| `frontend/src/lib/api/optimization-client.ts` | shared mapper import for response and propagated CSRF errors; duplicated local policy removed; pre-existing terminal-result normalization retained |
| `frontend/src/lib/api/optimization-client.test.ts` | existing table-driven audited status coverage exercises the shared mapper unchanged |
| `scripts/generate-api-types.py` | `AppError` source-contract drift validation, layered onto the pre-existing Phase 07 operation/unit generation work |
| `scripts/test_generate_api_types.py` | current-contract and deliberate category/retryability drift regressions, preserving pre-existing response-matrix tests |
| `docs/implementation/preparation/task-227-preparation.md` | this scoped implementation and verification evidence |

## Verification-criteria mapping

| Task 227 criterion | Evidence | Result |
| --- | --- | --- |
| Both clients import one mapper | Daily Diet and optimization clients import `mapErrorMessage`; duplicated `safeErrorFromSource`, status tables, and validators are removed | PASS |
| Unknown envelopes are parsed at runtime | mapper public input is `unknown`; object/error shape and primitive field types are checked before rule selection | PASS |
| Retryability is strictly boolean | malformed `"false"` selects the optimization 400 fallback; generator rejects non-boolean OpenAPI drift | PASS |
| Request IDs are bounded | table accepts exactly 120 safe characters and rejects empty, 121-character, whitespace, newline, NUL, and unsafe values | PASS |
| Messages are safe | all rendered messages are fixed table text; stack traces, SQL/PostgreSQL, Redis, provider names, URLs, credentials, oversized strings, and controls are ignored | PASS |
| Approved policy is stable | table-driven validation/auth/entitlement/dependency/rate-limit/timeout cases retain approved category, code, fixed message, retryability, and request ID | PASS |
| Daily Diet ownership is safe | direct mapper and client tests prove 403 and 404 are identical and do not expose source diagnostics | PASS |
| Generated types cannot silently weaken the boundary | generator contract check plus deliberate mutation tests cover category and retryability drift; generated output check passes | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `frontend/` | `bun test src/lib/api/error-message-mapper.test.ts` before implementation | EXPECTED FAIL: shared module did not exist |
| `frontend/` | `bun test src/lib/api/daily-diet-client.test.ts` before client migration | EXPECTED FAIL: raw `postgres password=secret` message crossed the old client boundary |
| `frontend/` | `bun test src/lib/api/error-message-mapper.test.ts --coverage` | PASS; mapper 100% functions and 100% lines |
| `frontend/` | `bun test src/lib/api/error-message-mapper.test.ts src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts` | PASS; 20 tests after final boundary additions |
| `frontend/` | `bun run check` | PASS; generated drift check, TypeScript typecheck, production build, and all 371 frontend tests |
| repository root | `python3 -m unittest scripts/test_generate_api_types.py` | PASS; 9 drift tests |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies; Task 227 remains `OPEN` |
| repository root | `git diff --check` | PASS |
| repository root | `python3 scripts/validate-traceability.py` | FAIL due only to pre-existing concurrent Task 224 declarations in `backend/internal/queue/job_queue.go` at lines 78, 465, 593, 604, 764, 844, and 848; no Task 227 frontend or generator finding |

## Current SHA-256 snapshot

| Path | SHA-256 | Attribution |
| --- | --- | --- |
| `frontend/src/lib/api/error-message-mapper.ts` | `7a5aa10e5029e001ec33a648d2f1762e7504508cb2525933695a563ededf0f5e` | Task 227 shared production mapper |
| `frontend/src/lib/api/error-message-mapper.test.ts` | `aff0fd048b0034916a63774c225eaa609edc79585a31c55570819ddeb34c7df1` | Task 227 table-driven mapper evidence |
| `frontend/src/lib/api/daily-diet-client.ts` | `2e1bbad5ce856b0beb64f859bac99d462d970313a28e13f1615fa1b5daa3554c` | shared client with Task 227 mapper integration and preserved prior work |
| `frontend/src/lib/api/daily-diet-client.test.ts` | `ebe875f56aa09aff558965de72285dda79feac677a4d7b15ba0c2e51634783d7` | shared tests with Task 227 safety regression and preserved prior work |
| `frontend/src/lib/api/optimization-client.ts` | `d1bc3b4944c6dc3ff5dc7bc2bd7fd31b5d3d948d5d0f1654c5f777f800569666` | shared client with Task 227 mapper integration and preserved prior work |
| `frontend/src/lib/api/optimization-client.test.ts` | `fab6abd590530acaf2d735ffc8c35f787d880f10bc6c1d887e70f83332aad744` | preserved prior table-driven client evidence exercising Task 227 mapper |
| `scripts/generate-api-types.py` | `17145445e24ccb0b1a807c251f771a52cac1cd659fced51ab7f5c31eb3e962c1` | shared generator with Task 227 AppError drift check and preserved prior work |
| `scripts/test_generate_api_types.py` | `a0a96fff54ac95d23aa56cd8ccedc4fbb8293d97a9e0f5253263aadd43417b21` | shared drift suite with Task 227 tests and preserved prior work |
| `docs/design/DESIGN-017.md` | `5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c` | preserved design source |
| `docs/implementation/02_TASK_LIST.md` | `3641b4740cc3c5e40b23740e2a090ff26d75ca9f6b19663d3c15b476c793b779` | preserved shared task/status source; Task 227 remains `OPEN` |

This preparation document intentionally omits its own hash.

## Residual boundaries

- Task 228 owns exact Daily Diet success-envelope decoding, endpoint status policy, API simplification, and user-operation idempotency-key lifetime.
- Task 230 owns exact optimization acknowledgement/job decoding, poll-URL validation, and caller-owned submission-key lifetime.
- Search, auth, and billing clients retain their current mapping implementations; Task 227's task row names Daily Diet and optimization clients only.
- The repository-wide traceability failure belongs to concurrent Task 224 queue work and was not changed under Task 227's frontend-only scope.

## Preparation decision

Task 227 implementation satisfies the shared runtime-safe mapping criteria with focused 100% mapper line coverage, full frontend validation, and generated-contract drift tests. Per explicit instruction, `docs/implementation/02_TASK_LIST.md` was not edited and Task 227 remains `OPEN`.
