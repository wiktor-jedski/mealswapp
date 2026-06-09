#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector traceability validation gate.

import json
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DESIGN_REF_RE = re.compile(r"Implements\s+(?P<refs>.*?DESIGN-\d{3}.*)")
DESIGN_ID_RE = re.compile(r"DESIGN-\d{3}")
GO_PACKAGE_DECLARATION_RE = re.compile(r"^(?:type|const|var|func)\s")
GO_TYPE_DECLARATION_RE = re.compile(r"^type\s+(?P<name>[A-Za-z_]\w*)\b")
GO_FUNC_DECLARATION_RE = re.compile(
	r"^func(?:\s+\([^)]*\))?\s+(?P<name>[A-Za-z_]\w*)\s*\("
)
TRACEABLE_SUFFIXES = {
	".go",
	".js",
	".ts",
	".svelte",
	".css",
	".html",
	".yaml",
	".yml",
	".sql",
	".sh",
	".py",
}
TRACEABLE_ROOTS = {
	".github",
	"api",
	"backend",
	"database",
	"frontend",
}
TRACEABLE_FILES = {
	"docker-compose.yml",
	"scripts/check.py",
	"scripts/generate-api-types.py",
	"scripts/generate_report.py",
	"scripts/start-services.sh",
	"scripts/validate-task-list.py",
	"scripts/validate-traceability.py",
	"scripts/verify-frontend.py",
	"scripts/verify-local-stack.py",
	"scripts/verify-phase02-uat.py",
	"scripts/verify-phase03-uat.py",
}
JSON_ROOTS_REQUIRING_SIDECARS = {
	"frontend",
}
SKIP_TRACEABILITY_NAMES = {
	"bun.lock",
	"go.mod",
	"go.sum",
}


def project_files() -> list[Path]:
	result = subprocess.run(
		["git", "ls-files", "--cached", "--others", "--exclude-standard"],
		cwd=ROOT,
		check=True,
		text=True,
		capture_output=True,
	)
	return [ROOT / line for line in result.stdout.splitlines() if line]


def line_number(text: str, index: int) -> int:
	return text.count("\n", 0, index) + 1


def rel(path: Path) -> str:
	return path.relative_to(ROOT).as_posix()


def existing_design_docs() -> set[str]:
	return {path.stem for path in (ROOT / "docs" / "design").glob("DESIGN-*.md")}


def is_traceable_source(path: Path) -> bool:
	relative = rel(path)
	if path.name in SKIP_TRACEABILITY_NAMES or relative.endswith("-trace.md"):
		return False
	if relative in TRACEABLE_FILES:
		return True
	first_part = Path(relative).parts[0]
	if first_part not in TRACEABLE_ROOTS:
		return False
	return path.suffix in TRACEABLE_SUFFIXES


def validate_trace_comment(path: Path, text: str, designs: set[str]) -> list[str]:
	errors = []
	matches = list(DESIGN_REF_RE.finditer(text))
	if not matches:
		errors.append(f"{rel(path)}:1: missing `Implements DESIGN-*` traceability comment")
		return errors

	for match in matches:
		line = line_number(text, match.start())
		comment_tail = match.group("refs").strip()
		ids = DESIGN_ID_RE.findall(comment_tail)
		for design_id in ids:
			if design_id not in designs:
				errors.append(f"{rel(path)}:{line}: references missing docs/design/{design_id}.md")
		static_aspect = DESIGN_ID_RE.sub("", comment_tail)
		static_aspect = re.sub(r"[,.;:()\\[\\]`*/_-]+", " ", static_aspect).strip()
		if not static_aspect:
			errors.append(f"{rel(path)}:{line}: traceability comment needs a static aspect after the design ID")
	return errors


def validate_go_declaration_traces(path: Path, text: str) -> list[str]:
	relative = rel(path)
	if not relative.startswith("backend/") or path.suffix != ".go" or path.name.endswith("_test.go"):
		return []

	errors = []
	lines = text.splitlines()
	for index, line in enumerate(lines):
		if not GO_PACKAGE_DECLARATION_RE.match(line):
			continue
		previous_index = index - 1
		while previous_index >= 0 and lines[previous_index].startswith("//go:"):
			previous_index -= 1
		if previous_index >= 0 and lines[previous_index] == "//" and index > 0 and lines[index - 1].startswith("//go:"):
			previous_index -= 1
		previous = lines[previous_index] if previous_index >= 0 else ""
		if not DESIGN_REF_RE.search(previous):
			errors.append(
				f"{relative}:{index + 1}: package-level Go declaration needs an adjacent "
				"`Implements DESIGN-*` traceability comment"
			)
	return errors


def validate_go_doc_comments(path: Path, text: str) -> list[str]:
	relative = rel(path)
	if not relative.startswith("backend/") or path.suffix != ".go" or path.name.endswith("_test.go"):
		return []

	errors = []
	lines = text.splitlines()
	for index, line in enumerate(lines):
		match = GO_TYPE_DECLARATION_RE.match(line) or GO_FUNC_DECLARATION_RE.match(line)
		if not match:
			continue
		name = match.group("name")
		previous_index = index - 1
		while previous_index >= 0 and lines[previous_index].startswith("//"):
			comment = lines[previous_index].removeprefix("//").strip()
			if comment.startswith(name):
				break
			previous_index -= 1
		else:
			errors.append(
				f"{relative}:{index + 1}: Go declaration `{name}` needs a doc comment "
				f"starting with `{name}`"
			)
	return errors


def validate_json_sidecar(path: Path, text: str, designs: set[str]) -> list[str]:
	errors = []
	relative = rel(path)
	try:
		json.loads(text)
	except json.JSONDecodeError as exc:
		errors.append(f"{relative}:{exc.lineno}: invalid JSON: {exc.msg}")
		return errors

	if Path(relative).parts[0] not in JSON_ROOTS_REQUIRING_SIDECARS:
		return errors

	sidecar = path.with_name(f"{path.name}-trace.md")
	if not sidecar.exists():
		errors.append(f"{relative}:1: missing JSON traceability sidecar `{rel(sidecar)}`")
		return errors

	sidecar_text = sidecar.read_text(encoding="utf-8")
	ids = DESIGN_ID_RE.findall(sidecar_text)
	if not ids:
		errors.append(f"{rel(sidecar)}:1: sidecar must list at least one DESIGN-* source")
	for design_id in ids:
		if design_id not in designs:
			errors.append(f"{rel(sidecar)}:1: references missing docs/design/{design_id}.md")
	return errors


def main() -> int:
	designs = existing_design_docs()
	errors: list[str] = []

	for path in project_files():
		if not path.is_file():
			continue
		try:
			text = path.read_text(encoding="utf-8")
		except UnicodeDecodeError:
			continue
		if is_traceable_source(path):
			errors.extend(validate_trace_comment(path, text, designs))
			errors.extend(validate_go_declaration_traces(path, text))
			errors.extend(validate_go_doc_comments(path, text))
		if path.suffix == ".json":
			errors.extend(validate_json_sidecar(path, text, designs))

	if errors:
		print("Traceability validation failed:")
		for error in errors:
			print(f"- {error}")
		return 1

	print("Traceability validation passed.")
	return 0


if __name__ == "__main__":
	sys.exit(main())
