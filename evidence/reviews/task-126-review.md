# Task 126 Re-review Evidence

Recommended status: PASSED

## Scope

Reviewed exactly task 126, "Phase 04 Authenticated Search History Persistence".

Verified task-list state:

- Task 126 status is `PREPARED`.
- Dependencies 96, 122, 123, and 124 are all `PASSED`.

Previous rejection source of truth:

- Missing verification that inserted 101+ encrypted history rows and asserted latest-100 cap/list behavior.

## Files Inspected

- `backend/internal/repository/postgres_repository_test.go`
- `backend/internal/repository/sql/encrypted_search_history_add.sql`
- `backend/internal/repository/sql/encrypted_search_history_list.sql`
- `backend/internal/repository/encrypted_identity_repository.go`
- `backend/internal/userdata/service.go`
- `backend/internal/httpapi/search_controller.go`
- `backend/internal/httpapi/search_controller_test.go`

## Checklist

- [x] Authenticated successful searches append history after a successful search response.
- [x] Anonymous successful searches do not persist history.
- [x] Rejected searches do not persist history.
- [x] Failed searches do not persist history.
- [x] Query text is encrypted before repository persistence.
- [x] Duplicate searches are retained.
- [x] Repository cap is verified with more than 100 encrypted history inserts.
- [x] Listing returns latest 100 rows, ordered newest first.
- [x] Other users' history rows are not included in the capped/listed rows.
- [x] Clear-history behavior remains covered by existing authenticated user-data tests.
- [x] HTTP test verifies history uses server-derived user ID, not request-supplied/spoofed user ID.

## Implementation Notes

The repair adds `TestPostgresEncryptedIdentityRepositoryEncryptedHistoryLatest100`, which inserts 102 encrypted history rows for one user, forces deterministic `created_at` values, adds one final current row, and adds another row for a second user. It asserts:

- persisted history for the target user is capped at 100 rows;
- the oldest row is pruned;
- `ListEncryptedHistory(ctx, userID, 0)` returns exactly 100 rows;
- newest row is first and the expected latest range is retained;
- duplicate encrypted ciphertext rows are retained twice;
- listed entries are scoped to the requested user;
- another user's history remains independently listable.

The SQL repair prunes old rows in `encrypted_search_history_add.sql` and updates `encrypted_search_history_list.sql` to order ties by `id DESC`.

## Verification Run

Commands run from `backend/`:

```text
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run TestPostgresEncryptedIdentityRepositoryEncryptedHistoryLatest100 -count=1 -v
```

Result:

```text
=== RUN   TestPostgresEncryptedIdentityRepositoryEncryptedHistoryLatest100
--- PASS: TestPostgresEncryptedIdentityRepositoryEncryptedHistoryLatest100 (0.84s)
PASS
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/repository	0.842s
```

```text
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/userdata ./internal/app ./internal/search -count=1
```

Result:

```text
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/httpapi	1.132s
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/userdata	0.006s
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/app	0.006s
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/search	1.757s
```

## Finding

No blocking findings. The previous rejection condition is repaired: there is now a practical repository test inserting more than 100 encrypted history rows and verifying the latest-100 cap/list behavior.

Recommended status: PASSED
