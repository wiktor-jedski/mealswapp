# System Design

## Tech Stack

### Frontend
Web framework: Svelte
Build tool: Bun
State management: Svelte stores + TanStack Query
CSS: Tailwind
Testing: Bun test runner + @testing-library/svelte + Playwright
Caching: Service Worker + localStorage

### Backend
Language: Go
Framework: Fiber
Query builder: raw SQL
Internal API: Direct function calls
LP Solver: native COIN-OR CLP `1.17.11` executable invoked by a pure-Go `exec.CommandContext` wrapper in the dedicated optimizer container; the `linux/amd64` Ubuntu 24.04 image packages the checksum-pinned official COIN-OR Ubuntu 24 release artifact with the Go worker, with no CGO binding and no solver execution in the Fiber API process
Cosine Similarity: Custom implementation
API Documentation: OpenAPI
Testing: testing package (built-in)
Database: PostgreSQL
Cache/Session/Job Queue: Redis
Password hashing: Argon2 (golang.org/x/crypto/argon2)
Session management: Fiber session middleware
Job queue: Redis Streams through `github.com/redis/go-redis/v9` (`XADD`, `XREADGROUP`, `XAUTOCLAIM`, `XACK`)
Logging: Fiber logger middleware + GCP Cloud Monitoring

### Data Layer
Database: PostgreSQL
Cache: Redis
Redis client: github.com/redis/go-redis/v9
PostgreSQL driver: lib/pq or pgx

### External Services
Email: Resend
Payments: Stripe
Food data: USDA FoodData Central, OpenFoodFacts
Hosting: GCP (Cloud Run, Cloud SQL, Memorystore, Cloud Storage, Cloud CDN)
CI/CD: GitHub Actions

### Security
OAuth: github.com/markbates/goth
Rate limiting: Fiber built-in limiter
Encryption: AES-256 (crypto/aes)
CSRF: Fiber csrf middleware

### Operational
Image hosting: GCP Cloud Storage + Cloud CDN
Monitoring: GCP Cloud Monitoring
Container orchestration: Not needed (managed services)
Secrets: GCP Secret Manager
