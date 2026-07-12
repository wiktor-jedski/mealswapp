#!/usr/bin/env python3

"""Validate identifier-leading Go Doc for exported Phase 07 constants."""

# Implements DESIGN-004 exported queue, solver, and worker vocabulary documentation gate.

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PHASE07_PACKAGES = ("dailydiet", "optimization", "queue", "worker")
DECLARATION = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*)\s+(?:[^=]+\s)?=|^\s*([A-Z][A-Za-z0-9_]*)\s*=")


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
            continue
        match = DECLARATION.match(line)
        if not match:
            continue
        name = match.group(1) or match.group(2)
        comment_start = index - 1
        while comment_start >= 0 and lines[comment_start].strip().startswith("//"):
            comment_start -= 1
        first_comment = lines[comment_start + 1].strip() if comment_start + 1 < index else ""
        if not first_comment.startswith(f"// {name}"):
            failures.append(f"{path.relative_to(ROOT)}:{index + 1}: Go Doc must start with {name}")
    return failures


def main() -> int:
    failures: list[str] = []
    for package in PHASE07_PACKAGES:
        for path in sorted((ROOT / "backend" / "internal" / package).glob("*.go")):
            if not path.name.endswith("_test.go"):
                failures.extend(validate_file(path))
    if failures:
        raise SystemExit("\n".join(failures))
    print("Phase 07 exported Go Doc validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
