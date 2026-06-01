#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector implementation task-list quality gate.

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TASK_LIST = ROOT / "docs" / "implementation" / "02_TASK_LIST.md"
VALID_STATUSES = {"OPEN", "PREPARED", "REJECTED", "PASSED"}
STATIC_ASPECT_RE = re.compile(r"^(?:DESIGN|ARCH)-\d{3}: [A-Za-z][A-Za-z0-9]*$")
TASK_ROW_RE = re.compile(r"^\| \d+ \|")


def parse_task_rows() -> list[list[str]]:
	rows = []
	for line_number, line in enumerate(TASK_LIST.read_text(encoding="utf-8").splitlines(), start=1):
		if not TASK_ROW_RE.match(line):
			continue
		columns = [column.strip() for column in line.strip("|").split("|")]
		if len(columns) != 9:
			raise ValueError(f"line {line_number}: expected 9 columns, found {len(columns)}")
		rows.append(columns)
	return rows


def validate_task_rows(rows: list[list[str]]) -> list[str]:
	errors = []
	if not rows:
		return ["task list contains no task rows"]

	ids = [int(row[0]) for row in rows]
	expected_ids = list(range(1, len(rows) + 1))
	if ids != expected_ids:
		errors.append(f"task IDs must be sequential from 1: found {ids}")

	known_ids = set(ids)
	for row in rows:
		task_id = int(row[0])
		status = row[3]
		retries = row[4]
		dependencies = row[6]

		if status not in VALID_STATUSES:
			errors.append(f"task {task_id}: invalid status {status!r}")
		if not retries.isdigit():
			errors.append(f"task {task_id}: retries must be a non-negative integer")
		if not STATIC_ASPECT_RE.fullmatch(row[2]):
			errors.append(f"task {task_id}: static aspect must match `DESIGN-NNN: Aspect` or `ARCH-NNN: Aspect`")

		for dependency in [value.strip() for value in dependencies.split(",") if value.strip()]:
			if not dependency.isdigit():
				errors.append(f"task {task_id}: dependency {dependency!r} must be a numeric task ID")
				continue
			dependency_id = int(dependency)
			if dependency_id not in known_ids:
				errors.append(f"task {task_id}: dependency {dependency_id} does not exist")
			elif dependency_id >= task_id:
				errors.append(f"task {task_id}: dependency {dependency_id} must reference an earlier task")

	return errors


def main() -> int:
	try:
		rows = parse_task_rows()
	except ValueError as exc:
		print(f"Task-list validation failed:\n- {exc}")
		return 1

	errors = validate_task_rows(rows)
	if errors:
		print("Task-list validation failed:")
		for error in errors:
			print(f"- {error}")
		return 1

	print(f"Task-list validation passed: {len(rows)} sequential tasks with ordered dependencies.")
	return 0


if __name__ == "__main__":
	sys.exit(main())
