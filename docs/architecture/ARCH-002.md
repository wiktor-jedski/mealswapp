# [ARCH-002] - Search Module

**Description:** Backend service responsible for processing search queries, implementing autocomplete ranking, and coordinating with the Similarity Engine for result retrieval and filtering.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | SearchController, AutocompleteRanker, QueryParser, PaginationHandler, FilterProcessor, CulinaryRoleWeighter |
| **Dependencies** | ARCH-003 (Similarity Engine), ARCH-005 (Data Repository), ARCH-011 (Caching Layer) |
| **Traceability** | SW-REQ-004, SW-REQ-010, SW-REQ-017, SW-REQ-019, SW-REQ-024, SW-REQ-026, SW-REQ-029, SW-REQ-031 |

**Dynamic Behavior:**

- **Query Processing:** Receives search terms, applies Search filters and Exclusion Rules, and routes to Catalog Search, Substitution Search, or Daily Daily Diet Alternative Search.
- **Autocomplete Ranking:** Implements three-tier priority: (1) Exact match, (2) Levenshtein distance, (3) String length. Executes in < 100ms.
- **Substitution Trigger:** Detects empty search bar with one or more Substitution Inputs to automatically initiate Substitution Search.
- **Pagination:** Returns max 10 results per page, sorted by Nutritional Similarity descending for Substitution Search.
- **Culinary Role Weighting (SW-REQ-031):** During single-input Substitution Searches, applies a relevance boost multiplier to Food Objects sharing Culinary Roles with the Substitution Input. Sorting combines Nutritional Similarity with Culinary Role match weight (e.g., `finalScore = similarityScore * (1 + 0.2 * culinaryRoleMatchCount)`) to prioritize contextually appropriate Substitutes. Multiple-input Substitution Searches combine inputs into one Macro Profile and skip per-input Culinary Role weighting.

**Interface Definition:**

- `Input`: SearchRequest { query: string, mode: SearchMode, filters: SearchFilter[], page: number, substitutionInputs?: SubstitutionInput[] }
- `Output`: SearchResponse { items: FoodObject[], totalCount: number, page: number, similarityScores: number[] }

**Alternative Analysis (BP6):**

- *Chosen Approach:* Dedicated Search Module with in-memory ranking algorithms
- *Alternative Considered:* Elasticsearch/Algolia for full-text search
- *Trade-off:* Custom module provides precise control over ranking algorithm (SW-REQ-004) and cosine similarity integration. External search services would require synchronization overhead and may not support custom similarity scoring. For the current scale (1000 users), custom solution is more cost-effective and controllable.

**Reference Documentation:** 
- 02_APPENDIX_A.md
