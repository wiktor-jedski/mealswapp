# Phase Acceptance Notes

This document records acceptance evidence for the implementation plan in `02_TASK_LIST.md`.

## Automated Evidence

Current repository-wide validation:

```bash
python scripts/check.py
```

This runs documentation/task-list validation, Docker Compose config validation, backend Go tests, frontend Bun unit tests, frontend production build, performance gates, monitoring config validation, backup policy validation, deployment config validation, and migration filename validation.

Additional browser validation:

```bash
cd frontend
bun run e2e
```

The Playwright suite covers deterministic fixtures for registration/login API contracts, basic search rendering, paid-mode gating, saved/history/favorites/settings/account deletion controls, and admin external import.

## Manual Or Staging Acceptance

Before production release, the project owner should rehearse these checks in staging:

| Area | Acceptance check |
| --- | --- |
| Auth/account | Register, log in, refresh, log out, request reset, verify email, delete account, confirm session/cache purge |
| Search | Single-item, replacement, and diet flows with real PostgreSQL seed data and external provider credentials |
| Subscription | Stripe checkout, portal return, webhook delivery, duplicate webhook delivery, entitlement refresh |
| Optimization | Submit diet job, observe queued/processing/completed/timeout states through API and worker |
| Admin | External USDA/OFF search, curated import, item/tag transitions, user disable/reset/audit views |
| Offline | Cached search render, offline banner, reconnect retry, service worker update and purge behavior |
| Accessibility | Keyboard-only search/admin/settings flows in desktop and mobile browser sizes |
| Operations | `/health`, `/ready`, metrics, monitoring alerts, backup policy validation, restore rehearsal |
| Deployment | Cloud Run API/worker deploy, frontend bucket sync, Secret Manager resolution, migration execution |

## Accepted Coverage Exceptions

| Area | Exception |
| --- | --- |
| Browser E2E auth UI | There is no dedicated frontend registration/login form yet; task 99 covers auth API contracts with deterministic browser fixtures. |
| Export button wiring | Settings exposes export controls, but current `SearchView` does not wire `onExport` to the API yet. |
| Offline Playwright depth | Unit and service-worker policy tests cover most offline behavior; full browser-level cache/offline toggling should be expanded in staging. |
| Heap measurement | Task 93 documents the 64 MiB client heap target, but automated browser heap measurement needs a CDP harness or Playwright extension. |
| PII encryption wiring | AES-256-GCM helper exists; repository-level field encryption needs field-by-field migration work before production PII storage. |
| Deployment dry run | Deployment manifests and workflow are validated locally; real GCP project IDs, buckets, service accounts, and secrets must be supplied in staging. |

## Current Risk Summary

The implementation is suitable for staging validation, not direct production launch. The main remaining risks are integration risks: real provider credentials, real Stripe webhook delivery, real GCP Secret Manager resolution, Cloud SQL/Redis latency, backup restore evidence, and browser E2E depth for workflows whose UI is still minimal.
