# Global catalog JSON import

`scripts/import-global-catalog.py` is the explicit operator workflow for moderate-volume global catalog changes. It authenticates as an existing administrator and sends one audited, idempotent `POST /api/v1/admin/items` request per item. It does not add a bulk API or connect to PostgreSQL.

Use this workflow only for reviewed documents containing at most 500 items. Larger datasets require a separately designed offline ingestion path with its own staging, reconciliation, rollback, and acceptance controls; do not split a large unreviewed dataset merely to bypass this threshold.

## Document contract

The root object is closed and versioned:

```json
{
  "schema": "mealswapp.global-catalog.v1",
  "items": [
    {
      "idempotencyKey": "catalog-2026-tofu-v1",
      "item": {
        "name": "Tofu",
        "physicalState": "solid",
        "prepTimeMinutes": 5,
        "averageUnitWeightGrams": 100,
        "macrosPer100": {"protein": 18, "carbohydrates": 3, "fat": 9},
        "micros": {"Iron": 2},
        "foodCategoryNames": ["Protein"],
        "culinaryRoleNames": ["Main"],
        "allergenKeys": ["peanut"]
      }
    }
  ]
}
```

Every item needs an operator-assigned key that remains stable across interrupted and repeated runs. Keys must not be derived from array positions. Classification names are the portable default and must resolve to exactly one active classification of the requested kind. `foodCategoryIds` and `culinaryRoleIds` are supported instead of their corresponding name fields for advanced environment-specific operation.

Documents created by `scripts/export-global-catalog.py` also contain item UUIDs, timestamps, deletion state, image alt text, direct source identity, full classification identity, and `curatedSources`. These fields are informational. The importer validates their outer shape, excludes them from `POST /api/v1/admin/items`, and does not recreate `curated_imports` records. Re-import recreates only fields supported by the manual-item route.

Only metric persistence fields are accepted: grams, milliliters, grams per milliliter, and nutrition per 100 grams or milliliters according to physical state. Canonical micronutrient keys are `Sodium`, `Potassium`, `Calcium`, `Iron`, `VitaminC`, `VitaminD`, `Fiber`, and `Sugar`. Allergen keys are resolved against the active server vocabulary.

## Runbook

Start with an authenticated dry run:

```sh
python3 scripts/import-global-catalog.py catalog.json \
  --base-url https://api.example.test \
  --dry-run \
  --report /tmp/catalog-report.json
```

The administrator email and password are read interactively; the password is not echoed and neither value has a command-line argument. Cookies and CSRF state remain only in process memory. After login, the tool obtains a fresh CSRF token, loads both active classification kinds and the allergen vocabulary, validates the complete document, and performs no mutations in dry-run mode.

Remove `--dry-run` only after reviewing the sanitized report. The tool preserves input order, waits at least 2.1 seconds between create attempts, accepts only bounded `Retry-After` delays, and retries ambiguous network or server failures with the exact same key and serialized body. HTTP 400 and 409 fail only that item; HTTP 401 or 403 aborts the run. Any partial failure returns a nonzero status.

Reports contain only item positions, irreversible key fingerprints, bounded outcomes, and HTTP status codes. They intentionally exclude names, email addresses, request/response bodies, request IDs, credentials, cookies, and CSRF tokens. Re-running the unchanged document is the recovery procedure after interruption: completed items replay without duplication and remaining items continue in deterministic order. Changing an item while retaining its old key is rejected by the API.
