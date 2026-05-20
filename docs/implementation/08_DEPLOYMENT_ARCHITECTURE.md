# Deployment Architecture

Task 98 defines a GCP deployment contract for staging and production.

Artifacts:

| Area | File |
| --- | --- |
| Backend API/worker image | `deploy/Dockerfile.backend` |
| Frontend static image/CDN build | `deploy/Dockerfile.frontend`, `deploy/nginx/frontend.conf` |
| GCP service contract | `deploy/gcp/services.json` |
| Monitoring | `deploy/monitoring/alerts.json` |
| Cloud SQL backups | `deploy/cloudsql/backup-policy.json` |
| GitHub Actions deploy workflow | `.github/workflows/deploy.yml` |

Target services:

| Component | GCP service |
| --- | --- |
| API | Cloud Run service `mealswapp-api` |
| Worker | Cloud Run service `mealswapp-worker` using `/app/worker` |
| PostgreSQL | Cloud SQL PostgreSQL 16 |
| Redis/cache/queue | Memorystore Redis 7 |
| Frontend assets | Cloud Storage bucket with Cloud CDN |
| Secrets | Secret Manager values injected into Cloud Run |
| Images | Artifact Registry repository `mealswapp` |

Secret references are names only; no secret values are committed. Required secrets are `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `STRIPE_WEBHOOK_SECRET`, `USDA_API_KEY`, and `MONITORING_NOTIFICATION_EMAIL`.

Validation:

```bash
python scripts/validate_deployment.py
python scripts/check.py
```

The deploy workflow runs the full project check, builds/pushes the backend image, deploys API and worker services, builds the frontend, and syncs `frontend/dist` to the configured Cloud Storage bucket. Concrete project IDs, service accounts, and buckets come from GitHub environment secrets.
