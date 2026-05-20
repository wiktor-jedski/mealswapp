# Data Retention Policy

Task 95 introduces a retention policy service in `backend/internal/services/retention`.

| Data class | Retention | Cleanup behavior |
| --- | --- | --- |
| Sessions | 30 minutes inactive | Delete expired session/cache records |
| Search history | 365 days | Delete persisted `saved_data` entries of search-history kind older than cutoff |
| Exports | 7 days | Delete generated export files or records older than cutoff |
| Deleted accounts | Immediate production deletion | Delete/anonymize remaining production records when account deletion has completed |
| Audit logs | 7 years | Preserve operational/security audit evidence, then delete beyond legal-retention window |
| Import records | 365 days | Delete stale external import records and payloads |
| Optimization jobs | 1 hour | Delete completed/failed/cancelled job results after status result TTL |
| Backups | 30 days | Enforced by backup manager/deployment configuration, verified separately |

The service supports dry runs through the same `DeleteBefore` interface used for enforcement. Current repository storage is partly interface-backed, so task 95 verifies the policy and cutoffs with a fake store; concrete PostgreSQL/Redis maintenance jobs can implement the store interface without changing policy rules.

Validation:

```bash
go test ./internal/services/retention -count=1
```
