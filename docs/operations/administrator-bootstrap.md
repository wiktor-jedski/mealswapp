# Administrator bootstrap

This procedure creates the first Mealswapp administrator from an existing, verified account. It is an operator-only database operation, not an API endpoint or normal role-management workflow.

## Preconditions

- Run migrations first.
- The target user must already be able to sign in with a verified email/password or OAuth Login Method. Bootstrap never creates credentials and never marks email verified.
- Use an operator environment with access to the intended PostgreSQL database and the same `MEALSWAPP_LOCAL_SECRET_KEY` used by the deployment.
- Confirm that no administrator already exists. The command enforces this again transactionally.

## Run

Prefer interactive email input so the address does not enter shell history:

```sh
cd backend
MEALSWAPP_ENV=development go run ./cmd/admin-bootstrap --environment development
```

The command prompts for the email on standard input, applies the same trim/validate/lower-case canonicalization used by registration, login, password reset, and OAuth linking, and performs the configured encrypted-email lookup. A historical mixed-case digest is reindexed only when the exact trimmed legacy spelling resolves that account; if canonical and legacy digests resolve different accounts, bootstrap refuses to merge or promote either. For automation where the history and process-list exposure are separately controlled, `--email` is available. `--user-id <uuid>` is an advanced alternative and cannot be combined with email.

Production requires both the exact environment and a separate confirmation:

```sh
cd backend
MEALSWAPP_ENV=production go run ./cmd/admin-bootstrap \
  --environment production \
  --confirm-production
```

The output contains only the result, target user UUID, audit UUID when a new audit was written (`none` on unchanged replay), operator actor kind, and reauthentication requirement. It never prints the email, credential material, database URL, encryption/lookup details, hashes, salts, tokens, or cookies.

## Outcomes and audit

- First successful run atomically changes the selected role to `admin` and writes one `bootstrap_administrator` audit entry.
- The audit actor is `operator`, with no administrator user actor. The selected user UUID is recorded only as the affected entity.
- Shared administrative audit readback preserves the `operator` actor kind and nullable administrator actor instead of presenting the operator as a zero or authenticated administrator UUID.
- Repeating the command for the same account is an unchanged success and writes no second audit entry.
- If another administrator exists, the command refuses the operation.
- Concurrent attempts are serialized. Only one administrator and one bootstrap audit effect can be created.
- Any audit failure rolls back the role change.

After success, the promoted user must sign out and sign in again. Existing access and refresh sessions retain their old signed role and verification claims and cannot gain administrator access in place.
