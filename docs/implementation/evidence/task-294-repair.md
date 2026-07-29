# Task 294 Repair Evidence

Date: 2026-07-30

## Repair

The malformed-success response seam now runs in the production browser client
path. The Vite API proxy owns the transport response, buffers the committed
manual-item response, and—when `MEALSWAPP_TASK294_CORRUPT_MANUAL_ITEM_RESPONSE_ONCE=1`—
returns a malformed `{` body while preserving the successful 2xx status. The
injection is limited to one successful `POST /api/v1/admin/items` response and
is consumed before the frontend production client decodes it. The existing
Task 282 dropped-response seam continues to use the same response owner.

This replaces the former backend-only proof that assigned a new body to an
already returned `http.Response`; that assignment did not exercise the
production transport or client decoder.

## Acceptance synchronization

Task 294's existing backend integration proof remains responsible for the
database, audit, idempotency, and Redis exactly-once assertions. The transport
seam is now configured by the same frontend proxy used by the real-stack
Playwright browser, so a real-stack acceptance run can enable the injection
without a route stub or post-response test mutation.

## Validation

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | PASS (previous task baseline) |
| Task 292 dependency status | OPEN; Task 294 is not prepared by this repair |

No task-list status row was changed because the required Task 292 dependency
has not yet reached `PREPARED`.
