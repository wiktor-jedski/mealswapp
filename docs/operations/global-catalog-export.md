# Global catalog export

`scripts/export-global-catalog.py` exports the complete ownerless global food catalog through the administration-only `GET /api/v1/admin/catalog-export` route. This is not Account Export: it contains no users, email addresses, credentials, sessions, consent, subscriptions, saved diets, search history, or private custom items.

## Runbook

Export active items as canonical compact JSON:

```sh
python3 scripts/export-global-catalog.py \
  --base-url https://api.example.test \
  --output ./global-catalog.json
```

Use `--include-deleted` only when deletion-state history is required. Use `--pretty` for reviewable JSON, or `--format csv` for a deterministic human-inspection CSV. CSV is not the import contract; JSON remains the authoritative `mealswapp.global-catalog.v1` representation. The CSV density provenance columns are `densitySourceProvider`, `densitySourceFoodId`, and `densitySourceKind`.

The administrator email and password are read interactively. The password is not echoed, and credentials, cookies, and CSRF state have no command-line or persistent-file form. The summary contains only the output path, item count, and a bounded server request ID.

The operator downloads into a private temporary file in the destination directory. It rejects duplicate keys, non-standard non-finite numbers, malformed fields, invalid ordering, and out-of-contract values while decoding JSON decimals without binary-float conversion. Pretty JSON and CSV therefore preserve the exact numeric tokens supplied by the API.

Before publication, the operator keeps a same-directory durable backup of any existing destination. It flushes and fsyncs file content, atomically replaces the destination, and fsyncs the directory entry. If replacement or directory fsync fails, it restores the prior bytes; when no prior destination existed it removes the uncommitted new file. An authentication, authorization, network, parse, conversion, interrupted-write, temporary-file, or publication failure exits nonzero, closes the response, removes temporary files, and preserves any prior destination.

## Representation and round trip

Items are ordered by UUID and use `global-catalog:<item UUID>` as their stable idempotency key. The JSON has no generated timestamp, so unchanged database state produces byte-identical compact exports. Relationships, canonical allergens, micronutrient keys, and curated-source identities are sorted deterministically.

Each entry contains the supported manual-item metric fields, portable Food Category and Culinary Role names, allergens, micronutrients, density provenance, and image URL. It additionally reports item UUID, image alt text, direct provider/external identity, timestamps, deletion state, relationship UUID/name metadata, and curated-import identity.

The informational metadata is retained for inspection but is deliberately stripped by `scripts/import-global-catalog.py` before calls to the manual-item route. In particular, re-import does not and must not recreate `curated_imports` records. Use the dedicated curated import workflow when truthful curated-source persistence is required.
