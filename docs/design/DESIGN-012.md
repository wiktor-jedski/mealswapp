## FILE: DESIGN-012.md
**Traceability:** ARCH-012

**Static aspects covered:** USDAClient, OpenFoodFactsClient, ExternalProviderRegistry, ExternalRecordEvidenceStore, DataNormalizer, RateLimitHandler.

### 0. Static Aspect Responsibilities
- `USDAClient`: owns USDA FoodData Central request construction, pagination, and payload parsing.
- `OpenFoodFactsClient`: owns OpenFoodFacts request construction, pagination, and payload parsing.
- `ExternalProviderRegistry`: is the single registration point for supported provider names and provider-owned identifier canonicalization.
- `ExternalRecordEvidenceStore`: owns short-lived opaque references to server-normalized external search records.
- `DataNormalizer`: owns provider-to-internal nutrient, unit, macro, micro, and warning mapping.
- `RateLimitHandler`: owns provider quota tracking, backoff windows, and retry decisions.

### 1. Data Structures & Types
- `interface ExternalSearchQuery { query: string; provider: "usda" | "openfoodfacts" | "all"; page: number; pageSize: number }`
- `interface ExternalFoodRecord { provider: string; externalId: string; name: string; servingSize?: number; servingUnit?: string; nutrients: map[string]float64; imageUrl?: string; rawPayload: []byte }`
- `interface NormalizedFoodCandidate { provider: string; externalId: string; recordToken: string; name: string; physicalState?: PhysicalState; densityGramsPerMilliliter?: number; densitySourceKind?: "imported"; macrosPer100: MacroValues; micros: MicroValues; imageUrl?: string; warnings: string[] }`
- `interface ExternalProviderIdentity { provider: string; externalId: string }`
- `interface ProviderRateLimit { provider: string; remaining: number; resetAt: time.Time; backoffUntil?: time.Time }`
- `interface ExternalDataWarning { provider: string; code: string; message: string }`

### 2. Logic & Algorithms (Step-by-Step)
1. Validate query text, provider, page, and page size from ARCH-009.
2. Check provider rate-limit state before issuing outbound requests.
3. Query USDA FoodData Central and/or OpenFoodFacts using provider-specific clients.
4. Retry transient provider failures up to 3 times with exponential backoff and jitter.
5. Parse provider payloads into `ExternalFoodRecord`, then canonicalize provider identity through the one external-provider registry.
6. Normalize nutrient names to internal protein, carbohydrate, fat, and micronutrient fields.
7. Convert serving-based or package-based values to per 100g or per 100ml when enough unit data is present.
8. Register each normalized candidate as a short-lived server record and return its opaque selection token without writing food or audit state.
9. Include warnings for missing images, incomplete nutrient data, or uncertain unit conversion.
10. Resolve a selected token before import. Missing, malformed, unknown, expired, mismatched, non-canonical, or unregistered evidence fails before food, import, idempotency, or audit persistence.
11. Derive imported density provider and food identity from the resolved record. Manual and estimated density carry neither field.

### 3. State Management & Error Handling
- `provider_available`: requests may be sent.
- `provider_rate_limited`: skip provider and return warning.
- `provider_unavailable`: return empty result set for that provider with warning.
- `partial_success`: one provider failed but another returned candidates.
- `normalization_incomplete`: candidate is returned with warnings and requires admin correction.
- `invalid_external_payload`: drop candidate and log provider diagnostic.
- `invalid_external_record_evidence`: reject import before persistence and require a current search selection.
- `timeout`: provider call fails after configured deadline and retry policy.

### 4. Component Interfaces
- `func SearchExternalFoods(ctx context.Context, query ExternalSearchQuery) ([]NormalizedFoodCandidate, []ExternalDataWarning, error)`
- `func (c *USDAClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error)`
- `func (c *OpenFoodFactsClient) Search(ctx context.Context, query ExternalSearchQuery) ([]ExternalFoodRecord, error)`
- `func NormalizeExternalRecord(record ExternalFoodRecord) (NormalizedFoodCandidate, error)`
- `func (r *ExternalProviderRegistry) Normalize(provider string, externalId string) (ExternalProviderIdentity, error)`
- `func (s *ExternalRecordEvidenceStore) Register(provider string, externalId string) (string, error)`
- `func (s *ExternalRecordEvidenceStore) Resolve(token string) (ExternalProviderIdentity, error)`
- `func ConvertNutrientsToPer100(record ExternalFoodRecord) (MacroValues, MicroValues, []string)`
- `func CheckRateLimit(provider string) ProviderRateLimit`
- `func RecordRateLimit(provider string, headers http.Header) error`
