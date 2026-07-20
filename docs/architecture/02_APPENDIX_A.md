# Appendix A

## 6. NFR Enforcement Strategies

This section explains **how** the architecture enforces each key non-functional requirement, mapping specific architectural mechanisms to measurable NFR targets.

---

### 6.1 Response Time: SW-REQ-080 (< 2s P95)

The 2-second P95 response time target is enforced through multiple complementary mechanisms:

| Mechanism | Component | Impact |
|:----------|:----------|:-------|
| **Redis Query Caching** | ARCH-011 | Hot queries served in <10ms from cache |
| **PostgreSQL Indexing** | ARCH-005 | B-tree indexes on `name`, `category_tags`, composite indexes on frequently filtered columns |
| **Pagination Limits** | ARCH-002 | Max 10 results per page prevents unbounded query times |
| **Async LP Processing** | ARCH-004 | CPU-intensive optimization runs in background workers, API returns immediately with job ID |
| **Input Debouncing** | ARCH-001 | 150ms debounce on search input reduces redundant API calls by ~70% |
| **Connection Pooling** | ARCH-005 | Pre-warmed DB connections eliminate connection overhead |

**Enforcement Path:**
```
User Input → 150ms debounce → Cache check (ARCH-011)
                                    │
                    ┌───────────────┴───────────────┐
                    │ Cache HIT                     │ Cache MISS
                    ▼                               ▼
              Return cached              PostgreSQL query (indexed)
              result (<10ms)                       │
                                                   ▼
                                           Cache result
                                                   │
                                                   ▼
                                           Return (<500ms typical)
```

**Monitoring:** ARCH-014 tracks P95 latency per endpoint. Alerts trigger if P95 exceeds 1.5s (warning) or 2s (critical).

---

### 6.2 Availability: SW-REQ-081 (99.9% Uptime)

99.9% availability (8.76 hours downtime/year max) requires eliminating single points of failure:

| Mechanism | Component | Impact |
|:----------|:----------|:-------|
| **Stateless API Design** | ARCH-010 | Any API node can handle any request; nodes are interchangeable |
| **Redis Session Store** | ARCH-011 | Sessions stored externally via github.com/redis/go-redis/v9; no sticky sessions required |
| **Connection Pooling** | ARCH-005 | PgBouncer or equivalent prevents connection exhaustion |
| **Health Check Endpoints** | ARCH-010 | `/health` and `/ready` endpoints for load balancer probing |
| **Graceful Shutdown** | ARCH-010 | SIGTERM triggers drain period; in-flight requests complete before termination |
| **Database Replication** | ARCH-005 | Read replicas for query distribution; automatic failover for primary |

**Stateless Architecture Diagram:**
```
                    ┌─────────────────────────────────────┐
                    │           Load Balancer             │
                    │  (health checks every 10s)          │
                    └───────────────┬─────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
        ┌─────▼─────┐         ┌─────▼─────┐         ┌─────▼─────┐
        │ API Node  │         │ API Node  │         │ API Node  │
        │ (stateless)│        │ (stateless)│        │ (stateless)│
        └─────┬─────┘         └─────┬─────┘         └─────┬─────┘
              │                     │                     │
              └─────────────────────┼─────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
        ┌─────▼─────┐         ┌─────▼─────┐         ┌─────▼─────┐
        │ PostgreSQL│         │   Redis   │         │ LP Worker │
        │  Primary  │         │  Cluster  │         │   Pool    │
        │ + Replica │         │           │         │           │
        └───────────┘         └───────────┘         └───────────┘
```

**Monitoring:** ARCH-014 performs synthetic health checks every 30s. Uptime calculated monthly; alerts if projected availability drops below 99.95%.

---

### 6.3 Concurrent Users: SW-REQ-082 (1000 Users)

Supporting 1000 concurrent users requires preventing resource exhaustion:

| Mechanism | Component | Impact |
|:----------|:----------|:-------|
| **Async Job Queue** | ARCH-004 | LP computations isolated from API event loop; prevents CPU starvation |
| **Connection Pool Limits** | ARCH-005 | Max 100 DB connections shared across requests via pooling |
| **Rate Limiting** | ARCH-010 | Per-IP and per-user limits prevent individual users from monopolizing resources |
| **Horizontal Scaling** | ARCH-010 | Additional API nodes can be added without code changes |
| **Worker Scaling** | ARCH-004 | LP worker pool scales independently based on queue depth |

**Capacity Model:**
```
1000 concurrent users
× 0.1 requests/second/user (avg)
= 100 requests/second sustained load

Go Fiber API Node capacity: ~500 req/s per node
Required nodes: 1 (with 2x headroom = 2 nodes minimum)

LP jobs: ~5% of requests = 5 jobs/second
Worker capacity: ~10 jobs/second per worker (Go wrapper with native COIN-OR CLP child process)
Required workers: 1 (with headroom = 2 workers)
```

**Monitoring:** ARCH-014 tracks concurrent connections, queue depth, and worker utilization. Auto-scaling triggers at 70% capacity.

---

## 7. Operational Behavior

This section defines runtime behavior, failure modes, and degradation strategies for production operation.

---

### 7.1 Critical Path

The **critical path** represents the minimum functionality required for core user value. These components must remain available even when other services degrade:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ ARCH-006    │────>│ ARCH-002    │────>│ ARCH-003    │────>│ ARCH-001    │
│ Auth        │     │ Search      │     │ Similarity  │     │ Web App     │
│             │     │             │     │ (optional)  │     │ (Results)   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
     │                    │                   │                   │
     │    Required        │    Required       │    Degradable     │
     │                    │                   │                   │
```

**Critical Components (must not fail):**
1. **ARCH-006 (Authentication):** Users cannot access any features without valid session
2. **ARCH-002 (Search):** Core value proposition; basic text search must always work
3. **ARCH-005 (Data Repository):** All queries depend on database connectivity

---

### 7.2 Graceful Degradation Order

When system resources are constrained, features degrade in this order (least critical first):

| Priority | Feature | Component | Degraded Behavior |
|:---------|:--------|:----------|:------------------|
| 1 (first to degrade) | Recommendations | ARCH-002 | Disable "similar items" suggestions |
| 2 | History Sync | ARCH-008 | Use local cache only; sync when recovered |
| 3 | LP Optimization | ARCH-004 | Return "optimization unavailable"; user retries later |
| 4 | Similarity Scores | ARCH-003 | Return search results without similarity indicators |
| 5 (last to degrade) | Basic Search | ARCH-002 | Text-only search; no filtering |
| Never | Authentication | ARCH-006 | If auth fails, return 503; do not allow unauthenticated access |

**Degradation Triggers:**
- Redis unavailable → Degrade priorities 1-2
- LP workers saturated → Degrade priority 3
- ARCH-003 response time >5s → Degrade priority 4
- PostgreSQL read replica lag >10s → Degrade priority 5

---

### 7.3 Failure Modes per Component

| Component | Failure Scenario | Detection | Response | User Impact |
|:----------|:-----------------|:----------|:---------|:------------|
| **ARCH-003** (Similarity) | High latency (>5s) | Timeout monitoring | Return results without similarity scores | Results display without color indicators; "Similarity unavailable" banner |
| **ARCH-004** (LP Optimizer) | Job timeout (>30s) | Worker `exec.CommandContext` deadline and child-process exit | Terminate CLP, remove the private solver directory, mark job failed, and return partial results if available | "Optimization taking longer than expected. Please try again." |
| **ARCH-012** (External APIs) | USDA/OpenFoodFacts/Resend down | HTTP 5xx or timeout | Log warning; return empty results to admin | Admin sees "External data source unavailable"; no user impact |
| **Redis** | Connection refused | github.com/redis/go-redis/v9 connection error | Fall back to direct PostgreSQL queries (lib/pq or pgx) | Slower responses (~500ms vs ~10ms); full functionality maintained |
| **PostgreSQL Primary** | Connection lost | Connection pool error (lib/pq or pgx) | Automatic failover to replica (if configured) | Brief interruption (<30s); read-only mode during failover |

**Redis Fallback Flow:**
```
Request → Check Redis (github.com/redis/go-redis/v9)
               │
     ┌─────────┴─────────┐
     │ Redis OK          │ Redis DOWN
     ▼                   ▼
Return cached       Query PostgreSQL directly (lib/pq or pgx)
                    Log degraded mode
                    Set circuit breaker (retry Redis in 30s)
```

---

### 7.4 Retry vs Fail-Fast Policies

Different failure types require different handling strategies:

| Operation Type | Policy | Rationale |
|:---------------|:-------|:----------|
| **External API calls** (USDA, OpenFoodFacts, Resend) | Retry 3x with exponential backoff (1s, 2s, 4s), then fail | External services have transient failures; retries often succeed |
| **Database queries** | Fail fast | DB issues indicate serious problems; retrying wastes resources and delays error reporting |
| **Stripe webhooks** | Handled by Stripe | Stripe retries automatically for up to 3 days; handler must be idempotent |
| **LP jobs** | Timeout at 30s, no retry | LP is deterministic through the pinned native CLP executable; if it fails once, it will fail again. User can manually retry. |
| **Redis operations** | Fail fast, fallback to DB | Redis failures via github.com/redis/go-redis/v9 should degrade gracefully, not block requests |
| **OAuth provider calls** | Retry 2x, then fail with user message | OAuth failures (github.com/markbates/goth) are often transient; limited retries appropriate |

**Exponential Backoff Implementation:**
```
attempt = 0
max_attempts = 3
base_delay = 1000ms

while attempt < max_attempts:
    try:
        result = call_external_api()
        return result
    except TransientError:
        attempt += 1
        if attempt < max_attempts:
            delay = base_delay * (2 ^ attempt) + random_jitter(100ms)
            sleep(delay)
        else:
            raise ExternalServiceUnavailable
```

---

## 8. Deployment Architecture

This section defines the deployment topology, environment configurations, and CI/CD pipeline for the Mealswapp application.

---

### 8.1 Environments

| Environment | Purpose | Configuration |
|:------------|:--------|:--------------|
| **Development** | Local developer workstations | Docker Compose with all services; hot reload enabled; debug logging |
| **Staging** | QA and integration testing | Single-node deployment; production-like config; test data |
| **Production** | Live user traffic | Horizontally scaled; monitoring enabled; automated backups |

**Environment Parity:**
- All environments use identical Docker images (different tags)
- Environment-specific configuration via environment variables only
- Database schema migrations applied identically across environments

---

### 8.2 Production Topology

```
                         ┌─────────────────────┐
                         │     Cloudflare      │
                         │   CDN / WAF / DDoS  │
                         │                     │
                         │ - Static asset cache│
                         │ - SSL termination   │
                         │ - Geographic routing│
                         └──────────┬──────────┘
                                    │
                                    │ HTTPS
                                    ▼
                         ┌─────────────────────┐
                         │   Load Balancer     │
                         │                     │
                         │ - Health checks     │
                         │ - Session affinity  │
                         │   (not required)    │
                         └──────────┬──────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
        ┌─────▼─────┐         ┌─────▼─────┐         ┌─────▼─────┐
        │ API Node  │         │ API Node  │         │ API Node  │
        │    1      │         │    2      │         │    N      │
        │           │         │           │         │           │
        │ ARCH-001  │         │ ARCH-001  │         │ ARCH-001  │
        │ thru      │         │ thru      │         │ thru      │
        │ ARCH-017  │         │ ARCH-017  │         │ ARCH-017  │
        │ (except   │         │ (except   │         │ (except   │
        │  ARCH-004 │         │  ARCH-004 │         │  ARCH-004 │
        │  workers) │         │  workers) │         │  workers) │
        └─────┬─────┘         └─────┬─────┘         └─────┬─────┘
              │                     │                     │
              └─────────────────────┼─────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
        ┌─────▼─────┐         ┌─────▼─────┐         ┌─────▼─────┐
        │PostgreSQL │         │   Redis   │         │ LP Worker │
        │           │         │           │         │   Pool    │
        │ Primary   │         │ - Cache   │         │           │
        │    +      │         │ - Sessions│         │ ARCH-004  │
        │ Read      │         │ - Job     │         │ workers   │
        │ Replicas  │         │   Queue   │         │ (1-N)     │
        └───────────┘         └───────────┘         └───────────┘
```

**Component Distribution:**
- **API Nodes:** Run all ARCH components except ARCH-004 workers. Horizontally scalable.
- **LP Worker Pool:** Dedicated processes for ARCH-004 job execution using the pure-Go wrapper and packaged native COIN-OR CLP executable. Scaled based on queue depth; the API nodes never execute the solver.
- **Redis:** Single cluster handling cache (ARCH-011), sessions (ARCH-006 via github.com/redis/go-redis/v9), and the ARCH-004 optimization Redis Stream (`XADD`/`XREADGROUP`/`XAUTOCLAIM`/`XACK`).
- **PostgreSQL:** Primary for writes, read replicas for query distribution.

---

### 8.3 Secrets Management

**Principles:**
- No secrets in source code or Docker images
- Secrets injected via environment variables at runtime
- Rotation capability without redeployment

**Secret Categories:**

| Secret | Storage | Rotation Frequency |
|:-------|:--------|:-------------------|
| `DATABASE_URL` | GCP Secret Manager | On credential rotation |
| `REDIS_URL` | GCP Secret Manager | On credential rotation |
| `JWT_SIGNING_KEY` | GCP Secret Manager | Quarterly (with grace period) |
| `STRIPE_SECRET_KEY` | GCP Secret Manager | On compromise or annually |
| `STRIPE_WEBHOOK_SECRET` | GCP Secret Manager | On endpoint recreation |
| `OAUTH_CLIENT_SECRET` (Google/Apple) | GCP Secret Manager | On compromise |

**Platform Secret Managers (by hosting provider):**
- GCP: Secret Manager (primary)
- AWS: Secrets Manager or Parameter Store
- Azure: Key Vault
- Railway/Render/Fly.io: Built-in secret management

---

### 8.4 CI/CD Pipeline

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Commit    │────>│   Build &   │────>│   Deploy    │────>│   Deploy    │
│   to PR     │     │   Test      │     │   Staging   │     │  Production │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                          │                   │                   │
                    ┌─────▼─────┐       ┌─────▼─────┐       ┌─────▼─────┐
                    │ - Lint    │       │ Automatic │       │  Manual   │
                    │ - Unit    │       │ on merge  │       │ promotion │
                    │   tests   │       │ to main   │       │           │
                    │ - Type    │       │           │       │           │
                    │   check   │       │           │       │           │
                    │ - Build   │       │           │       │           │
                    └───────────┘       └───────────┘       └───────────┘
```

**Pipeline Stages:**

| Stage | Trigger | Actions | Failure Behavior |
|:------|:--------|:--------|:-----------------|
| **Build & Test** | PR opened/updated | Lint, type check, unit tests, integration tests, Docker build | Block merge |
| **Staging Deploy** | Merge to `main` | Deploy to staging environment; run smoke tests | Alert team; block production |
| **Production Deploy** | Manual approval | Blue-green deployment; health check verification | Automatic rollback on health check failure |

**Deployment Strategy:**
- Blue-green deployment for zero-downtime releases
- Health checks must pass for 60s before traffic shift
- Automatic rollback if error rate exceeds 1% post-deploy
- Database migrations run as separate pre-deploy step

---

### 8.5 Development Environment

**Docker Compose Services:**
```yaml
services:
  api:           # Go backend (ARCH-001 through ARCH-017, except workers)
  worker:        # ARCH-004 LP workers (Go)
  postgres:      # Primary database
  redis:         # Cache, sessions, job queue (github.com/redis/go-redis/v9)
  maildev:       # Email testing (ARCH-006 verification emails via Resend)
```

**Local Development Commands:**
```bash
# Start all services
docker compose up -d

# Run database migrations
docker compose exec api go run migrate/main.go

# Seed development data
docker compose exec api go run seed/main.go

# View logs
docker compose logs -f api worker

# Run backend tests
docker compose exec api go test ./...

# Run frontend tests (from /home/wiktor/Work/mealswapp directory)
cd frontend && bun test
```
