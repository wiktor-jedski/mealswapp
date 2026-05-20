#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "deploy" / "monitoring" / "alerts.json"

REQUIRED_ALERTS = {
    "api-p95-latency-warning": ("warning", 1500),
    "api-p95-latency-critical": ("critical", 2000),
    "api-error-rate-critical": ("critical", 0.05),
    "database-unhealthy-critical": ("critical", 1),
    "redis-unhealthy-warning": ("warning", 1),
    "optimization-queue-failures-critical": ("critical", 0),
    "stripe-webhook-failures-critical": ("critical", 0),
}

REQUIRED_UPTIME_PATHS = {"/health", "/ready"}
VALID_SEVERITIES = {"warning", "critical"}
VALID_COMPARISONS = {">", "<", ">=", "<="}


def main() -> int:
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    uptime_paths = {check.get("path") for check in manifest.get("uptimeChecks", [])}
    if not REQUIRED_UPTIME_PATHS.issubset(uptime_paths):
        raise SystemExit(f"missing uptime checks: {sorted(REQUIRED_UPTIME_PATHS - uptime_paths)}")

    for check in manifest.get("uptimeChecks", []):
        if check.get("intervalSeconds") != 30 or check.get("timeoutSeconds") > 5 or check.get("expectedStatus") != 200:
            raise SystemExit(f"invalid uptime check: {check}")

    alerts = {policy.get("name"): policy for policy in manifest.get("alertPolicies", [])}
    missing_alerts = REQUIRED_ALERTS.keys() - alerts.keys()
    if missing_alerts:
        raise SystemExit(f"missing alert policies: {sorted(missing_alerts)}")

    for name, (severity, threshold) in REQUIRED_ALERTS.items():
        policy = alerts[name]
        if policy.get("severity") != severity or policy.get("threshold") != threshold:
            raise SystemExit(f"invalid alert policy threshold/severity for {name}: {policy}")
        if policy.get("comparison") not in VALID_COMPARISONS:
            raise SystemExit(f"invalid comparison for {name}: {policy}")
        if policy.get("durationSeconds", 0) < 60:
            raise SystemExit(f"alert duration too short for {name}: {policy}")

    for channel in manifest.get("notificationChannels", []):
        if "secretRef" not in channel:
            raise SystemExit(f"notification channel must use secretRef, got: {channel}")

    print(f"validated_monitoring_alerts={len(alerts)}")
    print(f"validated_uptime_checks={len(uptime_paths)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
