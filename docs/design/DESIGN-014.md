## FILE: DESIGN-014.md
**Traceability:** ARCH-014

**Static aspects covered:** LogAggregator, MetricsCollector, AlertManager, UptimeMonitor, FiberLogger.

### 0. Static Aspect Responsibilities
- `LogAggregator`: owns structured log ingestion into GCP Cloud Monitoring and retention expectations.
- `MetricsCollector`: owns latency, error, connection, dependency, queue, and worker metrics.
- `AlertManager`: owns alert rule registration, evaluation, severity, and notification handoff.
- `UptimeMonitor`: owns health/readiness probes and availability calculation.
- `FiberLogger`: owns request log emission from Fiber middleware.

### 1. Data Structures & Types
- `interface LogEvent { requestId: string; service: string; level: "debug" | "info" | "warn" | "error"; message: string; fields: map[string]any; createdAt: time.Time }`
- `interface MetricPoint { name: string; value: float64; unit: string; labels: map[string]string; observedAt: time.Time }`
- `interface AlertRule { name: string; metric: string; threshold: float64; comparison: ">" | "<" | ">=" | "<="; durationSeconds: number; severity: "warning" | "critical" }`
- `interface UptimeCheck { name: string; url: string; intervalSeconds: number; timeoutSeconds: number; expectedStatus: number }`
- `interface BackupVerification { backupId: string; completedAt: time.Time; restoreTestedAt?: time.Time; status: "passed" | "failed" | "pending" }`

### 2. Logic & Algorithms (Step-by-Step)
1. Configure Fiber logger middleware to emit structured request logs with request ID, route, status, latency, and user ID when known.
2. Send application logs and metrics to GCP Cloud Monitoring.
3. Collect response-time metrics per endpoint and compute P95 latency.
4. Collect error rates, concurrent connections, Redis health, database health, queue depth, and worker utilization.
5. Run uptime checks against `/health` and `/ready` every 30 seconds.
6. Define warning alert when P95 latency exceeds 1.5 seconds and critical alert when it exceeds 2 seconds.
7. Retain logs for at least 90 days.
8. Monitor daily backup completion and scheduled restore tests.
9. Correlate audit events from ARCH-013 with service logs by request ID.

### 3. State Management & Error Handling
- `healthy`: health and readiness checks pass.
- `degraded`: non-critical dependency is down but core service responds.
- `unhealthy`: readiness fails and load balancer should stop sending traffic.
- `alert_pending`: metric breached threshold but duration has not elapsed.
- `alert_firing`: alert rule duration elapsed; notify configured channels.
- `logging_backpressure`: drop debug logs first and preserve error/security logs.
- `backup_verification_failed`: emit critical alert and retain failed restore evidence.

### 4. Component Interfaces
- `func Log(ctx context.Context, event LogEvent) error`
- `func RecordMetric(ctx context.Context, point MetricPoint) error`
- `func RegisterAlertRule(rule AlertRule) error`
- `func EvaluateAlertRules(ctx context.Context, now time.Time) ([]AlertRule, error)`
- `func HealthHandler(ctx *fiber.Ctx) error`
- `func ReadinessHandler(ctx *fiber.Ctx) error`
- `func VerifyBackup(ctx context.Context, backupID string) (BackupVerification, error)`
- `func FiberLogger() fiber.Handler`
