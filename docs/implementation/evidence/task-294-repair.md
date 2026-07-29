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

The real-stack Task 283 harness now enables the seam before starting Vite and
runs a production-browser scenario before the existing catalog scenarios. The
scenario waits for the verification-required UI, performs authoritative
picker recovery, replays the captured immutable body and idempotency key, and
proves the picker contains exactly one item with the replayed stable ID. It
also records operation-scoped Redis snapshots; the harness then performs
read-only PostgreSQL proofs for exactly one food row, one `manual_create`
audit entry, and one idempotency record, plus exactly one Redis generation
advance. The resulting proof is committed with the acceptance run.

## Validation

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | PASS (previous task baseline) |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS after harness/test repair |
| `python3 scripts/run-task283-acceptance.py --timeout-seconds 240` | Task 294 browser recovery and exact-effect proof PASS; aggregate reports retain unrelated Phase 08 acceptance findings |
| Managed Task 294 proof | `foodCount=1`, `auditCount=1`, `idempotencyCount=1`, `generationBefore=0`, `generationAfter=1`, `generationDelta=1`, with no assertion failures |
| Task 292 dependency status | OPEN; Task 294 is not prepared by this repair |

No task-list status row was changed because the required Task 292 dependency
has not yet reached `PREPARED`.
