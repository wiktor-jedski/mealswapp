# Monitoring Configuration

Task 94 defines the first deployable monitoring contract in `deploy/monitoring/alerts.json`.

The manifest is secret-free. It references `MONITORING_NOTIFICATION_EMAIL` as a deployment-time secret or environment binding instead of storing notification targets in git.

Configured checks:

| Area | Rule |
| --- | --- |
| Uptime | `/health` and `/ready` every 30 seconds, 5 second timeout, expected HTTP 200 |
| Latency | Warning when API P95 latency is above 1.5 seconds for 5 minutes; critical above 2 seconds for 5 minutes |
| Error rate | Critical when 5xx errors exceed 5% of requests for 5 minutes |
| Dependency health | Critical when database readiness metric is unhealthy for 60 seconds; warning when Redis is unhealthy for 120 seconds |
| Queue failures | Critical on optimization queue failure metric above zero for 5 minutes |
| Webhook failures | Critical on Stripe webhook failure metric above zero for 5 minutes |

Local validation:

```bash
python scripts/validate_monitoring.py
```

The validator checks required alert classes, severity values, uptime paths, durations, and secret-free notification references. GCP-specific policy IDs and real notification channels are intentionally left for staging/production deployment configuration.
