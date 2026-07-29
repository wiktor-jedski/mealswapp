### Dev verification

#### SW-REQ-054 — Administrative access

- Visit /admin anonymously.
- Visit /admin as an ordinary user.
- Call representative admin endpoints directly as both.
- Sign in as an admin and repeat.

Accept when:

- Anonymous API calls return 401.
- Authenticated non-admin calls return 403.
- Admin calls succeed.
- Direct navigation does not reveal restricted data.
- Spoofing role/user information in the request does not help.

#### SW-REQ-055 — External data curation

- Search USDA.
- Search OpenFoodFacts.
- Search both simultaneously.
- Select a result.
- Edit its name, nutrition, physical state and classifications.
- Confirm import.
- Search for the new item in the local catalog.

Accept when:

- External search alone does not modify the local database.
- Provider warnings appear without exposing raw payloads.
- Edited values are used.
- Exactly one global item is imported.
- The imported item becomes locally searchable.
- Retrying a lost/ambiguous response does not duplicate it.
- One provider failing still allows results from the other.

#### SW-REQ-056 — Manual global items

- Create a solid item.
- Create a liquid item with density.
- Update their names and nutrition.
- Verify updated search results.
- Delete them.
- Verify they disappear from search.

Accept when:

- Created items are global and have no private owner.
- Liquid-density rules are enforced.
- Invalid nutrition/classification/image data is rejected.
- Create retry does not duplicate the item.
- Updates become visible in Catalog and Substitution Search.
- Deletion removes the item from active search.
- Audit failure rolls back the entire mutation.

Our review has open questions about what “delete” should mean and how density provenance should be established.

#### SW-REQ-057 — Classifications

- Create a Food Category.
- Create a Culinary Role.
- Rename/reparent them.
- Attach one to an item.
- Attempt to create a duplicate or hierarchy cycle.
- Attempt to delete an in-use classification.
- Remove its use and delete it.
- Open Substitution Search in another browser/application instance.

Accept when:

- Duplicate and cycle attempts fail without mutation.
- In-use deletion returns a conflict.
- Successful rename appears in filter options.
- Successful deletion removes the option.
- Changes appear across application instances without a restart.
- Failed mutations do not invalidate the cache.

## Existing requirements completed or extended by Phase 08

### SW-REQ-019 — Classification-Based Filtering

> WHERE user-defined classification preferences exist, the software shall filter all search results to include only
> 'whitelisted' classifications and exclude all 'blacklisted' classifications.

Dev check:

- Select an include classification and confirm only matching results appear.
- Select an exclusion and confirm matching results disappear.
- Verify requests use classification IDs, not display labels.
- Rename a classification and confirm filtering still works.

### SW-REQ-032 — Unit Conversion Logic (Imperial)

> WHILE the application is set to 'US Imperial' mode, the software shall convert grams to ounces (1g ≈ 0.035oz) and milliliters
> to fluid ounces (1ml ≈ 0.033fl oz) for all displays.

Dev check:

- Switch the UI between metric and imperial.
- Verify solid and liquid displays.
- Submit an imperial quantity and verify backend calculations match its metric equivalent.

This requirement is currently affected by the metric-domain open point we added.

### SW-REQ-033 — Standardized Storage Units

> The software shall store all item macronutrient values normalized to 100 grams (for solids) or 100 milliliters (for liquids).

Dev check:

- Import a solid provider item.
- Import a liquid provider item.
- Compare displayed/stored macro bases with 100g and 100ml.
- Verify changing display units does not change canonical nutritional values.
- Confirm no liquid path silently assumes 1 ml = 1 g.

### SW-REQ-043 — Private Item Visibility

> The software shall ensure that custom items created by a user are only accessible and visible to that specific authenticated
> user.

Dev check:

- Create private items for users A and B.
- Attempt to read, update and delete A’s item as B.
- Attempt to guess A’s item ID.
- Export both accounts.

Accept when B cannot discover whether A’s item exists and each export includes only its owner’s data.

### SW-REQ-072 — Data Portability

> The software shall allow the user to export all personal data, including saved ingredients, diets, and search history, in
> machine-readable JSON and CSV formats.

Dev check:

- Create a private custom item.
- Create saved diet/history data.
- Export JSON and CSV.
- Confirm all owned data is present.
- Confirm global and other-user data is absent.
- Delete a private item and verify the chosen retention semantics are reflected correctly.

### SW-REQ-073 — Right to Erasure

> WHEN a user confirms account deletion, the software shall permanently remove all associated Personally Identifiable
> Information (PII), private custom items, and historical logs from the active production database.

Dev check:

- Create a disposable test account with private items and history.
- Request account deletion.
- Verify new writes are blocked while deletion is pending.
- Let the worker complete deletion.
- Attempt login, profile access, export and custom-item access.
- Confirm another user and global items remain unaffected.

This requires the backend worker and database; frontend-only verification is insufficient.

### SW-REQ-084 — Application Logging

> The software shall log all authentication events, API requests, errors, and administrative actions with timestamps and user
> identifiers to a centralized logging system.

Dev check:

- Perform a successful and failed admin mutation.
- Perform external search/import.
- Inspect deployed logs.

Accept when:

- Events have timestamps and correlation/request IDs.
- Successful and failed admin actions are distinguishable.
- Logs do not contain names, emails, search text, item IDs, idempotency keys, secrets or provider payloads.
- The deployment actually centralizes logs; console output alone does not fully satisfy the wording.

### SW-REQ-090 — Micronutrient nomenclature

> The software shall validate all micronutrient keys against a predefined standardized vocabulary (e.g., allowing "Sodium" but
> rejecting "Na") before they are stored or assigned to an item.

Dev check:

- Create/import an item using Sodium.
- Try the same using Na.
- Try an unknown micronutrient.
- Disable a vocabulary entry and retry.

Accept when only active canonical keys are persisted.

## Recommended review strategy

Do not continue reading every backend file equally. Review these flows deeply:

1. SW-REQ-043: cross-user isolation.
2. SW-REQ-054: admin authorization.
3. SW-REQ-055: import idempotency, provider handling and audit rollback.
4. SW-REQ-057: classification invalidation across instances.
5. SW-REQ-072/073: export and permanent erasure.

For repetitive validation and repository code, use the tests and dev deployment as evidence. If a validator merely repeats an
OpenAPI bound and has representative tests, scanning it is enough.
