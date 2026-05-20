#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SERVICES = ROOT / "deploy" / "gcp" / "services.json"
DEPLOY_WORKFLOW = ROOT / ".github" / "workflows" / "deploy.yml"

REQUIRED_SECRETS = {
    "DATABASE_URL",
    "REDIS_URL",
    "JWT_SECRET",
    "STRIPE_WEBHOOK_SECRET",
    "USDA_API_KEY",
    "MONITORING_NOTIFICATION_EMAIL",
}


def main() -> int:
    services = json.loads(SERVICES.read_text(encoding="utf-8"))
    assert_file("deploy/Dockerfile.backend")
    assert_file("deploy/Dockerfile.frontend")
    assert_file("deploy/nginx/frontend.conf")
    assert_file("deploy/cloudsql/backup-policy.json")
    assert_file("deploy/monitoring/alerts.json")
    assert_file(".github/workflows/deploy.yml")

    if services.get("region") != "europe-west1":
        raise SystemExit("GCP region must be europe-west1")
    if services.get("cloudRun", {}).get("api", {}).get("health", {}).get("startupPath") != "/ready":
        raise SystemExit("Cloud Run API startup health must use /ready")
    if services.get("cloudSql", {}).get("databaseVersion") != "POSTGRES_16":
        raise SystemExit("Cloud SQL must use PostgreSQL 16")
    if services.get("memorystore", {}).get("version") != "REDIS_7_0":
        raise SystemExit("Memorystore must use Redis 7")
    if services.get("frontend", {}).get("cdn") is not True:
        raise SystemExit("frontend CDN must be enabled in deployment contract")

    configured_secrets = set(services.get("secretManager", {}).get("requiredSecrets", []))
    missing = REQUIRED_SECRETS - configured_secrets
    if missing:
        raise SystemExit(f"missing required secrets: {sorted(missing)}")

    workflow = DEPLOY_WORKFLOW.read_text(encoding="utf-8")
    for expected in [
        "google-github-actions/auth@v2",
        "docker build -f deploy/Dockerfile.backend",
        "gcloud run deploy mealswapp-api",
        "gcloud run deploy mealswapp-worker",
        "gsutil -m rsync",
        "python scripts/check.py",
    ]:
        if expected not in workflow:
            raise SystemExit(f"deploy workflow missing {expected}")

    print("validated_deployment_services=api,worker,frontend")
    print(f"validated_required_secrets={len(configured_secrets)}")
    return 0


def assert_file(path: str) -> None:
    if not (ROOT / path).is_file():
        raise SystemExit(f"missing required deployment file: {path}")


if __name__ == "__main__":
    raise SystemExit(main())
