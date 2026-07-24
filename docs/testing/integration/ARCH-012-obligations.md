# ARCH-012 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for ARCH-012, the External Data Integration service. It verifies provider-client, quota/retry, normalization, ARCH-009 proxy, generated-client, and Administration Panel collaboration without direct provider-driven persistence.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-012 |
| Name | External Data Integration |
| Source Documents | `docs/architecture/ARCH-012.md`, `docs/architecture/01_SOFT_ARCH_DESIGN.md`, `docs/architecture/02_APPENDIX_A.md`, `docs/design/DESIGN-012.md`, `docs/design/DESIGN-009.md` |
| Related Units | USDAClient, OpenFoodFactsClient, RateLimitHandler, DataNormalizer, ExternalSearchProxy, generated external-data client, ExternalImportWorkflow |
| Collaborating Architecture | ARCH-001, ARCH-005, ARCH-009, ARCH-010, ARCH-013 |
| Related Requirements | SW-REQ-055, SW-REQ-090 |

## IT-ARCH-012-001 Provider Fetch, Normalize, and Curate Nominal Flow

### Intent

Verify that selected provider clients, rate handling, normalization, the ARCH-009 proxy, generated contracts, and the Administration Panel exchange bounded deterministic data for curation.

### System Under Test

ARCH-012 External Data Integration centered on USDAClient, OpenFoodFactsClient, DataNormalizer, and RateLimitHandler.

### Real Components

- USDAClient and OpenFoodFactsClient request/response adapters
- RateLimitHandler, DataNormalizer, and ExternalSearchProxy
- production ARCH-009 composition and authenticated external-search HTTP controller
- generated external-data contract and ExternalImportWorkflow

### Allowed Test Doubles

- `httptest.Server` may stand in for USDA and OpenFoodFacts.
- Deterministic provider adapters may supply bounded records to the proxy composition test.
- Browser route interception may stand in for the backend HTTP transport while preserving generated DTOs.

### Trigger / Stimulus

An administrator searches each provider and both providers with a query and page, receives nutrient/unit records, and opens a candidate for curation.

### Expected Integrated Behavior

1. Provider clients encode query, pagination, credentials/caller identity, deadlines, and bounded bodies correctly.
2. Rate-limit state is isolated per provider and response headers update only that provider.
3. DataNormalizer loads one vocabulary snapshot per workflow and maps provider aliases, units, macros, micronutrients, density provenance, and warnings.
4. ExternalSearchProxy returns deterministic bounded candidates without raw payloads and without repository mutation.
5. Generated-client UI preserves provider/page selection and renders editable normalized fields and warnings.

### Required Evidence

- Fake-server request capture, provider projections, quota state, vocabulary call count, normalized candidate fields, absence of raw payload, and browser request/render assertions.

### Requirement Traceability

- SW-REQ-055
- SW-REQ-090

### Verification Status

Implemented by:

- `backend/internal/externaldata/usda_test.go::TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically`
- `backend/internal/externaldata/openfoodfacts_test.go::TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically`
- `backend/internal/externaldata/search_proxy_test.go::TestExternalSearchProxySelectsProviderAndPaginationAndMergesDeterministically`
- `backend/internal/externaldata/normalizer_test.go::TestDataNormalizerLoadsOneVocabularySnapshotPerWorkflow`
- `backend/internal/app/task261_external_import_integration_test.go::TestTask261ProviderHTTPImportPostgresFlow`
- `frontend/tests/external-import-workflow.spec.ts::searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result`

Status: PASS.

## IT-ARCH-012-002 Partial Provider Success and Normalization Warnings

### Intent

Verify that one unavailable, rate-limited, malformed, or incomplete provider does not discard valid candidates from another provider, and that incomplete normalization remains explicit and correctable.

### System Under Test

ARCH-012 provider orchestration, RateLimitHandler, and DataNormalizer as consumed by ARCH-009 ExternalSearchProxy.

### Real Components

- concurrent provider orchestration and RateLimitHandler retry/quota state
- DataNormalizer and ExternalSearchProxy result/warning shaping
- generated external-data contract and ExternalImportWorkflow warning UI

### Allowed Test Doubles

- Provider boundary fixtures may return success, rate limit, unavailable, timeout, malformed, and partial records.
- Deterministic clock, sleep, and jitter may control retry behavior.

### Trigger / Stimulus

One provider returns a valid record while the other is unavailable or rate limited; records also omit data, contain uncertain conversion, or require liquid-density correction.

### Expected Integrated Behavior

1. Valid provider candidates are returned with a bounded warning for the failed provider.
2. Provider retries remain bounded; permanent failure is not retried and one provider's quota/backoff does not block the other.
3. Incomplete candidates carry closed normalization warnings rather than silently inventing values.
4. No 1 ml = 1 g assumption is introduced; suspicious liquid totals remain warnings and micronutrient vocabulary remains enforced.
5. The UI presents safe provider and normalization warnings and allows administrator correction before import.

### Required Evidence

- Provider call counts, retry/backoff state, candidate/warning arrays, normalized density/macro/micro fields, closed warning codes, and browser correction request.

### Requirement Traceability

- SW-REQ-055
- SW-REQ-090

### Verification Status

Implemented by:

- `backend/internal/externaldata/rate_limit_test.go::TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings`
- `backend/internal/externaldata/search_proxy_test.go::TestExternalSearchProxyPartialAndCompleteOutageWarnings`
- `backend/internal/externaldata/normalizer_test.go::TestNormalizeLiquidMassServingWithoutDensityStaysIncomplete`
- `backend/internal/externaldata/normalizer_test.go::TestNormalizeNeverAssumesOneMilliliterEqualsOneGram`
- `backend/internal/externaldata/normalizer_test.go::TestNormalizeEmitsMissingWarningsAndRejectsUnknownCanonicalMicronutrients`
- `backend/internal/app/task261_external_import_integration_test.go::TestTask261ProviderHTTPImportPostgresFlow`
- `frontend/tests/external-import-workflow.spec.ts::searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result`

Status: PASS.

## IT-ARCH-012-003 Complete Outage, Retry Recovery, Cancellation, and Safe UI Degradation

### Intent

Verify graceful degradation and recovery when providers are absent, unavailable, timed out, rate limited, or canceled, without unsafe diagnostics, direct persistence, or stale UI replacement.

### System Under Test

ARCH-012 RateLimitHandler and provider orchestration as exposed through ARCH-009 ExternalSearchProxy and ExternalImportWorkflow.

### Real Components

- RateLimitHandler, provider orchestration, DataNormalizer, and ExternalSearchProxy
- generated client error mapping and ExternalImportWorkflow state machine

### Allowed Test Doubles

- Provider boundary fixtures may fail deterministically.
- Injected clock/sleep/jitter may advance quota reset and retries.
- Browser transport may return 429, 503, 504, malformed, delayed, or empty responses.

### Trigger / Stimulus

Both providers are unavailable; a provider exhausts retries or times out; quota reset later permits recovery; caller cancellation/stale response occurs; the browser receives empty and degraded outcomes.

### Expected Integrated Behavior

1. Complete outage returns an empty candidate list with bounded provider warnings, not fabricated local data.
2. Transient failures retry at most three times; quota/backoff skips calls until reset and a later request can recover.
3. Cancellation and deadlines stop work and preserve their context result.
4. Raw payloads, provider hosts, credentials, and diagnostics do not reach result warnings or UI.
5. The UI exposes loading, empty, rate-limit, timeout, and unavailable states; safe retry is available and stale responses cannot replace newer state.

### Required Evidence

- Empty result/warning contract, provider call counts, reset transition, cancellation/deadline errors, secret exclusion checks, browser state/copy, and stale-response assertion.

### Requirement Traceability

- SW-REQ-055

### Verification Status

Implemented by:

- `backend/internal/externaldata/rate_limit_test.go::TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset`
- `backend/internal/externaldata/rate_limit_test.go::TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings`
- `backend/internal/externaldata/search_proxy_test.go::TestExternalSearchProxyPartialAndCompleteOutageWarnings`
- `backend/internal/externaldata/search_proxy_test.go::TestExternalSearchProxyPropagatesCancellation`
- `frontend/tests/external-import-workflow.spec.ts::shows loading, empty, rate-limit, timeout, and unavailable states without raw diagnostics`
- `frontend/tests/external-import-workflow.spec.ts::ignores a stale external search response after a newer query wins`

Status: PASS.

## Coverage Matrix

| Required path | Obligations |
| --- | --- |
| Nominal | IT-ARCH-012-001 |
| Authorization | Verified at the consuming ARCH-009 boundary by IT-ARCH-009-001 and IT-ARCH-009-002 |
| Isolation | IT-ARCH-012-001, IT-ARCH-012-002 |
| Replay | Provider retry/reset recovery in IT-ARCH-012-003; persistence replay belongs to IT-ARCH-009-003 |
| Conflict | Provider quota isolation in IT-ARCH-012-002; import conflict belongs to IT-ARCH-009-003 |
| Rollback | Read-only/no-persistence behavior in IT-ARCH-012-001; mutation rollback belongs to IT-ARCH-009-003 |
| Provider | IT-ARCH-012-001, IT-ARCH-012-002, IT-ARCH-012-003 |
| Normalization | IT-ARCH-012-001, IT-ARCH-012-002 |
| Deletion | Not owned by ARCH-012; verified by IT-ARCH-009-004, IT-ARCH-009-006, and IT-ARCH-009-007 |
| Invalidation | Quota reset/recovery in IT-ARCH-012-003; classification consumer invalidation belongs to IT-ARCH-009-005 |
| UI | IT-ARCH-012-001, IT-ARCH-012-002, IT-ARCH-012-003 |
| Degraded | IT-ARCH-012-002, IT-ARCH-012-003 |

## SWE.5 Completion Criteria

- Every obligation is implemented by at least one test carrying its obligation ID.
- Tests exercise provider/client, orchestration, normalization, generated-client, and UI collaboration rather than isolated helpers alone.
- Test doubles remain at external API, deterministic time/failure, or browser HTTP boundaries.
- Focused external-data and frontend browser integration suites pass.
- Requirement/design traceability and task-list validation pass.
- No obligation remains uncovered.
