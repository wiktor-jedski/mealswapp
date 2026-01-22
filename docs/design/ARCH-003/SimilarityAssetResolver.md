## FILE: SimilarityAssetResolver.md

**Traceability:** ARCH-003, SW-REQ-018

### 1. Data Structures & Types

```go
// AssetConfig holds the configuration for asset resolution
type AssetConfig struct {
    CDNBaseURL     string // Base URL for CDN (e.g., "https://cdn.mealswapp.com")
    AssetVersion   string // Optional version suffix for cache busting (e.g., "v1.2.0")
    FallbackScheme string // Fallback scheme if CDN unavailable ("https")
}

// ResolvedAsset contains the fully resolved asset URL and metadata
type ResolvedAsset struct {
    FullURL      string         `json:"fullUrl"`      // Complete CDN URL
    RelativePath string         `json:"relativePath"` // Original relative path
    Tier         SimilarityTier `json:"tier"`         // Associated tier
}

// assetPathMap maps tiers to their relative asset paths
var assetPathMap = map[SimilarityTier]string{
    TierExcellent: "/assets/indicators/star.png",
    TierGood:      "/assets/indicators/sparkle.png",
    TierFair:      "/assets/indicators/thumbs-up.png",
    TierPoor:      "/assets/indicators/thumbs-down.png",
}
```

**Configuration Constants:**

```go
const (
    // Default CDN configuration
    DefaultCDNBaseURL   = "https://storage.googleapis.com/mealswapp-assets"
    DefaultAssetVersion = ""

    // Asset directory structure
    IndicatorAssetDir = "/assets/indicators"

    // Supported image format
    AssetFormat = ".png"
)
```

**Environment Variable:**

```
MEALSWAPP_CDN_BASE_URL  - Override the default CDN base URL
MEALSWAPP_ASSET_VERSION - Optional version string for cache busting
```

### 2. Logic & Algorithms (Step-by-Step)

**NewAssetResolver(cfg AssetConfig) *AssetResolver**

1. Validate CDNBaseURL is not empty; use DefaultCDNBaseURL if empty
2. Trim trailing slashes from CDNBaseURL
3. Store configuration in resolver instance
4. Return initialized resolver

```
FUNCTION NewAssetResolver(cfg: AssetConfig) -> *AssetResolver:
    IF cfg.CDNBaseURL == "" THEN
        cfg.CDNBaseURL = DefaultCDNBaseURL
    END IF

    cfg.CDNBaseURL = TrimSuffix(cfg.CDNBaseURL, "/")

    IF cfg.FallbackScheme == "" THEN
        cfg.FallbackScheme = "https"
    END IF

    RETURN &AssetResolver{config: cfg}
END FUNCTION
```

**ResolveAssetURL(relativePath string) string**

1. Validate relativePath is not empty
2. Ensure relativePath starts with "/"
3. Construct full URL: CDNBaseURL + relativePath
4. Append version query parameter if AssetVersion is set
5. Return fully qualified URL

```
FUNCTION ResolveAssetURL(relativePath: string) -> string:
    IF relativePath == "" THEN
        RETURN ""
    END IF

    IF NOT HasPrefix(relativePath, "/") THEN
        relativePath = "/" + relativePath
    END IF

    fullURL = config.CDNBaseURL + relativePath

    IF config.AssetVersion != "" THEN
        fullURL = fullURL + "?v=" + config.AssetVersion
    END IF

    RETURN fullURL
END FUNCTION
```

**ResolveIndicatorAsset(tier SimilarityTier) ResolvedAsset**

1. Look up relative path for the given tier in assetPathMap
2. If tier not found, default to TierPoor path
3. Call ResolveAssetURL with the relative path
4. Return ResolvedAsset with full URL, relative path, and tier

```
FUNCTION ResolveIndicatorAsset(tier: SimilarityTier) -> ResolvedAsset:
    relativePath = assetPathMap[tier]

    IF relativePath == "" THEN
        relativePath = assetPathMap[TierPoor]
        tier = TierPoor
    END IF

    fullURL = ResolveAssetURL(relativePath)

    RETURN ResolvedAsset{
        FullURL:      fullURL,
        RelativePath: relativePath,
        Tier:         tier
    }
END FUNCTION
```

**ResolveAllIndicatorAssets() map[SimilarityTier]ResolvedAsset**

1. Initialize empty result map
2. Iterate over all defined tiers (Excellent, Good, Fair, Poor)
3. For each tier, call ResolveIndicatorAsset
4. Store result in map keyed by tier
5. Return complete map

```
FUNCTION ResolveAllIndicatorAssets() -> map[SimilarityTier]ResolvedAsset:
    result = make(map[SimilarityTier]ResolvedAsset)

    FOR tier IN [TierExcellent, TierGood, TierFair, TierPoor]:
        result[tier] = ResolveIndicatorAsset(tier)
    END FOR

    RETURN result
END FUNCTION
```

**GetAssetManifest() []ResolvedAsset**

Returns a slice of all resolved assets for preloading or validation purposes.

```
FUNCTION GetAssetManifest() -> []ResolvedAsset:
    allAssets = ResolveAllIndicatorAssets()
    manifest = make([]ResolvedAsset, 0, len(allAssets))

    FOR _, asset IN allAssets:
        manifest = append(manifest, asset)
    END FOR

    RETURN manifest
END FUNCTION
```

### 3. State Management & Error Handling

**Error States:**

| Error Condition | Handling Strategy |
| :--- | :--- |
| Empty CDNBaseURL in config | Use DefaultCDNBaseURL constant |
| Empty relativePath argument | Return empty string (no error) |
| Unknown SimilarityTier | Default to TierPoor asset path |
| Invalid URL characters in path | No sanitization; caller responsible for valid paths |
| CDN unreachable at runtime | Not handled here; CDN availability is infrastructure concern |

**State Transitions:**

This component is stateless after initialization. The AssetConfig is immutable once the resolver is created. All resolution methods are pure functions that produce deterministic output based on input and configuration.

**Initialization States:**

```
[Uninitialized] --NewAssetResolver(cfg)--> [Ready]
     |
     v
[Ready] --Any method call--> [Ready] (no state change)
```

**Configuration Validation:**

| Config Field | Validation Rule | Default Value |
| :--- | :--- | :--- |
| CDNBaseURL | Non-empty string | DefaultCDNBaseURL |
| CDNBaseURL | No trailing slash | Trimmed automatically |
| AssetVersion | Any string (including empty) | "" (no versioning) |
| FallbackScheme | Non-empty | "https" |

**Cache Busting Strategy:**

When AssetVersion is set, URLs are appended with `?v={version}`:
- `https://cdn.example.com/assets/indicators/star.png?v=1.2.0`

This ensures browser caches are invalidated when assets are updated.

### 4. Component Interfaces

```go
// SimilarityAssetResolver resolves relative asset paths to full CDN URLs
type SimilarityAssetResolver interface {
    // ResolveAssetURL converts a relative path to a fully qualified CDN URL
    // Returns empty string if relativePath is empty
    ResolveAssetURL(relativePath string) string

    // ResolveIndicatorAsset resolves the asset for a specific similarity tier
    // Returns TierPoor asset if tier is unrecognized
    ResolveIndicatorAsset(tier SimilarityTier) ResolvedAsset

    // ResolveAllIndicatorAssets returns resolved assets for all tiers
    ResolveAllIndicatorAssets() map[SimilarityTier]ResolvedAsset

    // GetAssetManifest returns all indicator assets as a slice for preloading
    GetAssetManifest() []ResolvedAsset

    // GetCDNBaseURL returns the configured CDN base URL
    GetCDNBaseURL() string
}

// NewSimilarityAssetResolver creates a resolver with the given configuration
func NewSimilarityAssetResolver(cfg AssetConfig) SimilarityAssetResolver

// NewSimilarityAssetResolverFromEnv creates a resolver using environment variables
// Reads MEALSWAPP_CDN_BASE_URL and MEALSWAPP_ASSET_VERSION from environment
func NewSimilarityAssetResolverFromEnv() SimilarityAssetResolver
```

**Usage Example:**

```go
// Create resolver with explicit configuration
cfg := AssetConfig{
    CDNBaseURL:   "https://cdn.mealswapp.com",
    AssetVersion: "1.2.0",
}
resolver := NewSimilarityAssetResolver(cfg)

// Resolve a specific tier's asset
asset := resolver.ResolveIndicatorAsset(TierExcellent)
// Returns: ResolvedAsset{
//     FullURL:      "https://cdn.mealswapp.com/assets/indicators/star.png?v=1.2.0",
//     RelativePath: "/assets/indicators/star.png",
//     Tier:         TierExcellent
// }

// Resolve an arbitrary asset path
url := resolver.ResolveAssetURL("/assets/custom/badge.png")
// Returns: "https://cdn.mealswapp.com/assets/custom/badge.png?v=1.2.0"

// Get all indicator assets for preloading
manifest := resolver.GetAssetManifest()
// Returns slice of 4 ResolvedAsset entries

// Create resolver from environment
envResolver := NewSimilarityAssetResolverFromEnv()
```

**Integration with SimilarityIndicatorMapper:**

The AssetResolver enhances the SimilarityIndicatorMapper output by resolving relative paths to full CDN URLs:

```go
// In similarity engine orchestration
mapper := NewSimilarityIndicatorMapper()
resolver := NewSimilarityAssetResolverFromEnv()

indicator := mapper.MapScoreToIndicator(0.78)
// indicator.ImageURL is "/assets/indicators/sparkle.png"

fullURL := resolver.ResolveAssetURL(indicator.ImageURL)
// fullURL is "https://cdn.mealswapp.com/assets/indicators/sparkle.png?v=1.2.0"
```

**HTTP Handler Integration (Fiber):**

```go
// Route to serve asset manifest for client preloading
app.Get("/api/v1/assets/indicators", func(c *fiber.Ctx) error {
    resolver := NewSimilarityAssetResolverFromEnv()
    manifest := resolver.GetAssetManifest()
    return c.JSON(manifest)
})
```
