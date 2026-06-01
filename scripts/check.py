#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector bootstrap quality gate.

import re
import subprocess
import sys
import os
import argparse
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / "backend"
FRONTEND = ROOT / "frontend"

reqs = """[SW-REQ-001]
[SW-REQ-002]
[SW-REQ-003]
[SW-REQ-004]
[SW-REQ-005]
[SW-REQ-006]
[SW-REQ-007]
[SW-REQ-008]
[SW-REQ-009]
[SW-REQ-010]
[SW-REQ-011]
[SW-REQ-012]
[SW-REQ-013]
[SW-REQ-014]
[SW-REQ-015]
[SW-REQ-016]
[SW-REQ-017]
[SW-REQ-018]
[SW-REQ-019]
[SW-REQ-020]
[SW-REQ-021]
[SW-REQ-022]
[SW-REQ-023]
[SW-REQ-024]
[SW-REQ-025]
[SW-REQ-026]
[SW-REQ-027]
[SW-REQ-028]
[SW-REQ-029]
[SW-REQ-030]
[SW-REQ-031]
[SW-REQ-032]
[SW-REQ-033]
[SW-REQ-034]
[SW-REQ-035]
[SW-REQ-036]
[SW-REQ-037]
[SW-REQ-038]
[SW-REQ-039]
[SW-REQ-040]
[SW-REQ-041]
[SW-REQ-042]
[SW-REQ-043]
[SW-REQ-044]
[SW-REQ-045]
[SW-REQ-046]
[SW-REQ-047]
[SW-REQ-048]
[SW-REQ-049]
[SW-REQ-050]
[SW-REQ-051]
[SW-REQ-052]
[SW-REQ-053]
[SW-REQ-054]
[SW-REQ-055]
[SW-REQ-056]
[SW-REQ-057]
[SW-REQ-058]
[SW-REQ-059]
[SW-REQ-060]
[SW-REQ-061]
[SW-REQ-062]
[SW-REQ-063]
[SW-REQ-064]
[SW-REQ-065]
[SW-REQ-066]
[SW-REQ-067]
[SW-REQ-068]
[SW-REQ-069]
[SW-REQ-070]
[SW-REQ-071]
[SW-REQ-072]
[SW-REQ-073]
[SW-REQ-074]
[SW-REQ-075]
[SW-REQ-076]
[SW-REQ-077]
[SW-REQ-078]
[SW-REQ-079]
[SW-REQ-080]
[SW-REQ-081]
[SW-REQ-082]
[SW-REQ-083]
[SW-REQ-084]
[SW-REQ-085]
[SW-REQ-086]
[SW-REQ-087]
[SW-REQ-088]
[SW-REQ-089]"""

def run(command: list[str], cwd: Path = ROOT) -> None:
	print(f"+ {' '.join(command)}")
	run_env(command, cwd=cwd)


def run_env(command: list[str], cwd: Path = ROOT, capture: bool = False) -> subprocess.CompletedProcess[str]:
	env = {
		**dict(os.environ),
		"GOCACHE": str(BACKEND / ".go-cache"),
		"GOMODCACHE": str(BACKEND / ".go-mod-cache"),
		"BUN_TMPDIR": str(FRONTEND / ".bun-tmp"),
		"BUN_INSTALL": str(FRONTEND / ".bun-install"),
	}
	return subprocess.run(command, cwd=cwd, check=True, env=env, text=True, capture_output=capture)


def running_compose_services() -> set[str]:
	result = run_env(["docker", "compose", "ps", "--status", "running", "--services"], capture=True)
	return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def validate_go_coverage() -> str:
	print("+ go test ./internal/... -count=1 -coverprofile=coverage.out")
	run_env(["go", "test", "./internal/...", "-count=1", "-coverprofile=coverage.out"], BACKEND)
	result = run_env(["go", "tool", "cover", "-func=coverage.out"], BACKEND, capture=True)
	print(result.stdout, end="")
	total_line = next(line for line in result.stdout.splitlines() if line.startswith("total:"))
	if not total_line.rstrip().endswith("100.0%"):
		raise SystemExit(f"Go internal coverage below 100%: {total_line}")
	return result.stdout


def validate_frontend_coverage() -> str:
	print("+ bun test --coverage")
	result = run_env(["bun", "test", "--coverage"], FRONTEND, capture=True)
	print(result.stdout, end="")
	print(result.stderr, end="")
	coverage_output = f"{result.stdout}\n{result.stderr}"
	all_files = next(line for line in coverage_output.splitlines() if line.strip().startswith("All files"))
	columns = [part.strip() for part in all_files.split("|")]
	if len(columns) < 3 or columns[1] != "100.00" or columns[2] != "100.00":
		raise SystemExit(f"Frontend coverage below 100%: {all_files}")
	return coverage_output


def validate_requirements() -> tuple[int, int]:
	text = (ROOT / "docs/architecture/01_SOFT_ARCH_DESIGN.md").read_text()
	missing = []
	total = 0
	checked = 0
	for req in reqs.split("\n"):
		total += 1
		plain_req = req.strip("[]")
		if req in text or plain_req in text:
			checked += 1
		else:
			missing.append(req)
	if missing:
		for req in missing:
			print(f"{req} MISSING")
		raise SystemExit(1)
	return checked, total


TRACEABLE_SUFFIXES = {".go", ".js", ".ts", ".svelte", ".css", ".html", ".yaml", ".yml", ".sql", ".sh", ".py"}
TRACEABLE_ROOTS = {".github", "api", "backend", "database", "frontend"}
TRACEABLE_FILES = {
	"docker-compose.yml", "scripts/check.py", "scripts/generate_report.py",
	"scripts/start-services.sh", "scripts/validate-traceability.py",
	"scripts/validate-task-list.py", "scripts/verify-frontend.py",
	"scripts/verify-local-stack.py",
}
SKIP_TRACEABILITY_NAMES = {"bun.lock", "go.mod", "go.sum"}


def project_files() -> list[Path]:
	result = subprocess.run(
		["git", "ls-files", "--cached", "--others", "--exclude-standard"],
		cwd=ROOT, check=True, text=True, capture_output=True,
	)
	return [ROOT / line for line in result.stdout.splitlines() if line]


def is_traceable_source(path: Path) -> bool:
	relative = path.relative_to(ROOT).as_posix()
	if path.name in SKIP_TRACEABILITY_NAMES or relative.endswith("-trace.md"):
		return False
	if relative in TRACEABLE_FILES:
		return True
	first_part = relative.split("/", 1)[0]
	if first_part not in TRACEABLE_ROOTS:
		return False
	return path.suffix in TRACEABLE_SUFFIXES


DESIGN_STATIC_ASPECT_RE = re.compile(r"Implements\s+.*?DESIGN-\d{3}[^\w\d]*(?P<aspect>[A-Za-z]\w*)")
DESIGN_ID_RE = re.compile(r"DESIGN-\d{3}")


def parse_design_docs() -> dict[str, list[str]]:
	design_dir = ROOT / "docs" / "design"
	design_aspects: dict[str, list[str]] = {}
	for path in sorted(design_dir.glob("DESIGN-*.md")):
		text = path.read_text()
		match = re.search(r"Static aspects covered:\s*(.+?)(?:\n|$)", text, re.DOTALL)
		if match:
			aspects_str = match.group(1).strip()
			raw_aspects = aspects_str.split(",")
			cleaned = []
			for a in raw_aspects:
				cleaned_a = a.strip().lstrip("*").rstrip(".").strip()
				if cleaned_a:
					cleaned.append(cleaned_a)
			design_id = path.stem
			design_aspects[design_id] = cleaned
	return design_aspects


def scan_implemented_aspects() -> set[str]:
	implemented: set[str] = set()
	for path in project_files():
		if not path.is_file():
			continue
		try:
			text = path.read_text(encoding="utf-8")
		except UnicodeDecodeError:
			continue
		if not is_traceable_source(path):
			continue
		for match in DESIGN_STATIC_ASPECT_RE.finditer(text):
			aspect = match.group("aspect")
			implemented.add(aspect)
	return implemented


def validate_design_coverage() -> tuple[dict[str, list[str]], dict[str, list[str]], int, int]:
	design_aspects = parse_design_docs()
	implemented = scan_implemented_aspects()

	implemented_by_design: dict[str, list[str]] = {}
	missing_by_design: dict[str, list[str]] = {}
	total_aspects = 0

	for design_id, aspects in sorted(design_aspects.items()):
		implemented_list = [a for a in aspects if a in implemented]
		missing_list = [a for a in aspects if a not in implemented]
		implemented_by_design[design_id] = implemented_list
		missing_by_design[design_id] = missing_list
		total_aspects += len(aspects)

	checked = sum(len(v) for v in implemented_by_design.values())
	return implemented_by_design, missing_by_design, checked, total_aspects


def main() -> int:
	parser = argparse.ArgumentParser(description="Mealswapp aggregate quality gate script.")
	parser.add_argument("--output", help="Path to write the HTML coverage and quality gate report.")
	args = parser.parse_args()
	screenshot_stem = Path(args.output).stem if args.output else "frontend-verification"

	checked_reqs, total_reqs = validate_requirements()
	run(["python3", "scripts/validate-traceability.py"])
	run(["python3", "scripts/validate-task-list.py"])
	initially_running_services = running_compose_services()
	run(["python3", "scripts/verify-local-stack.py", "--keep-services"])
	run(["python3", "scripts/verify-frontend.py", "--screenshot-stem", screenshot_stem])
	run(["go", "fmt", "./..."], BACKEND)
	try:
		run(["go", "test", "./...", "-count=1"], BACKEND)
		go_coverage_stdout = validate_go_coverage()
	finally:
		started_services = {"postgres", "redis"} - initially_running_services
		if started_services:
			run(["docker", "compose", "stop", *sorted(started_services)])
	run(["bun", "run", "build"], FRONTEND)
	run(["bun", "test"], FRONTEND)
	bun_coverage_stdout = validate_frontend_coverage()

	design_implemented, design_missing, design_checked, design_total = validate_design_coverage()

	if args.output:
		import sys
		sys.path.insert(0, str(ROOT / "scripts"))
		from generate_report import build_html_report
		from check import parse_design_docs
		build_html_report(
			go_raw=go_coverage_stdout,
			bun_raw=bun_coverage_stdout,
			reqs_checked=checked_reqs,
			reqs_total=total_reqs,
			design_implemented=design_implemented,
			design_missing=design_missing,
			design_checked=design_checked,
			design_total=design_total,
			design_aspects=parse_design_docs(),
			output_path=args.output,
			screenshot_stem=screenshot_stem
		)
	return 0


if __name__ == "__main__":
    sys.exit(main())
