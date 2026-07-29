#!/usr/bin/env python3
"""Run Task 283 manual-catalog acceptance on the disposable real stack."""

# Implements DESIGN-005 RepositoryInterfaces and DESIGN-009 AdminController acceptance.
from __future__ import annotations

import argparse
import importlib.util
import json
import os
import re
import sys
import threading
import time
from pathlib import Path
from urllib.parse import unquote, urlparse

ROOT = Path(__file__).resolve().parents[1]
REAL_STACK_PATH = ROOT / "scripts/run-real-stack-e2e.py"
spec = importlib.util.spec_from_file_location("task283_real_stack", REAL_STACK_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("real-stack harness could not be loaded")
real_stack = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = real_stack
spec.loader.exec_module(real_stack)

TASK_REQUIREMENTS = {"SW-REQ-019", "SW-REQ-032", "SW-REQ-033", "SW-REQ-056", "SW-REQ-057", "SW-REQ-090"}
REQUIREMENT_CRITERIA = {
    requirement: tuple(
        criterion["id"]
        for scenario in json.loads((ROOT / "docs/testing/phase08/acceptance-manifest.json").read_text())["scenarios"]
        if scenario["requirement"] == requirement
        for criterion in scenario["criteria"]
    )
    for requirement in TASK_REQUIREMENTS
}
CRITERIA = tuple(item for requirement in sorted(REQUIREMENT_CRITERIA) for item in REQUIREMENT_CRITERIA[requirement])
ROOTS = {
    "P08-SWR019": "ROOT-T283-FILTER-CROSS-INSTANCE",
    "P08-SWR032": "ROOT-T283-METRIC-NORMALIZATION",
    "P08-SWR033": "ROOT-T283-DISCOVERY-PARTITION",
    "P08-SWR056": "ROOT-T283-MANUAL-CATALOG",
    "P08-SWR057": "ROOT-T283-CLASSIFICATION-LIFECYCLE",
    "P08-SWR090": "ROOT-T283-MICRONUTRIENT-VALIDATION",
}
INFRASTRUCTURE_ROOT = "ROOT-T283-ACCEPTANCE-INFRASTRUCTURE"
SYNCHRONIZED_ROOTS = tuple(ROOTS.values()) + (INFRASTRUCTURE_ROOT,)
RUN_ID_RE = re.compile(r"[a-z0-9][a-z0-9-]{0,63}")
REQUEST_ID_RE = re.compile(r"[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}")
BACKEND_KEYS = {"http_status", "mutation_count", "audit_count", "row_count", "owner_state", "cache_generation", "worker_state", "provider_state", "log_sink_state", "metric_basis", "export_record_count", "request_correlation", "rollback_state"}
BACKEND_RE = re.compile(r"[a-z][a-z0-9_]{0,63}(?:=[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127})?")
GENERATION_FIELDS = (
    ("generationBefore", "generationAfter", "generationDelta"),
    ("failedGenerationBefore", "failedGenerationAfter", "failedGenerationDelta"),
)
SNAPSHOT_ID_RE = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}", re.I)


def parse_run_owned_database(url: str) -> tuple[str, str]:
    """Strictly parse a loopback database URL for read-only proof and harness tests."""
    parsed = urlparse(url)
    if parsed.scheme not in {"postgres", "postgresql"} or parsed.hostname not in {"127.0.0.1", "localhost"}:
        raise ValueError("database must be loopback postgres")
    if parsed.username or parsed.password or parsed.port not in {None, 5432} or parsed.params or parsed.query or parsed.fragment:
        raise ValueError("database URL contains forbidden authority or suffix")
    database = unquote(parsed.path.lstrip("/"))
    if not re.fullmatch(r"mealswapp_e2e_[a-z0-9-]{1,52}_test", database) or "/" in database or "\\" in database:
        raise ValueError("database name is not run-owned")
    return parsed.hostname, database


def read_only_sql(sql: str) -> str:
    """Validate the parameterized proof statement before execution by the harness."""
    normalized = sql.strip().lower()
    if ";" in normalized or not normalized.startswith(("select ", "with ", "show ", "explain ")):
        raise ValueError("proof SQL must be one read-only statement")
    if re.search(r"\b(insert|update|delete|drop|alter|create|truncate|grant|revoke|copy)\b", normalized):
        raise ValueError("proof SQL contains mutation or DDL")
    return sql


def read_only_psql(target, database: str, sql: str, parameters: tuple[str, ...] = ()) -> str:
    """Execute a parameterized proof inside a PostgreSQL read-only transaction."""
    statement = read_only_sql(sql)
    if any(";" in value or "'" in value for value in parameters):
        raise ValueError("proof parameters contain unsafe delimiters")
    variables = []
    for index, _ in enumerate(parameters):
        variables.extend((f"--set=p{index}={parameters[index]}",))
        statement = statement.replace("%s", f":'p{index}'", 1)
    if "%s" in statement:
        raise ValueError("proof SQL has more placeholders than parameters")
    wrapped = f"BEGIN; SET TRANSACTION READ ONLY; {statement}; COMMIT;"
    try:
        return real_stack.run_command([*target.command(database), *variables, "-Atqf", "-"], env=target.environment(), input_text=wrapped).stdout.strip()
    except __import__("subprocess").CalledProcessError as error:
        detail = str(error.stderr or error.stdout or "sql proof command failed").replace("\n", " ")[:240]
        raise RuntimeError(f"read-only proof query failed: {detail}") from error


def compare_expected(expected: dict[str, object], actual: dict[str, object], prefix: str = "") -> list[str]:
    """Return bounded field paths whose operation-scoped actual values violate expectations."""
    failures: list[str] = []
    for key, wanted in expected.items():
        path = f"{prefix}.{key}" if prefix else key
        if key not in actual:
            failures.append(f"{path}:missing")
            continue
        observed = actual[key]
        if isinstance(wanted, dict):
            if not isinstance(observed, dict):
                failures.append(f"{path}:type")
            else:
                failures.extend(compare_expected(wanted, observed, path))
        elif isinstance(wanted, (int, float)) and isinstance(observed, (int, float)):
            if abs(float(wanted) - float(observed)) > 0.000001:
                failures.append(f"{path}:value")
        elif wanted != observed:
            failures.append(f"{path}:value")
    return failures


def redis_generation_actual(
    expected: dict[str, object],
    snapshot_ids: object,
    observation_directory: Path,
) -> dict[str, object]:
    """Load independently queried, immutable Redis observations for one operation."""
    required = {
        field
        for before, after, _delta in GENERATION_FIELDS
        for field in (before, after)
        if field in expected
    }
    if not required:
        if snapshot_ids is not None:
            raise ValueError("operation has unexpected Redis generation snapshots")
        return {}
    if not isinstance(snapshot_ids, dict) or set(snapshot_ids) != required:
        raise ValueError("operation Redis generation snapshots are incomplete")
    if len(set(snapshot_ids.values())) != len(snapshot_ids):
        raise ValueError("operation Redis generation snapshot identities are not unique")
    actual: dict[str, object] = {}
    for field in sorted(required):
        snapshot_id = snapshot_ids[field]
        if not isinstance(snapshot_id, str) or not SNAPSHOT_ID_RE.fullmatch(snapshot_id):
            raise ValueError("operation Redis generation snapshot identity is invalid")
        path = observation_directory / f"{snapshot_id}.json"
        try:
            observation = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as error:
            raise ValueError(f"operation Redis generation observation is unavailable: {field}") from error
        if (
            not isinstance(observation, dict)
            or observation.get("schema") != "mealswapp.task283-redis-observation.v1"
            or observation.get("id") != snapshot_id
            or observation.get("key") != "classification:cache-generation:v1"
            or not isinstance(observation.get("value"), str)
            or not observation["value"].isdigit()
        ):
            raise ValueError(f"operation Redis generation observation is invalid: {field}")
        actual[field] = observation["value"]
    for before, after, delta in GENERATION_FIELDS:
        if before in actual and after in actual:
            actual[delta] = int(actual[after]) - int(actual[before])
    return actual


class Task283Harness(real_stack.Harness):
    """Own the Task 283 database, Redis, APIs, browser suite, evidence, and cleanup."""

    second_api_port: int | None = None

    def application_environment(self, database_url: str, redis_url: str, api_port: int, frontend_port: int) -> dict[str, str]:
        environment = super().application_environment(database_url, redis_url, api_port, frontend_port)
        environment.update({
            "MEALSWAPP_TASK283_REAL_E2E": "1",
            "MEALSWAPP_REAL_STACK_MANAGED": "1",
            "MEALSWAPP_TASK294_CORRUPT_MANUAL_ITEM_RESPONSE_ONCE": "1",
            "MEALSWAPP_TASK294_REAL_E2E": "1",
        })
        return environment

    def start_application_stack(self, api_binary, database_url, redis_url, reservations):
        """Start the inherited frontend/primary API and api-2 on shared database/Redis."""
        env, api_port, frontend_port, api, frontend = super().start_application_stack(api_binary, database_url, redis_url, reservations)
        second = real_stack.reserve_ports(1)[0]
        self.second_api_port = second.port
        second_env = self.application_environment(database_url, redis_url, second.port, frontend_port)
        second_process = self.start_process("api-2", [str(api_binary)], ROOT / "backend", second_env, "api-2.raw.log", second)
        real_stack.wait_http(f"http://127.0.0.1:{second.port}/health", second_process, self.timeout)
        self.events.append("second_api_ready")
        env["MEALSWAPP_TASK283_SECOND_API_URL"] = f"http://127.0.0.1:{second.port}"
        return env, api_port, frontend_port, api, frontend

    def execute(self) -> None:
        """Run migrations, two shared-target APIs, Playwright, and exact proof collection."""
        reservations = []
        try:
            real_stack.create_database(self.target, self.database, self.comment); self.events.append("database_created")
            redis_port = self.start_redis(); reservations = real_stack.reserve_ports(2)
            database_url = self.target.database_url(self.database); redis_url = f"redis://127.0.0.1:{redis_port}/0"
            env = self.application_environment(database_url, redis_url, reservations[0].port, reservations[1].port)
            real_stack.run_command(["go", "run", "./cmd/migrate", "up"], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            assert self.raw_dir is not None
            api_binary = self.raw_dir / "mealswapp-api"
            bootstrap_binary = self.raw_dir / "admin-bootstrap"
            real_stack.run_command(["go", "build", "-o", str(api_binary), "./cmd/api"], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            real_stack.run_command(["go", "build", "-o", str(bootstrap_binary), "./cmd/admin-bootstrap"], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            env, api_port, frontend_port, _api, frontend = self.start_application_stack(api_binary, database_url, redis_url, reservations)
            user, ids = real_stack.register_fixture(f"http://127.0.0.1:{api_port}", self.run_id); self.request_ids.extend(ids)
            bootstrap = real_stack.run_command([str(bootstrap_binary), "--environment", "development", "--email", user["email"]], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            if f"user_id={user['user_id']}" not in bootstrap.stdout or "actor=operator" not in bootstrap.stdout:
                raise RuntimeError("administrator bootstrap returned an unexpected target")
            self.events.append("administrator_bootstrapped")
            real_stack.psql(self.target, "INSERT INTO entitlements (user_id,tier,status,search_limit_per_24h,allowed_modes,expires_at) VALUES ('%s'::uuid,'trial','active',100,ARRAY['catalog','substitution','daily_diet','daily_diet_alternative'],now()+interval '1 day')" % user["user_id"], database=self.database)
            capability = self.raw_dir / "task283-harness-capability.json"
            nonce = __import__("secrets").token_hex(24)
            capability.write_text(json.dumps({"schema":"mealswapp.task283-harness-capability.v1","runId":self.run_id,"nonce":nonce,"baseURL":f"http://127.0.0.1:{frontend_port}","evidenceRoot":str(self.artifacts / "acceptance"),"frontendProcess":{"pid":frontend.pid,"startToken":real_stack.validate_process_start_token(real_stack.process_start_token(frontend.pid))}}, sort_keys=True)+"\n")
            capability.chmod(0o600)
            evidence = self.artifacts / "acceptance"; evidence.mkdir()
            redis_requests = self.raw_dir / "task283-redis-observation-requests"
            redis_observations = evidence / "backend/redis-observations"
            redis_requests.mkdir()
            redis_observations.mkdir(parents=True)
            self.redis_observation_directory = redis_observations
            observer_stop = threading.Event()
            observer = threading.Thread(
                target=self.observe_redis_generation_requests,
                args=(redis_requests, redis_observations, observer_stop),
                daemon=True,
            )
            observer.start()
            playwright_env = {**env,"MEALSWAPP_TASK283_REAL_E2E":"1","MEALSWAPP_REAL_STACK_MANAGED":"1","MEALSWAPP_REAL_STACK_BASE_URL":f"http://127.0.0.1:{frontend_port}","MEALSWAPP_TASK283_CAPABILITY_FILE":str(capability),"MEALSWAPP_TASK283_CAPABILITY_NONCE":nonce,"MEALSWAPP_TASK283_SECOND_API_URL":f"http://127.0.0.1:{self.second_api_port}","MEALSWAPP_TASK283_REDIS_CONTAINER":self.container,"MEALSWAPP_TASK283_AUTH_STATE_DIR":str(self.raw_dir / "task283-auth-state"),"MEALSWAPP_E2E_EMAIL":user["email"],"MEALSWAPP_E2E_PASSWORD":user["password"],"PHASE08_ACCEPTANCE_RESULT_DIR":str(evidence),"MEALSWAPP_PHASE08_RESULT_FILE":"browser.json","MEALSWAPP_PHASE08_CRITERIA":",".join(CRITERIA),"MEALSWAPP_PHASE08_EXPECTED_PROJECTS":"real-stack-desktop-chromium,real-stack-mobile-chromium","MEALSWAPP_PHASE08_INFRASTRUCTURE_ROOT":INFRASTRUCTURE_ROOT,"MEALSWAPP_PHASE08_SYNCHRONIZED_ROOTS":",".join(SYNCHRONIZED_ROOTS),"PLAYWRIGHT_OUTPUT_DIR":str(self.raw_dir / "playwright")}
            playwright_env["MEALSWAPP_TASK283_REDIS_OBSERVATION_REQUEST_DIR"] = str(redis_requests)
            generation_before = self.redis_generation_snapshot()
            (evidence / "backend").mkdir(exist_ok=True)
            (evidence / "backend/task283-redis-before.json").write_text(json.dumps(generation_before, sort_keys=True) + "\n")
            browser_failure = None
            try:
                real_stack.run_command(
                    ["bunx", "playwright", "test", "-c", "playwright.real-stack.config.ts", "tests/task283-manual-catalog.spec.ts", "--grep", "Task 294 production transport corruption"],
                    cwd=ROOT / "frontend", env=playwright_env, timeout=self.timeout,
                )
                suite_environment = dict(playwright_env)
                suite_environment["MEALSWAPP_TASK294_REAL_E2E"] = "0"
                suite_environment["MEALSWAPP_TASK283_AUTH_STATE_DIR"] = str(self.raw_dir / "task283-auth-state-suite")
                real_stack.run_command(
                    ["bunx", "playwright", "test", "-c", "playwright.real-stack.config.ts", "tests/task283-manual-catalog.spec.ts", "--grep-invert", "Task 294 production transport corruption"],
                    cwd=ROOT / "frontend", env=suite_environment, timeout=self.timeout,
                )
            except BaseException as error:
                browser_failure = error
                self.events.append("browser_product_nonpass" if isinstance(error, __import__("subprocess").CalledProcessError) else "browser_infrastructure_nonpass")
                diagnostic = "\n".join(
                    value.decode("utf-8", errors="replace") if isinstance(value, bytes) else str(value)
                    for value in (getattr(error, "stdout", None), getattr(error, "stderr", None))
                    if value
                )
                for secret in (user["email"], user["password"], self.run_id):
                    diagnostic = diagnostic.replace(secret, "[redacted]")
                safe_lines = [line[:400] for line in diagnostic.splitlines() if any(marker in line for marker in ("task283", "Error", "Expected", "Received", "Timeout"))][:100]
                (evidence / "browser-diagnostics.txt").write_text(
                    f"{type(error).__name__}\n" + "\n".join(safe_lines) + "\n", encoding="utf-8"
                )
            finally:
                observer_stop.set()
                observer.join(timeout=10)
                if observer.is_alive():
                    raise RuntimeError("Redis generation observer did not stop")
            generation_after = self.redis_generation_snapshot()
            (evidence / "backend/task283-redis-after.json").write_text(json.dumps(generation_after, sort_keys=True) + "\n")
            browser_result = evidence / "browser.json"
            if not browser_result.is_file():
                self.write_synthetic_browser(evidence, "BLOCKED")
            browser_payload = json.loads(browser_result.read_text(encoding="utf-8"))
            expected_projects = {"real-stack-desktop-chromium", "real-stack-mobile-chromium"}
            if not expected_projects.issubset(set(browser_payload.get("projects", []))):
                self.write_synthetic_browser(evidence, "BLOCKED")
            try:
                self.write_task294_proof(evidence)
                self.write_backend_evidence(evidence)
            except Exception as error:
                self.events.append("backend_proof_nonpass")
                detail = str(error).replace("\n", " ")[:400]
                (evidence / "backend-proof-diagnostics.txt").write_text(f"{type(error).__name__}: {detail}\n", encoding="utf-8")
                self.combine_results(evidence, "BLOCKED")
            else:
                self.combine_results(evidence, "BLOCKED" if browser_failure is not None and not isinstance(browser_failure, __import__("subprocess").CalledProcessError) else None)
            self.events.append("task283_results_finalized")
        finally:
            for reservation in reservations: reservation.release()

    def write_task294_proof(self, evidence: Path) -> None:
        """Prove the browser-recovered Task 294 mutation has exactly-once effects."""
        source = evidence / "task294-transport-proof.json"
        if not source.is_file():
            raise ValueError("Task 294 browser proof is missing")
        operation = json.loads(source.read_text(encoding="utf-8"))
        if operation.get("schema") != "mealswapp.task294-transport-proof.v1":
            raise ValueError("Task 294 browser proof schema is invalid")
        name, entity_id, key = operation.get("name"), operation.get("entityId"), operation.get("idempotencyKey")
        if not isinstance(name, str) or not name.startswith("Task 294 transport ") or not isinstance(entity_id, str) or not re.fullmatch(r"[0-9a-f-]{36}", entity_id, re.I) or not isinstance(key, str) or not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}", key):
            raise ValueError("Task 294 browser proof identity is invalid")
        actual = {
            "foodCount": int(read_only_psql(self.target, self.database, "SELECT count(*) FROM food_items WHERE id=%s::uuid AND name=%s", (entity_id, name))),
            "auditCount": int(read_only_psql(self.target, self.database, "SELECT count(*) FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=%s::uuid AND action='manual_create'", (entity_id,))),
            "idempotencyCount": int(read_only_psql(self.target, self.database, "SELECT count(*) FROM mutation_idempotency_keys WHERE method='POST' AND route='/admin/items' AND key=%s", (key,))),
        }
        expected = operation.get("expected")
        snapshots = operation.get("generationSnapshots")
        if not isinstance(expected, dict) or not isinstance(snapshots, dict):
            raise ValueError("Task 294 browser proof expectations are invalid")
        actual.update(redis_generation_actual(expected, snapshots, self.redis_observation_directory))
        failures = compare_expected(expected, actual)
        proof = {"schema": "mealswapp.task294-transport-proof.v1", "operation": operation, "actual": actual, "assertionFailures": failures, "transactionReadOnly": True}
        (evidence / "backend/task294-transport-proof.json").write_text(json.dumps(proof, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        if failures:
            raise ValueError("Task 294 exact-effect proof failed: " + ", ".join(failures))

    def redis_generation_snapshot(self) -> dict[str, object]:
        """Read the application generation key from the owned Redis container."""
        result = real_stack.run_command(["docker", "exec", self.container, "redis-cli", "GET", "classification:cache-generation:v1"], timeout=10)
        raw = result.stdout.strip()
        return {"key": "classification:cache-generation:v1", "value": raw if raw.isdigit() else None, "reachable": True}

    def observe_redis_generation_requests(
        self,
        request_directory: Path,
        observation_directory: Path,
        stop: threading.Event,
    ) -> None:
        """Answer browser handshakes with operation-scoped Redis queries owned by the harness."""
        handled: set[str] = set()
        while not stop.is_set() or any(request_directory.glob("*.request.json")):
            for request_path in sorted(request_directory.glob("*.request.json")):
                snapshot_id = request_path.name.removesuffix(".request.json")
                if snapshot_id in handled:
                    continue
                handled.add(snapshot_id)
                try:
                    request = json.loads(request_path.read_text(encoding="utf-8"))
                    if (
                        request != {"schema": "mealswapp.task283-redis-observation-request.v1", "id": snapshot_id}
                        or not SNAPSHOT_ID_RE.fullmatch(snapshot_id)
                    ):
                        raise ValueError("Redis generation observation request is invalid")
                    snapshot = self.redis_generation_snapshot()
                    # A fresh isolated database has no cache-generation key until
                    # the first committed classification mutation; treat that
                    # absent baseline as generation zero for operation deltas.
                    if snapshot["value"] is None:
                        snapshot["value"] = "0"
                    observation = {
                        "schema": "mealswapp.task283-redis-observation.v1",
                        "id": snapshot_id,
                        "key": snapshot["key"],
                        "value": snapshot["value"],
                    }
                    response = observation
                    destination = observation_directory / f"{snapshot_id}.json"
                    temporary = destination.with_suffix(".tmp")
                    temporary.write_text(json.dumps(observation, sort_keys=True) + "\n", encoding="utf-8")
                    os.replace(temporary, destination)
                except Exception as error:
                    response = {
                        "schema": "mealswapp.task283-redis-observation-error.v1",
                        "id": snapshot_id,
                        "error": str(error).replace("\n", " ")[:240],
                    }
                response_path = request_directory / f"{snapshot_id}.response.json"
                temporary_response = response_path.with_suffix(".tmp")
                temporary_response.write_text(json.dumps(response, sort_keys=True) + "\n", encoding="utf-8")
                os.replace(temporary_response, response_path)
                request_path.unlink(missing_ok=True)
            time.sleep(0.01)

    @staticmethod
    def write_synthetic_browser(evidence: Path, status: str) -> None:
        """Emit the complete browser producer shape when setup, timeout, or signal interrupts Playwright."""
        results = []
        for criterion in CRITERIA:
            requirement = next(key for key, ids in REQUIREMENT_CRITERIA.items() if criterion in ids)
            results.append({
                "criterionId": criterion,
                "status": status,
                "rootCauseId": ROOTS["P08-SWR" + requirement.split("-")[-1]],
                "requestIds": [],
                "evidence": [],
                "backendEvidence": [],
            })
        (evidence / "browser.json").write_text(
            json.dumps({"results": results, "projects": []}, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )

    def write_backend_evidence(self, evidence: Path) -> None:
        """Collect operation-scoped read-only mutation, audit, rollback, and canonical-state proofs."""
        operations = evidence / "operations"
        backend = evidence / "backend"
        backend.mkdir(exist_ok=True)
        index: dict[str, object] = {"criteria": {}, "operations": {}, "failedCriteria": []}
        failed_criteria: set[str] = set()
        for operation_path in sorted(operations.glob("*.json")) if operations.is_dir() else []:
            operation = json.loads(operation_path.read_text(encoding="utf-8"))
            proof = self.operation_proof(operation)
            proof_name = f"task283-{operation_path.stem}-proof.json"
            proof_path = backend / proof_name
            proof_path.write_text(json.dumps(proof, indent=2, sort_keys=True) + "\n", encoding="utf-8")
            relative = f"backend/{proof_name}"
            index["operations"][operation_path.relative_to(evidence).as_posix()] = relative
            for criterion in operation["criterionIds"]:
                index["criteria"].setdefault(criterion, []).append(relative)
                if proof["assertionFailures"]:
                    failed_criteria.add(criterion)
        index["failedCriteria"] = sorted(failed_criteria)
        (backend / "task283-proof-index.json").write_text(json.dumps(index, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    def operation_proof(self, operation: dict[str, object]) -> dict[str, object]:
        """Resolve one browser-recorded operation to exact persisted rows and expected invariants."""
        if operation.get("schema") != "mealswapp.task283-operation.v1":
            raise ValueError("operation evidence schema is invalid")
        criteria = operation.get("criterionIds")
        if not isinstance(criteria, list) or not criteria or any(value not in CRITERIA for value in criteria):
            raise ValueError("operation evidence criteria are invalid")
        kind = operation.get("kind")
        name = operation.get("name")
        expected = operation.get("expected")
        request_ids = operation.get("requestIds")
        if kind not in {"item", "classification", "rejected_item", "rejected_classification", "audit_rollback"} or not isinstance(name, str) or not name.startswith("Task 283 ") or not isinstance(expected, dict):
            raise ValueError("operation evidence identity is invalid")
        if not isinstance(request_ids, list) or any(not isinstance(value, str) or not REQUEST_ID_RE.fullmatch(value) for value in request_ids):
            raise ValueError("operation request correlation is invalid")
        actual: dict[str, object] = {}
        entity_id = operation.get("entityId")
        if kind == "item":
            if not isinstance(entity_id, str) or not re.fullmatch(r"[0-9a-f-]{36}", entity_id, re.I):
                raise ValueError("item operation identity is invalid")
            actual.update(json.loads(read_only_psql(
                self.target, self.database,
                """SELECT json_build_object(
                    'rowCount', count(*), 'active', count(*) FILTER (WHERE deleted_at IS NULL) = 1,
                    'deleted', count(*) FILTER (WHERE deleted_at IS NOT NULL) = 1,
                    'name', max(name), 'physicalState', max(physical_state),
                    'metricBasis', max(CASE physical_state WHEN 'solid' THEN '100g' ELSE '100ml' END),
                    'density', max(density_grams_per_milliliter), 'densitySourceKind', max(density_source_kind),
                    'macros', COALESCE((json_agg(json_build_object('protein',protein_per_100,'carbohydrates',carbohydrates_per_100,'fat',fat_per_100)) FILTER (WHERE id IS NOT NULL))->0, '{}'::json),
                    'micros', COALESCE((json_agg(micronutrients) FILTER (WHERE id IS NOT NULL))->0, '{}'::json)
                ) FROM food_items WHERE id=%s::uuid""",
                (entity_id,),
            )))
            actual["ownerless"] = int(read_only_psql(self.target, self.database, "SELECT count(*) FROM custom_food_items WHERE name=%s", (name,))) == 0
            actual["auditActions"] = json.loads(read_only_psql(
                self.target, self.database,
                "SELECT COALESCE(json_object_agg(action, amount), '{}'::json)::text FROM (SELECT action,count(*) amount FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=%s::uuid GROUP BY action) grouped",
                (entity_id,),
            ))
        elif kind == "classification":
            if not isinstance(entity_id, str) or not re.fullmatch(r"[0-9a-f-]{36}", entity_id, re.I):
                raise ValueError("classification operation identity is invalid")
            actual.update(json.loads(read_only_psql(
                self.target, self.database,
                "SELECT json_build_object('rowCount',count(*),'active',count(*) FILTER (WHERE deleted_at IS NULL)=1,'deleted',count(*) FILTER (WHERE deleted_at IS NOT NULL)=1,'name',max(name),'kind',max(kind),'parentId',max(parent_id::text)) FROM classifications WHERE id=%s::uuid",
                (entity_id,),
            )))
            actual["auditActions"] = json.loads(read_only_psql(
                self.target, self.database,
                "SELECT COALESCE(json_object_agg(action, amount), '{}'::json)::text FROM (SELECT action,count(*) amount FROM admin_audit_entries WHERE entity_type='classification' AND entity_id=%s::uuid GROUP BY action) grouped",
                (entity_id,),
            ))
        else:
            actual["rowCount"] = int(read_only_psql(self.target, self.database, "SELECT count(*) FROM food_items WHERE name=%s", (name,)))
            actual["auditCount"] = int(read_only_psql(
                self.target, self.database,
                "SELECT count(*) FROM admin_audit_entries WHERE request_id = ANY(string_to_array(%s, ','))",
                (",".join(request_ids),),
            )) if request_ids else 0
        key = operation.get("idempotencyKey")
        if key is not None:
            if not isinstance(key, str) or not re.fullmatch(r"[0-9a-f-]{36}", key, re.I):
                raise ValueError("operation idempotency key is invalid")
            actual["idempotencyCount"] = int(read_only_psql(
                self.target, self.database,
                "SELECT count(*) FROM mutation_idempotency_keys WHERE method='POST' AND route='/admin/items' AND key=%s",
                (key,),
            ))
        observed = operation.get("observed")
        if observed is not None:
            if not isinstance(observed, dict) or any(not isinstance(name, str) or not isinstance(value, (str, int, float, bool, type(None))) for name, value in observed.items()):
                raise ValueError("operation observed values are invalid")
            actual.update(observed)
        actual.update(redis_generation_actual(
            expected,
            operation.get("generationSnapshots"),
            self.redis_observation_directory,
        ))
        failures = compare_expected(expected, actual)
        return {
            "schema": "mealswapp.task283-operation-proof.v1",
            "operation": operation,
            "actual": actual,
            "assertionFailures": failures,
            "transactionReadOnly": True,
        }

    @staticmethod
    def combine_results(evidence: Path, failure_status: str | None = None) -> None:
        """Normalize exactly the Task 283 criteria and map non-pass shards to Task 283 roots."""
        try: values = json.loads((evidence / "browser.json").read_text()) .get("results", [])
        except (OSError, ValueError, AttributeError): values = []
        by_id = {}; malformed = False
        for value in values if isinstance(values, list) else []:
            if not isinstance(value, dict) or not isinstance(value.get("criterionId"), str): malformed = True; continue
            if value["criterionId"] in by_id: malformed = True
            by_id[value["criterionId"]] = value
        results = []
        for criterion in CRITERIA:
            source = {} if malformed else by_id.get(criterion, {})
            status = source.get("status") if isinstance(source, dict) else None
            if failure_status == "FAIL" or failure_status == "BLOCKED" and status == "PASS":
                status = failure_status
            if status not in {"PASS", "FAIL", "BLOCKED"}: status = "BLOCKED"
            request_ids = [item for item in source.get("requestIds", []) if isinstance(item, str) and REQUEST_ID_RE.fullmatch(item)]
            source_evidence = [item for item in source.get("evidence", []) if isinstance(item, dict) and item.get("type") in {"playwright", "backend"} and isinstance(item.get("path"), str) and not item["path"].startswith(("/", "\\"))]
            backend_summaries = [item for item in source.get("backendEvidence", []) if isinstance(item, str) and BACKEND_RE.fullmatch(item) and item.split("=", 1)[0] in BACKEND_KEYS]
            result = {"criterionId": criterion, "status": status, "requestIds": request_ids, "evidence": source_evidence, "backendEvidence": sorted(set(backend_summaries))}
            proof_index_file = evidence / "backend/task283-proof-index.json"
            if proof_index_file.is_file():
                proof_index = json.loads(proof_index_file.read_text())
                for proof_path in proof_index.get("criteria", {}).get(criterion, []):
                    if not any(item.get("type") == "backend" and item.get("path") == proof_path for item in result["evidence"]):
                        result["evidence"].append({"type": "backend", "path": proof_path})
                if criterion in proof_index.get("failedCriteria", []):
                    status = result["status"] = "FAIL"
            if status != "PASS":
                requirement = next(key for key, ids in REQUIREMENT_CRITERIA.items() if criterion in ids)
                expected_root = ROOTS.get("P08-SWR" + requirement.split("-")[-1], INFRASTRUCTURE_ROOT)
                result["rootCauseId"] = expected_root if source.get("rootCauseId") == INFRASTRUCTURE_ROOT else source.get("rootCauseId") if source.get("rootCauseId") in SYNCHRONIZED_ROOTS else expected_root
            results.append(result)
        (evidence / "results.json").write_text(json.dumps({"results":results}, indent=2, sort_keys=True)+"\n")
        for requirement, ids in REQUIREMENT_CRITERIA.items():
            (evidence / f"{requirement}.json").write_text(json.dumps({"results":[r for r in results if r["criterionId"] in ids]}, indent=2, sort_keys=True)+"\n")


def finalize_reports(run_id: str, evidence: Path, defer_report: bool) -> int:
    """Report each Task 283 requirement once, preserving FAIL over BLOCKED."""
    results = json.loads((evidence / "results.json").read_text(encoding="utf-8"))["results"]
    statuses = {item["status"] for item in results}
    expected = 1 if "FAIL" in statuses else 2 if "BLOCKED" in statuses else 0
    if defer_report:
        return expected
    codes = []
    for requirement in REQUIREMENT_CRITERIA:
        command = [sys.executable, str(ROOT / "scripts/phase08_acceptance.py"), "report", "--run-id", f"task283-{run_id}-{requirement.lower()}", "--result", str(evidence / f"{requirement}.json"), "--evidence-root", str(evidence), "--requirement", requirement]
        codes.append(__import__("subprocess").run(command, cwd=ROOT, check=False).returncode)
    return 1 if 1 in codes else 2 if 2 in codes else 0


def finalize_failure_reports(harness: Task283Harness, defer_report: bool) -> int:
    """Materialize and report all 44 mapped blockers after setup, timeout, or signal failure."""
    evidence = harness.artifacts / "acceptance"
    evidence.mkdir(parents=True, exist_ok=True)
    browser = evidence / "browser.json"
    if not browser.is_file():
        harness.write_synthetic_browser(evidence, "BLOCKED")
    harness.combine_results(evidence, "BLOCKED")
    return finalize_reports(harness.run_id, evidence, defer_report)


def main(argv: list[str] | None = None) -> int:
    """Allocate owned infrastructure, run both browser projects, report, and tear down."""
    parser = argparse.ArgumentParser(); parser.add_argument("--timeout-seconds", type=float, default=300); parser.add_argument("--defer-report", action="store_true")
    args = parser.parse_args(argv)
    run_id = "pending"
    harness = None
    controller = real_stack.SignalController()
    previous = real_stack.install_signal_handlers(controller)
    try:
        real_stack.validate_environment("development")
        target = real_stack.PostgresTarget.parse(
            __import__("os").environ.get("MEALSWAPP_E2E_POSTGRES_ADMIN_URL", "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres")
        )
        harness = Task283Harness(target, timeout=args.timeout_seconds, signal_controller=controller)
        run_id = harness.run_id
        harness.run()
        return finalize_reports(run_id, harness.artifacts / "acceptance", args.defer_report)
    except BaseException as error:
        if harness is None:
            print(f"Task 283 infrastructure blocked before ownership: {type(error).__name__}", file=sys.stderr)
            return 2
        print(f"Task 283 real-stack run failed safely: {type(error).__name__}", file=sys.stderr)
        try:
            finalize_failure_reports(harness, args.defer_report)
        except BaseException as finalizer_error:
            print(f"Task 283 report finalization failed: {type(finalizer_error).__name__}", file=sys.stderr)
            return 1
        return 2
    finally:
        real_stack.restore_signal_handlers(previous)


if __name__ == "__main__":
    raise SystemExit(main())
