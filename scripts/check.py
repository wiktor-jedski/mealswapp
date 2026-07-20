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
OPEN_POINTS = ROOT / "docs" / "implementation" / "04_OPEN.md"

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
[SW-REQ-089]
[SW-REQ-090]
[SW-REQ-091]"""

def run(command: list[str], cwd: Path = ROOT) -> None:
	print(f"+ {' '.join(command)}")
	run_env(command, cwd=cwd)


def run_env(command: list[str], cwd: Path = ROOT, capture: bool = False, extra_env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
	env = {
		**dict(os.environ),
		"GOCACHE": str(BACKEND / ".go-cache"),
		"GOMODCACHE": str(BACKEND / ".go-mod-cache"),
		"BUN_TMPDIR": str(FRONTEND / ".bun-tmp"),
		"BUN_INSTALL": str(FRONTEND / ".bun-install"),
	}
	if extra_env:
		env.update(extra_env)
	return subprocess.run(command, cwd=cwd, check=True, env=env, text=True, capture_output=capture)


def running_compose_services() -> set[str]:
	result = run_env(["docker", "compose", "ps", "--status", "running", "--services"], capture=True)
	return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def validate_go_coverage() -> str:
	print("+ go test ./internal/... -count=1 -coverprofile=coverage.out")
	run_env(["go", "test", "./internal/...", "-p", "1", "-count=1", "-coverprofile=coverage.out"], BACKEND, extra_env={"MEALSWAPP_REDIS_URL": "redis://localhost:6379/12"})
	result = run_env(["go", "tool", "cover", "-func=coverage.out"], BACKEND, capture=True)
	print(result.stdout, end="")
	validate_phase07_go_coverage(result.stdout)
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
		validate_documented_frontend_coverage_deviations(coverage_output, all_files)
	validate_phase07_frontend_coverage(coverage_output)
	return coverage_output


def validate_documented_frontend_coverage_deviations(coverage_output: str, all_files: str) -> None:
	# Implements DESIGN-014 MetricsCollector documented coverage-deviation gate.
	open_points = OPEN_POINTS.read_text(encoding="utf-8")
	undocumented = []
	for line in coverage_output.splitlines():
		stripped = line.strip()
		if not stripped.startswith("src/"):
			continue
		columns = [part.strip() for part in stripped.split("|")]
		if len(columns) < 3:
			continue
		path, funcs, lines = columns[:3]
		if funcs != "100.00" or lines != "100.00":
			if path not in open_points:
				undocumented.append(f"{path} ({funcs}% funcs, {lines}% lines)")
	if undocumented:
		raise SystemExit(
			"Frontend coverage below 100% without documented deviations: "
			+ ", ".join(undocumented)
			+ f"; aggregate row: {all_files}"
		)
	print(f"Frontend coverage below 100% with documented deviations: {all_files}")


PHASE07_GO_PACKAGES = {
	"github.com/wiktor-jedski/mealswapp/backend/internal/dailydiet",
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization",
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue",
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker",
}
PHASE07_FRONTEND_SOURCES = {
	"src/lib/api/daily-diet-client.ts",
	"src/lib/api/error-message-mapper.ts",
	"src/lib/api/generated.ts",
	"src/lib/api/optimization-client.ts",
	"src/lib/api/search-client.ts",
	"src/lib/stores/daily-diet.ts",
	"src/lib/stores/optimization.ts",
	"src/lib/stores/search.ts",
	"src/lib/stores/selected-daily-diet.ts",
	"src/lib/units.ts",
}


def validate_phase07_go_coverage(coverage_output: str) -> None:
	# Implements DESIGN-014 MetricsCollector Phase 07 coverage-deviation gate.
	open_points = OPEN_POINTS.read_text(encoding="utf-8")
	below_functions = []
	for line in coverage_output.splitlines():
		match = re.match(r"^.+?/backend/(internal/(?:dailydiet|optimization|queue|worker)/[^:]+):(\d+):\s+(\S+)\s+([0-9.]+%)$", line)
		if match and match.group(4) != "100.0%":
			path, declaration_line, function, coverage = match.groups()
			marker = f"`{path}:{declaration_line} {function}` | `{coverage}`"
			if marker not in open_points:
				below_functions.append(marker)
	if below_functions:
		raise SystemExit(
			"Phase 07 Go coverage has below-100% functions without exact file/line/function evidence: "
			+ ", ".join(below_functions)
		)
	package_totals: dict[str, str] = {}
	for package in sorted(PHASE07_GO_PACKAGES):
		extra_env = {}
		if package.endswith("/queue"):
			extra_env["MEALSWAPP_REDIS_URL"] = "redis://localhost:6379/12"
		elif package.endswith("/worker"):
			extra_env["MEALSWAPP_REDIS_URL"] = "redis://localhost:6379/13"
		result = run_env(["go", "test", package, "-count=1", "-cover"], BACKEND, capture=True, extra_env=extra_env)
		output = f"{result.stdout}\n{result.stderr}"
		match = re.search(r"coverage: ([0-9.]+)% of statements", output)
		if match:
			package_totals[package] = f"{match.group(1)}%"
		print(output, end="")
	missing = sorted(PHASE07_GO_PACKAGES - package_totals.keys())
	if missing:
		raise SystemExit("Phase 07 Go coverage is missing package totals: " + ", ".join(missing))
	below = {package: total for package, total in package_totals.items() if total != "100.0%"}
	if below:
		undocumented = [
			f"{package} ({total})"
			for package, total in sorted(below.items())
			if (package not in open_points and package.rsplit("/", 1)[-1] not in open_points) or total not in open_points
		]
		if undocumented:
			raise SystemExit(
				"Phase 07 Go coverage below 100% without documented measured exceptions: "
				+ ", ".join(undocumented)
			)
		print("Phase 07 Go coverage below 100% with documented measured exceptions: " + ", ".join(f"{package} {total}" for package, total in sorted(below.items())))
	else:
		print("Phase 07 Go coverage passed: all dedicated Phase 07 packages are at 100%.")


def validate_phase07_frontend_coverage(coverage_output: str) -> None:
	# Implements DESIGN-014 MetricsCollector Phase 07 frontend coverage gate.
	rows: dict[str, tuple[str, str]] = {}
	for line in coverage_output.splitlines():
		stripped = line.strip()
		if not stripped.startswith("src/"):
			continue
		columns = [part.strip() for part in stripped.split("|")]
		if len(columns) >= 3:
			rows[columns[0]] = (columns[1], columns[2])
	missing = sorted(PHASE07_FRONTEND_SOURCES - rows.keys())
	if missing:
		raise SystemExit("Phase 07 frontend coverage is missing source rows: " + ", ".join(missing))
	below = {
		path: values
		for path, values in rows.items()
		if path in PHASE07_FRONTEND_SOURCES and values[1] != "100.00"
	}
	if below:
		open_points = OPEN_POINTS.read_text(encoding="utf-8")
		undocumented = [
			f"{path} ({funcs}% funcs, {lines}% lines)"
			for path, (funcs, lines) in sorted(below.items())
			if path not in open_points or f"{funcs}% funcs, {lines}% lines" not in open_points
		]
		if undocumented:
			raise SystemExit(
				"Phase 07 frontend coverage below 100% without documented measured exceptions: "
				+ ", ".join(undocumented)
			)
		print("Phase 07 frontend coverage below 100% with documented measured exceptions: " + ", ".join(f"{path} {lines}% lines" for path, (_, lines) in sorted(below.items())))
	else:
		print("Phase 07 frontend coverage passed: all testable source rows are at 100%.")


def validate_go_format() -> None:
	# Implements DESIGN-014 MetricsCollector backend formatting gate.
	cache_dirs = {".go-cache", ".go-mod-cache"}
	go_files = sorted(
		str(path)
		for path in BACKEND.rglob("*.go")
		if not cache_dirs.intersection(path.relative_to(BACKEND).parts)
	)
	result = run_env(["gofmt", "-l", *go_files], BACKEND, capture=True)
	if result.stdout.strip():
		raise SystemExit("Go formatting check failed:\n" + result.stdout)


def validate_phase07_backend_workflows() -> None:
	# Implements DESIGN-014 MetricsCollector Phase 07 focused backend aggregate gate.
	def run_phase07_test(command: list[str], redis_db: int | None = None) -> None:
		extra_env = {} if redis_db is None else {"MEALSWAPP_REDIS_URL": f"redis://localhost:6379/{redis_db}"}
		print(f"+ {' '.join(command)}" + (f"  # Redis DB {redis_db}" if redis_db is not None else ""))
		run_env(command, BACKEND, extra_env=extra_env)

	run_phase07_test(["go", "test", "./internal/migrations", "-run", "^TestRun", "-count=1"])
	run_phase07_test(["go", "test", "./internal/repository", "-run", "^TestPostgresSavedDiet", "-count=1"])
	run_phase07_test(["go", "test", "./internal/dailydiet", "-count=1"])
	run_phase07_test(["go", "test", "./internal/optimization", "-run", "^(TestBuild|TestGenerate|TestLPSolver|TestValidate|TestSafe)", "-count=1"])
	run_phase07_test(["go", "test", "./internal/queue", "-run", "^TestJobQueue", "-count=1"], 15)
	run_phase07_test(["go", "test", "./internal/worker", "-run", "^(TestRun|TestRedis|TestOptimization)", "-count=1"], 15)
	run_phase07_test(["go", "test", "./internal/httpapi", "-run", "^(TestProfileControllerDailyDiet|TestOptimizationHTTP)", "-count=1"], 14)
	run_phase07_test(["go", "test", "./internal/app", "-run", "^(TestDailyDietProductionAPIWithLivePostgres|TestTask206)", "-count=1"], 14)


def validate_phase07_frontend_workflows() -> None:
	# Implements DESIGN-014 MetricsCollector Phase 07 Daily Diet and accessibility gate.
	run([
		"bun", "run", "test:e2e", "--",
		"tests/daily-diet-workflow.spec.ts",
		"tests/phase07-browser-acceptance.spec.ts",
	], FRONTEND)


def validate_phase07_capacity_tests() -> None:
	# Implements DESIGN-014 MetricsCollector Phase 07 capacity regression gate.
	run(["python3", "-m", "unittest", "scripts/test_verify_optimization_capacity.py"])


def validate_start_dev_process_tests() -> None:
	# Implements DESIGN-010 RouteHandler local development process lifecycle gate.
	run(["python3", "-m", "unittest", "scripts/test_start_dev.py"])


def validate_stripe_webhook_tests() -> None:
	# Implements DESIGN-007 SubscriptionController Stripe webhook aggregate gate.
	run([
		"go", "test", "./internal/subscription",
		"-run", "TestStripeWebhookService",
		"-count=1",
	], BACKEND)
	run([
		"go", "test", "./internal/httpapi",
		"-run", "TestStripeWebhookHandler",
		"-count=1",
	], BACKEND)


def validate_phase0601_backend_auth_billing_smoke_tests() -> None:
	# Implements DESIGN-014 MetricsCollector auth and billing compatibility aggregate gate.
	run([
		"go", "test",
		"./internal/auth",
		"./internal/httpapi",
		"./internal/subscription",
		"./internal/entitlement",
		"-count=1",
	], BACKEND)


def validate_phase0601_frontend_auth_workflows() -> None:
	# Implements DESIGN-014 MetricsCollector focused DESIGN-018 browser workflow aggregate gate.
		run([
		"bun", "run", "test:e2e", "--",
		"tests/auth-session.spec.ts",
		"tests/subscription-billing.spec.ts",
		"tests/search-workflow.spec.ts",
	], FRONTEND)


def validate_frontend_e2e() -> None:
	# Implements DESIGN-014 MetricsCollector complete Playwright and axe aggregate gate.
	run(["bun", "run", "test:e2e"], FRONTEND)


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
	"scripts/generate-api-types.py",
	"scripts/start-services.sh", "scripts/validate-traceability.py",
	"scripts/validate-phase07-go-doc.py",
	"scripts/validate-task-list.py", "scripts/verify-frontend.py",
	"scripts/verify-local-stack.py", "scripts/verify-phase02-uat.py", "scripts/verify-phase03-uat.py",
	"scripts/verify-optimization-capacity.py", "scripts/test_verify_optimization_capacity.py",
	"scripts/dev-processes.sh", "scripts/start-dev.sh", "scripts/test_start_dev.py",
	"scripts/verify-clp-worker-image.sh",
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
	run(["python3", "scripts/validate-phase07-go-doc.py"])
	# Implements DESIGN-010 RouteHandler contract and backend quality gates.
	run(["npx", "--no-install", "redocly", "lint", "api/openapi.yaml"])
	validate_phase07_capacity_tests()
	validate_start_dev_process_tests()
	run(["go", "vet", "./..."], BACKEND)
	run(["go", "run", "golang.org/x/vuln/cmd/govulncheck@v1.3.0", "./..."], BACKEND)
	validate_stripe_webhook_tests()
	validate_phase0601_backend_auth_billing_smoke_tests()
	initially_running_services = running_compose_services()
	run(["python3", "scripts/verify-local-stack.py", "--keep-services"])
	run(["python3", "scripts/verify-phase02-uat.py", "--keep-services"])
	run(["python3", "scripts/verify-phase03-uat.py", "--keep-services"])
	validate_phase07_backend_workflows()
	run(["python3", "scripts/verify-frontend.py", "--screenshot-stem", screenshot_stem])
	validate_go_format()
	try:
		# Keep package-parallel Redis integration tests isolated from the local stack and
		# from the focused gates above; this is test isolation, not a product override.
		run_env(["go", "test", "./...", "-p", "1", "-count=1"], BACKEND, extra_env={"MEALSWAPP_REDIS_URL": "redis://localhost:6379/10"})
		run_env(["go", "test", "-race", "./...", "-p", "1", "-count=1"], BACKEND, extra_env={"MEALSWAPP_REDIS_URL": "redis://localhost:6379/11"})
		go_coverage_stdout = validate_go_coverage()
	finally:
		started_services = ({"postgres", "redis"} & running_compose_services()) - initially_running_services
		if started_services:
			run(["docker", "compose", "stop", *sorted(started_services)])
	run(["bun", "run", "check:api-types"], FRONTEND)
	run(["bun", "run", "typecheck"], FRONTEND)
	run(["bun", "run", "build"], FRONTEND)
	run(["bun", "test"], FRONTEND)
	bun_coverage_stdout = validate_frontend_coverage()
	validate_phase0601_frontend_auth_workflows()
	validate_phase07_frontend_workflows()
	validate_frontend_e2e()

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
