#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
POLICY = ROOT / "deploy" / "cloudsql" / "backup-policy.json"

REQUIRED_EVIDENCE = {
    "backup_id",
    "restore_started_at",
    "restore_completed_at",
    "migration_check_result",
    "application_readiness_result",
}


def main() -> int:
    policy = json.loads(POLICY.read_text(encoding="utf-8"))
    backup = policy.get("backup", {})
    rehearsal = policy.get("restoreRehearsal", {})

    if policy.get("provider") != "gcp-cloud-sql-postgres":
        raise SystemExit("backup policy must target gcp-cloud-sql-postgres")
    if backup.get("enabled") is not True:
        raise SystemExit("automated backups must be enabled")
    if backup.get("retentionDays") != 30 or backup.get("retainedBackups") != 30:
        raise SystemExit(f"backup retention must be 30 days / 30 backups, got {backup}")
    if backup.get("pointInTimeRecovery") is not True:
        raise SystemExit("point-in-time recovery must be enabled")
    if backup.get("transactionLogRetentionDays", 0) < 7:
        raise SystemExit("transaction log retention must be at least 7 days")
    if rehearsal.get("frequency") != "monthly":
        raise SystemExit("restore rehearsal must be monthly")

    evidence = set(rehearsal.get("requiredEvidence", []))
    missing = REQUIRED_EVIDENCE - evidence
    if missing:
        raise SystemExit(f"restore rehearsal missing evidence fields: {sorted(missing)}")

    alerts = set(policy.get("alerts", []))
    if {"backup-verification-failed", "backup-retention-breach"} - alerts:
        raise SystemExit("backup verification and retention breach alerts must be configured")

    print("validated_backup_retention_days=30")
    print("validated_restore_rehearsal=monthly")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
