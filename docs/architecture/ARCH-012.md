# [ARCH-012] - External Data Integration

**Description:** Integration layer for fetching and normalizing food data from external APIs (USDA FoodData Central, OpenFoodFacts).

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | USDAClient, OpenFoodFactsClient, DataNormalizer, RateLimitHandler |
| **Dependencies** | External APIs (USDA, OpenFoodFacts), ARCH-005 (Data Repository) |
| **Traceability** | SW-REQ-055 |

**Dynamic Behavior:**

- **API Fetching:** Queries external APIs based on search terms, handles pagination and rate limits.
- **Data Normalization:** Converts external formats to internal schema, maps to standard units (per 100g/ml).
- **Error Handling:** Graceful degradation when external APIs are unavailable (returns empty results with warning).

**Interface Definition:**

- `Input`: Search queries, item identifiers
- `Output`: Normalized FoodItem candidates for admin curation

**Alternative Analysis (BP6):**

- *Chosen Approach:* On-demand fetching with admin curation workflow
- *Alternative Considered:* Bulk data import with scheduled synchronization
- *Trade-off:* On-demand fetching with curation (SW-REQ-055) ensures data quality and proper functionality tagging. Bulk import would populate database faster but with uncurated, potentially inconsistent data. Quality over quantity is critical for accurate similarity matching.

**Reference Documentation:** 
- 02_APPENDIX_A.md
