# Task 285 Centralized Logging Acceptance

This verifier implements the deployed SW-REQ-084 acceptance boundary from
`docs/design/DESIGN-014.md`. It drives the production frontend and API, then
queries GCP Cloud Logging through `entries:list`. Local API stdout, Playwright
responses, and browser traces are never accepted as centralized-log evidence.

## Deployment preconditions

Use a dedicated deployed test environment with:

- an administrator fixture and a privacy-minimized user-lookup fixture;
- controlled external-provider success and dependency-failure responses;
- an acceptance-only gateway or database fault control that makes requests
  carrying `X-Mealswapp-Acceptance-Scenario: audit-failure` fail at audit
  persistence, and requests carrying `dependency-failure` observe the
  controlled provider outage;
- cleanup authorization for the exact global item and classification created
  by the run;
- structured events exported to GCP Cloud Logging;
- a least-privilege Application Default Credentials identity able to call
  `logging.entries.list` only for the configured test project; and
- one safe request-correlation probe older than 90 days whose event contains
  only `requestId`, `action=logging_retention`,
  `resource=centralized_log`, and `outcome=retained`.

The acceptance-only fault controls must not be enabled on a production
deployment. The application contains no test bypass for these headers.

## Configuration

Set these values in the operator environment. Do not put them in shell history,
committed files, or command arguments.

```text
MEALSWAPP_TASK285_DEPLOYMENT_ACK=deployed-test-with-centralized-log-sink
MEALSWAPP_TASK285_DEPLOYED_BASE_URL=https://deployed-test.example
MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST=deployed-test.example
MEALSWAPP_TASK285_GCP_PROJECT=<test-project>
MEALSWAPP_TASK285_LOG_RESOURCE_NAMES=projects/<test-project>
MEALSWAPP_TASK285_ADMIN_EMAIL=<fixture-admin-email>
MEALSWAPP_TASK285_ADMIN_PASSWORD=<fixture-admin-password>
MEALSWAPP_TASK285_EXTERNAL_QUERY=<controlled-success-query>
MEALSWAPP_TASK285_DEPENDENCY_QUERY=<controlled-outage-query>
MEALSWAPP_TASK285_USER_LOOKUP=<fixture-user-email-or-id>
MEALSWAPP_TASK285_RETENTION_REQUEST_ID=<safe-request-uuid>
MEALSWAPP_TASK285_RETENTION_OCCURRED_AT=<rfc3339-timestamp-at-least-90-days-old>
```

`MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST` is the exact DNS name owned and
approved by the operator for this deployed test. The runner resolves it and
rejects every loopback, private, link-local, reserved, unspecified, multicast,
or otherwise non-global address. IP literals and DNS names must match the
approved host exactly.

Optional query bounds are
`MEALSWAPP_TASK285_LOG_POLL_SECONDS` (maximum 10),
`MEALSWAPP_TASK285_LOG_TIMEOUT_SECONDS` (maximum 300), and
`MEALSWAPP_TASK285_LOG_PAGE_SIZE` (maximum 1000).

Run:

```text
python3 scripts/run-task285-acceptance.py
```

Exit `0` means every criterion passed, `1` means at least one criterion failed,
and `2` means no criterion failed but at least one was blocked. Missing
deployment configuration, sink authority, complete ingestion, or retention
proof intentionally returns `2` and synchronizes `P08-FIND-285-001`.

Sanitized action/sink artifacts are written under `logs/task285-deployed/`.
The Task 280 requirement report is written under
`logs/phase08-acceptance/task285-<run-id>/`.
