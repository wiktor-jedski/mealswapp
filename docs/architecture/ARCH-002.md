# [ARCH-002] - Search Module

**Description:** Backend service responsible for processing search queries, implementing autocomplete ranking, and coordinating with the Similarity Engine for result retrieval and filtering.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | SearchController, AutocompleteRanker, QueryParser, PaginationHandler, FilterProcessor, FunctionalityTagWeighter |
| **Dependencies** | ARCH-003 (Similarity Engine), ARCH-005 (Data Repository), ARCH-011 (Caching Layer) |
| **Traceability** | SW-REQ-004, SW-REQ-010, SW-REQ-017, SW-REQ-019, SW-REQ-024, SW-REQ-026, SW-REQ-029, SW-REQ-031 |

**Dynamic Behavior:**

- **Query Processing:** Receives search terms, applies tag whitelist/blacklist filters, and routes to appropriate search strategy (text-based or similarity-based).
- **Autocomplete Ranking:** Implements three-tier priority: (1) Exact match, (2) Levenshtein distance, (3) String length. Executes in < 100ms.
- **Implicit Trigger:** Detects empty search bar with 2+ ingredients to automatically initiate similarity search.
- **Pagination:** Returns max 10 results per page, sorted by cosine similarity descending.
- **Functionality Tag Weighting (SW-REQ-031):** During replacement searches, applies a relevance boost multiplier to items sharing the same Functionality Tags as the source item. Sorting combines cosine similarity score with tag match weight (e.g., `finalScore = similarityScore * (1 + 0.2 * tagMatchCount)`) to prioritize contextually appropriate replacements.

**Interface Definition:**

- `Input`: SearchRequest { query: string, mode: SearchMode, filters: TagFilter[], page: number, ingredients?: string[] }
- `Output`: SearchResponse { items: FoodItem[], totalCount: number, page: number, similarityScores: number[] }

**Alternative Analysis (BP6):**

- *Chosen Approach:* Dedicated Search Module with in-memory ranking algorithms
- *Alternative Considered:* Elasticsearch/Algolia for full-text search
- *Trade-off:* Custom module provides precise control over ranking algorithm (SW-REQ-004) and cosine similarity integration. External search services would require synchronization overhead and may not support custom similarity scoring. For the current scale (1000 users), custom solution is more cost-effective and controllable.

**Reference Documentation:** 
- 02_APPENDIX_A.md
