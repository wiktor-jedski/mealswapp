# Task 285 Preparation Evidence

Date: 2026-07-29  
Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`  
Task status: `PREPARED` (task-list row intentionally not edited by this repair)  
Task row: Phase 08.02 Centralized Logging Acceptance Scenarios

## Repair result

Every finding in `task-285-review.md`, `task-285-rereview.md`, and
`task-285-final-review.md` is addressed:

- `sanitize_entries` recursively scans the complete raw Cloud Logging entry
  before projection. This includes `jsonPayload`, `textPayload`,
  `httpRequest`, labels, and arbitrarily nested maps and arrays. Sensitive key
  matching now normalizes camelCase, snake_case, punctuation, and casing, then
  rejects names, text/diagnostic payloads, request metadata, authorization,
  credentials, tokens, and key material at every depth. The normalized
  key-material policy explicitly rejects `privateKey`, `publicKey`,
  `encryptionKey`, `signingKey`, `clientKey`, `accessKey`, `refreshKey`,
  `private_key`, and equivalent case/separator variants.
- The deployed-origin boundary requires the exact operator-approved hostname,
  resolves every address, and rejects non-global IPv4, IPv6, and IPv4-mapped
  IPv6 targets. The dedicated Playwright configuration independently rejects
  local/non-public literals, resolves the approved DNS hostname before use,
  rejects empty or failed DNS responses, and fails closed when any returned
  address is private, loopback, link-local, shared, multicast, documentation,
  reserved, malformed, or IPv4-mapped.
- The public `--actions-file` option is removed. `verify` always invokes the
  Playwright producer in production, generates a per-run 48-hex provenance
  value, and accepts only a receipt carrying that value and the exact canonical
  category-to-action/resource/outcome map. The injectable producer is
  underscore-prefixed and exists only as a focused-test seam.
- One monotonic deadline now spans token acquisition, every paginated HTTP
  request, every poll, and bounded sleeps. Each operation receives only the
  remaining budget.
- Playwright timeout/spawn failures, malformed or reversed action receipts,
  invalid CLI input, unexpected validation errors, and Task 280 report
  timeout/spawn failures produce complete seven-row structured `BLOCKED`
  evidence with nonzero exit and no traceback. If Task 280 itself cannot be
  spawned, a safe fallback report retains `P08-FIND-285-001`.

The action producer now includes provenance in
`mealswapp.task285-actions.v1`. The operator contract documents
`MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST` as an exact owned and approved DNS
name. No local console, browser response, trace, screenshot, or caller-supplied
receipt is accepted as centralized-log evidence.

## Focused regression evidence

`scripts/test_run_task285_acceptance.py` has 22 focused tests and
`frontend/task285-target.test.ts` adds three resolver-boundary tests. New
regressions prove:

- every named key-material alias and representative case/separator variant is
  rejected at the exact
  `metadata -> level1 -> list -> level2 -> alias` shape; the regression failed
  all 14 subtests before the policy repair and passes after it;
- mixed `textPayload`, `httpRequest.requestUrl`, and deeply nested sensitive
  metadata are rejected even beside a safe `jsonPayload`; direct regressions
  cover nested `displayName`, diagnostic `textPayload`,
  `httpRequest.userAgent`, `authorization`, and camel/snake-case aliases;
- loopback, private, link-local, reserved, IPv6 loopback, mapped IPv6
  loopback, ownership mismatch, failed/empty DNS, malformed results, mixed
  public/private results, shared, multicast, and privately resolving DNS
  targets fail closed in both runner-owned boundaries;
- mutated canonical tuples, wrong provenance, and reversed receipt timestamps
  cannot produce PASS;
- slow pagination consumes one cumulative deadline rather than resetting a
  request timeout per page;
- Playwright timeout/spawn failures and CLI/input/validation/report failures
  emit complete structured `BLOCKED` reports.

Fresh focused and dependency execution:

```text
python3 -m unittest -v scripts/test_run_task285_acceptance.py
PASS: 22 tests

python3 -m unittest -v scripts/test_run_task285_acceptance.py scripts/test_phase08_acceptance.py
PASS: 47 tests

cd frontend && bun test ./task285-target.test.ts
PASS: 3 tests

cd frontend && bun run typecheck
PASS

MEALSWAPP_TASK285_DEPLOYMENT_ACK=deployed-test-with-centralized-log-sink \
MEALSWAPP_TASK285_DEPLOYED_BASE_URL=https://example.com \
MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST=example.com \
bunx playwright test -c playwright.task285.config.ts \
  tests/task285-centralized-logging.spec.ts --list
PASS: 4 deployed scenarios discovered after independent public DNS validation
```

## Truthful deployed result

The workspace still has no approved deployed origin, deployed fixtures, GCP
project scope, sink-reader authority, or 90-day retention probe. A fresh
environment-cleared run therefore returned exit `2` with exactly `0 PASS`,
`0 FAIL`, and `7 BLOCKED`.

- run ID: `task285-17145c6d0ea585123024c5a5`
- status: `BLOCKED`
- synchronized finding: `P08-FIND-285-001`
- root cause: `ROOT-T285-DEPLOYED-ENVIRONMENT`
- report:
  `logs/phase08-acceptance/task285-17145c6d0ea585123024c5a5/report.json`
- sink evidence:
  `logs/task285-deployed/17145c6d0ea585123024c5a5/sink-evidence.json`

All seven rows retain empty request IDs and fixed
`log_sink_state=blocked` evidence. This is truthful missing-deployment
evidence, not a deployed PASS.

## Repository gates

Fresh executions after the repair:

```text
python3 scripts/check.py --quick
PASS

python3 scripts/check.py --output logs/task285-full-check.html
PASS

python3 scripts/phase08_acceptance.py validate
PASS: 12 scenarios, 91 criteria

python3 scripts/validate-task-list.py
PASS: 286 sequential tasks

python3 scripts/validate-traceability.py
PASS

git diff --check
PASS
```

The full gate completed frontend build/typecheck, 537 frontend unit tests,
309 maintained browser tests with 77 environment-gated skips, OpenAPI and
generated-contract checks, Go formatting/vet/vulnerability checks, local-stack
and UAT lanes, backend integration and coverage lanes, and the repository's
documented coverage contracts. Recorded aggregate coverage remains 96.06%
frontend lines, 87.4% Go internal statements, and exact Phase 08 Go coverage
`4645/4986` (`93.2%`). The aggregate report is quality-gate evidence only and
is not treated as centralized-log evidence.

The fresh quick gate passed on its first run. The first full-gate run reached
all backend and static checks but the browser screenshot helper timed out
waiting for the `Food search` label. The exact frontend verifier was rerun
immediately and passed, including all desktop/mobile captures. The complete
full gate was then rerun from the start and passed all lanes; that successful
rerun produced the authoritative aggregate artifact.

## SHA-256 inventory

```text
54f85c35b95b4a1366023a61a26a5ee70c0fbafd9272aad4658d40fe273c62cc  scripts/task285_log_sink.py
29f8cce15ef70e5e208dad874673f36a9033eeefb70eebad13d1553de57c0f58  scripts/run-task285-acceptance.py
22e31909146027f6c0e2a9241222eb899095e137a3a5afe75ac26fa16c2c413c  scripts/test_run_task285_acceptance.py
4343808d63552d481671815303456185ef76ef0aa0752cb5ae84756fa480d281  scripts/check.py
ec7c6f7b660af286ee2ec5d9983bdf1933994c7b7db2e5be9c71fbcc496b1a23  frontend/playwright.task285.config.ts
ecd72a6fd546a29915786ac8f33c21c5182cc81ad64f7f7f475a9c6ef1e56f49  frontend/task285-target.ts
f6738f73badede24c88d4c3d6684129d8e84a12384190489681ba9fdae5a626c  frontend/task285-target.test.ts
040f0e5763f9f73c65f1f8b701de57dc0854bbac936ba008ac3e469fbbe5137e  frontend/tests/task285-centralized-logging.spec.ts
442de77c47d1c57f4bb564f810023eb0ce1458dde59117cda3eb15059ebba31a  docs/operations/task285-centralized-logging-acceptance.md
de951074a8dc70f49c01eedc99af2cf56c0d7b5894733c838190fa93defbdf15  docs/implementation/04_OPEN.md
d18286a98a3e957cf7a2f4ece91c517337513c2b78a3c5dbd1a9983492d77e5b  docs/testing/phase08/finding-history.json
64803ce4d4af01bcda803dc843623c2ab20fae73de75b4701153c6602946aa63  docs/implementation/02_TASK_LIST.md
8c76b258a70417fd9bb9ec579df44b750d5332d7813b66d8ea584df0f2c41946  logs/phase08-acceptance/task285-17145c6d0ea585123024c5a5/report.json
124b899b1a6ad0405b38ad91b835db9b665b83bc2e4131f2d83fb5001c0a5585  logs/task285-deployed/17145c6d0ea585123024c5a5/sink-evidence.json
0fa4788eee6426d43143241122ebaa34c79520cc4bff26b89aaddb5214d60617  logs/task285-full-check.html
```

The task-list hash records the existing `PREPARED` row only; this repair did
not modify `docs/implementation/02_TASK_LIST.md`.
