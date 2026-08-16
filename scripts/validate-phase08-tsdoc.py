#!/usr/bin/env python3

"""Validate TSDoc on hand-written Phase 08 frontend exports."""

# Implements DESIGN-009 UserAdminPanel exported frontend documentation gate.

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PHASE08_SOURCES = (
	"frontend/src/lib/admin-access.ts",
	"frontend/src/lib/admin-workflows.ts",
	"frontend/src/lib/api/account-data-client.ts",
	"frontend/src/lib/api/admin-client.ts",
	"frontend/src/lib/api/external-admin-client.ts",
	"frontend/src/lib/api/filter-options-client.ts",
	"frontend/src/lib/api/generated.phase08-typecheck.ts",
	"frontend/src/lib/substitution-filter-options.ts",
)
EXPORT = re.compile(
	r"(?m)^export\s+(?:async\s+)?(?:type|interface|class|function|const)\s+"
	r"(?P<name>[A-Za-z_$][A-Za-z0-9_$]*)"
)
TSDOC = re.compile(r"/\*\*(?P<body>.*?)\*/", re.DOTALL)


def validate_file(path: Path) -> list[str]:
	"""Return missing or malformed TSDoc diagnostics for one source file."""
	source = path.read_text(encoding="utf-8")
	failures: list[str] = []
	for declaration in EXPORT.finditer(source):
		prefix = source[:declaration.start()]
		comments = list(TSDOC.finditer(prefix))
		comment = comments[-1] if comments and not prefix[comments[-1].end():].strip() else None
		body = comment.group("body").strip() if comment else ""
		if not body or body.startswith("*") or "\n" in body:
			line = source.count("\n", 0, declaration.start()) + 1
			display_path = path.relative_to(ROOT) if path.is_relative_to(ROOT) else path
			failures.append(
				f"{display_path}:{line}: {declaration.group('name')} requires concise TSDoc"
			)
	return failures


def main() -> int:
	"""Validate every hand-written export added by Phase 08."""
	failures = [
		failure
		for relative in PHASE08_SOURCES
		for failure in validate_file(ROOT / relative)
	]
	if failures:
		raise SystemExit("\n".join(failures))
	print("Phase 08 exported frontend TSDoc validation passed.")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())
