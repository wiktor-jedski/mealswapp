#!/usr/bin/env python3

"""Validate identifier-leading Go Doc for exported phase constants and errors."""

# Implements DESIGN-004, DESIGN-009, DESIGN-012, and DESIGN-014 exported Go vocabulary documentation gate.

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PHASE07_PACKAGES = ("dailydiet", "optimization", "queue", "worker")
PHASE08_PACKAGES = ("externaldata", "dataimporter", "itemcurator", "useradmin")
PHASE08_FILES = (ROOT / "backend" / "internal" / "observability" / "admin_external.go",)
DECLARATION = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*)\s+(?:[^=]+\s)?=|^\s*([A-Z][A-Za-z0-9_]*)\s*=")
SINGLE_DECLARATION = re.compile(r"^\s*(?:const|var)\s+([A-Z][A-Za-z0-9_]*)\s+(?:[^=]+\s)?=")


def validate_comment(path: Path, lines: list[str], index: int, name: str) -> str | None:
    comment_start = index - 1
    while comment_start >= 0 and lines[comment_start].strip().startswith("//"):
        comment_start -= 1
    first_comment = lines[comment_start + 1].strip() if comment_start + 1 < index else ""
    if first_comment != f"// {name}" and not first_comment.startswith(f"// {name} "):
        return f"{path.relative_to(ROOT)}:{index + 1}: Go Doc must start with {name}"
    return None


def validate_file(path: Path) -> list[str]:
    lines = path.read_text(encoding="utf-8").splitlines()
    failures: list[str] = []
    in_group = False
    for index, line in enumerate(lines):
        stripped = line.strip()
        if stripped in {"const (", "var ("}:
            in_group = True
            continue
        if in_group and stripped == ")":
            in_group = False
            continue
        if not in_group:
            match = SINGLE_DECLARATION.match(line)
            if match:
                failure = validate_comment(path, lines, index, match.group(1))
                if failure:
                    failures.append(failure)
            continue
        match = DECLARATION.match(line)
        if not match:
            continue
        name = match.group(1) or match.group(2)
        failure = validate_comment(path, lines, index, name)
        if failure:
            failures.append(failure)
    return failures


def main() -> int:
    failures: list[str] = []
    for package in PHASE07_PACKAGES + PHASE08_PACKAGES:
        for path in sorted((ROOT / "backend" / "internal" / package).glob("*.go")):
            if not path.name.endswith("_test.go"):
                failures.extend(validate_file(path))
    for path in PHASE08_FILES:
        failures.extend(validate_file(path))
    if failures:
        raise SystemExit("\n".join(failures))
    print("Phase 07 and Phase 08 exported Go Doc validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
