# Backup And Restore

Task 96 defines the Cloud SQL backup contract in `deploy/cloudsql/backup-policy.json`.

Required production settings:

| Setting | Value |
| --- | --- |
| Automated backups | Enabled |
| Backup cadence | Daily |
| Backup start time | 03:00 UTC |
| Backup retention | 30 days / 30 retained backups |
| Point-in-time recovery | Enabled |
| Transaction log retention | 7 days |
| Restore rehearsal | Monthly to staging or local |

Restore rehearsal checklist:

1. Record the source `backup_id`.
2. Restore to a staging or local PostgreSQL instance, never over production.
3. Apply migrations with `python scripts/check.py` or the deployment migration step.
4. Run readiness checks against `/ready`.
5. Run a smoke search and admin read-only check.
6. Record `restore_started_at`, `restore_completed_at`, migration result, readiness result, and any exceptions.
7. Keep failed restore evidence and trigger the `backup-verification-failed` alert.

Local validation:

```bash
python scripts/validate_backup_policy.py
```

The validator is intentionally secret-free and checks retention/PITR/rehearsal requirements before task 98 translates the contract into concrete GCP infrastructure.
