# Phase 02 UAT: API Gateway, Security, Errors, and Observability Baseline

## Scope

Phase 02 implements tasks 50-72 and the foundations described by `DESIGN-010`, `DESIGN-013`, `DESIGN-014`, and `DESIGN-017`.

- Versioned `/api/v1` route registration, request IDs, 10-second deadlines, timeout cancellation, CORS, security headers, deployed TLS redirects, CSRF hooks, request validation hooks, and scoped rate limits.
- Safe `AppError` envelopes, panic recovery, request-correlated structured logs, basic metrics, readiness dependency metrics, and P95 alert-rule defaults.
- AES-256-GCM PII envelope encryption with versioned key-loader interfaces.
- PostgreSQL security-audit persistence through migration `000012_security_audit`.
- OpenAPI gateway source of truth and generated frontend contracts at `frontend/src/lib/api/generated.ts`.

## Automated Evidence

Run from the repository root:

```sh
python3 scripts/check.py
python3 scripts/validate-traceability.py
python3 scripts/validate-task-list.py
cd frontend && bun run check:api-types
```

The aggregate gate verifies migration up/down/up behavior, root and versioned liveness/readiness probes, browser screenshots, Go formatting, backend tests, backend 100% line coverage, frontend build, generated-type drift detection, frontend tests, and frontend 100% line coverage.

## Project-Owner Checks

### Integration

1. Start services and the API:

```sh
bash scripts/start-services.sh
cd backend && go run ./cmd/migrate up && go run ./cmd/api
```

2. Verify probes:

```sh
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
curl -i http://localhost:8080/api/v1/health
curl -i http://localhost:8080/api/v1/ready
```

3. Confirm each response has `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`, `Content-Security-Policy`, and a JSON `requestId`.

### Functional

- Run `cd backend && go test ./internal/httpapi/...` to verify CORS rejection, CSRF acceptance and expiry, validation rejection, scoped rate limiting, timeout cancellation, panic recovery, security headers, request logging, and metrics.
- Run `cd backend && go test ./internal/security/... ./internal/repository/...` to verify encryption key rotation, tamper rejection, email normalization, fail-closed audit mutation policy, and security-audit persistence.

### End To End

- Domain mutation workflows are intentionally deferred until Phase 03 and later phases. Repeat protected-route browser checks when authentication routes are exposed.

### Acceptance

- Confirm production proxy topology before enabling `MEALSWAPP_TRUST_PROXY=true`.
- Confirm the Phase 03 PII field inventory and authorized decrypting service boundaries before wiring encrypted fields into account workflows.

## Deferred Deployment Work

Phase 09 owns deployed GCP Cloud Monitoring resources, notification channels, dashboards, backup monitoring, and trusted reverse-proxy configuration. Phase 04 extends OpenAPI generation with search contracts before Phase 05 consumes them.
