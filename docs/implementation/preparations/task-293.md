# Task 293 Repair Preparation

## Result

Task 293's rejected-review repair synchronizes the Phase 08 frontend coverage contract with the repository's exact current Bun target and records local Redocly evidence. No task-list row was edited.

## Dependency gate

Task 293 declares dependencies `239, 267, 280, 284`. Their current orchestrator states are `PASSED`, `PREPARED`, `PASSED`, and `PASSED`, respectively. Dependency 267 has implementation and verification evidence in [task-267.md](task-267.md), but remains `PREPARED` in `docs/implementation/02_TASK_LIST.md`. The preparation contract does not permit this role to promote another task or edit its status; the gate is therefore resolved as far as allowed, with orchestrator promotion of 267 to `PASSED` still required before Task 293 becomes eligible.

## Repair evidence

- Backend `internal/userdata` coverage: `95.8%`; `export.go` is `99/104` statements (`94.2%`) with the existing documented B1 exception ranges.
- Frontend full coverage: `537` tests, `2,820` expectations, `All files` at `95.19%` functions and `96.09%` lines; `account-data-client.ts` is `100.00%` functions and `99.12%` lines with no stable uncovered-line range. The exact contract in `docs/implementation/04_OPEN.md` matches this output.
- Redocly lint: local `/home/wiktor/Work/mealswapp/node_modules/.bin/redocly lint api/openapi.yaml` passes; the only output is the pre-existing ignored OAuth callback 302-only response warning.
- Repository coverage-contract tests: `python3 -m unittest scripts/test_check_coverage.py` passes.
- `python3 scripts/check.py --quick` is not a Task 293 failure: its unrelated Phase 08 acceptance fixtures fail on stale `docs/implementation/02_TASK_LIST.md` evidence hashes and missing dependency-280 synchronized reports. No unrelated fixture or task row was changed.

## Required handoff

The orchestrator must promote dependency 267 from `PREPARED` to `PASSED` before selecting or promoting Task 293. This file records that gate without changing unrelated task rows.
