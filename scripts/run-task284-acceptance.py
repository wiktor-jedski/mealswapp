#!/usr/bin/env python3
"""Run Task 284 private-isolation, portability, and erasure acceptance."""

# Implements DESIGN-005 RepositoryInterfaces and DESIGN-008 AccountDeleter acceptance.
from __future__ import annotations

import argparse
import importlib.util
import json
import re
import secrets
import sys
import threading
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def load_module(name: str, path: Path):
    """Load one repository script without adding an importable package surface."""
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"{name} could not be loaded")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


real_stack = load_module("task284_real_stack", ROOT / "scripts/run-real-stack-e2e.py")
task283 = load_module("task284_task283_helpers", ROOT / "scripts/run-task283-acceptance.py")

TASK_REQUIREMENTS = {"SW-REQ-043", "SW-REQ-072", "SW-REQ-073"}
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
    "SW-REQ-043": "ROOT-T284-PRIVATE-ISOLATION",
    "SW-REQ-072": "ROOT-T284-DATA-PORTABILITY",
    "SW-REQ-073": "ROOT-T284-ACCOUNT-ERASURE",
}
EXPORT_OWNER_ROOT = "ROOT-T284-EXPORT-OWNER-PROJECTION"
INFRASTRUCTURE_ROOT = "ROOT-T284-ACCEPTANCE-INFRASTRUCTURE"
SYNCHRONIZED_ROOTS = tuple(ROOTS.values()) + (EXPORT_OWNER_ROOT, INFRASTRUCTURE_ROOT)
UUID_RE = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}", re.I)
REQUEST_ID_RE = task283.REQUEST_ID_RE
BACKEND_KEYS = task283.BACKEND_KEYS
BACKEND_RE = task283.BACKEND_RE


def validate_snapshot_request(
    value: object,
    filename_id: str,
    capability: dict[str, str],
) -> dict[str, object]:
    """Validate one private raw proof request before any owned target is queried."""
    if not isinstance(value, dict) or set(value) != {
        "schema", "id", "capabilityNonce", "phase", "userA", "userB", "itemA",
        "deletedItemA", "itemB", "globalItem", "deletionRequest",
    }:
        raise ValueError("snapshot request shape is invalid")
    if (
        value["schema"] != "mealswapp.task284-snapshot-request.v2"
        or value["id"] != filename_id
        or value["capabilityNonce"] != capability["nonce"]
    ):
        raise ValueError("snapshot request identity is invalid")
    if value["userA"] != capability["userA"] or value["userB"] != capability["userB"]:
        raise ValueError("snapshot users are outside the run-owned capability")
    if value["phase"] not in {"before", "after"}:
        raise ValueError("snapshot phase is invalid")
    for key in ("id", "userA", "userB", "itemA", "deletedItemA", "itemB", "globalItem"):
        if not isinstance(value[key], str) or not UUID_RE.fullmatch(value[key]):
            raise ValueError("snapshot target identity is invalid")
    if value["deletionRequest"] is not None and (
        not isinstance(value["deletionRequest"], str) or not UUID_RE.fullmatch(value["deletionRequest"])
    ):
        raise ValueError("deletion request identity is invalid")
    return value


class Task284Harness(real_stack.Harness):
    """Own Task 284 PostgreSQL, Redis, API, worker, browser, evidence, and cleanup."""

    snapshots: list[dict[str, object]]
    bound_targets: dict[str, str]
    proof_capability: dict[str, str]
    proof_query_observations: list[dict[str, object]]

    def application_environment(self, database_url: str, redis_url: str, api_port: int, frontend_port: int) -> dict[str, str]:
        """Add only Task 284 managed-run markers to the inherited application environment."""
        environment = super().application_environment(database_url, redis_url, api_port, frontend_port)
        environment.update({"MEALSWAPP_TASK284_REAL_E2E": "1", "MEALSWAPP_REAL_STACK_MANAGED": "1"})
        return environment

    def execute(self) -> None:
        """Run the production API/UI/worker and collect independent read-only proof."""
        reservations = []
        observer_stop = threading.Event()
        observer: threading.Thread | None = None
        try:
            real_stack.create_database(self.target, self.database, self.comment)
            self.events.append("database_created")
            redis_port = self.start_redis()
            reservations = real_stack.reserve_ports(2)
            database_url = self.target.database_url(self.database)
            redis_url = f"redis://127.0.0.1:{redis_port}/0"
            env = self.application_environment(database_url, redis_url, reservations[0].port, reservations[1].port)
            real_stack.run_command(["go", "run", "./cmd/migrate", "up"], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            assert self.raw_dir is not None
            api_binary = self.raw_dir / "mealswapp-api"
            worker_binary = self.raw_dir / "mealswapp-worker"
            bootstrap_binary = self.raw_dir / "admin-bootstrap"
            for output, command in (
                (api_binary, "./cmd/api"),
                (worker_binary, "./cmd/worker"),
                (bootstrap_binary, "./cmd/admin-bootstrap"),
            ):
                real_stack.run_command(["go", "build", "-o", str(output), command], cwd=ROOT / "backend", env=env, timeout=self.timeout)
            env, api_port, frontend_port, _api, frontend = self.start_application_stack(
                api_binary, database_url, redis_url, reservations
            )
            user_a, ids_a = real_stack.register_fixture(f"http://127.0.0.1:{api_port}", self.run_id)
            user_b, ids_b = real_stack.register_fixture(f"http://127.0.0.1:{api_port}", secrets.token_hex(12))
            self.request_ids.extend(ids_a + ids_b)
            bootstrap = real_stack.run_command(
                [str(bootstrap_binary), "--environment", "development", "--email", user_b["email"]],
                cwd=ROOT / "backend",
                env=env,
                timeout=self.timeout,
            )
            if f"user_id={user_b['user_id']}" not in bootstrap.stdout or "actor=operator" not in bootstrap.stdout:
                raise RuntimeError("administrator bootstrap returned an unexpected target")
            for user in (user_a, user_b):
                task283.read_only_sql("SELECT 1")
                real_stack.run_command(
                    [*self.target.command(self.database), "--set", f"uid={user['user_id']}", "-Atqf", "-"],
                    env=self.target.environment(),
                    input_text=(
                        "INSERT INTO entitlements (user_id,tier,status,search_limit_per_24h,allowed_modes,expires_at) "
                        "VALUES (:'uid'::uuid,'trial','active',100,ARRAY['catalog','substitution','daily_diet','daily_diet_alternative'],now()+interval '1 day');"
                    ),
                )
                real_stack.run_command(
                    ["docker", "exec", self.container, "redis-cli", "SET", f"user:{user['user_id']}:task284", "present"],
                    timeout=10,
                )
            real_stack.run_command(
                [
                    *self.target.command(self.database),
                    "--set", f"uid={user_a['user_id']}",
                    "--set", f"marker=task284-{self.run_id}",
                    "-Atqf", "-",
                ],
                env=self.target.environment(),
                input_text=(
                    "INSERT INTO oauth_identities (user_id,provider,provider_user_id,email) "
                    "VALUES (:'uid'::uuid,'apple',:'marker',:'marker'||'@example.test');"
                    "INSERT INTO password_reset_tokens (token_hash,user_id,expires_at) "
                    "VALUES (:'marker'||'-reset',:'uid'::uuid,now()+interval '1 day');"
                    "INSERT INTO usage_windows (user_id,feature,started_at,search_count) "
                    "VALUES (:'uid'::uuid,'task284-erasure',date_trunc('hour',now()),1);"
                ),
            )
            worker = self.start_process("worker", [str(worker_binary)], ROOT / "backend", env, "worker.raw.log")
            self.worker_identity = {
                "pid": worker.pid,
                "startToken": real_stack.validate_process_start_token(real_stack.process_start_token(worker.pid)),
            }
            time.sleep(0.5)
            if worker.poll() is not None:
                raise RuntimeError("production deletion worker exited before acceptance")

            capability = self.raw_dir / "task284-harness-capability.json"
            nonce = secrets.token_hex(24)
            evidence = self.artifacts / "acceptance"
            evidence.mkdir()
            capability.write_text(json.dumps({
                "schema": "mealswapp.task284-harness-capability.v1",
                "runId": self.run_id,
                "nonce": nonce,
                "baseURL": f"http://127.0.0.1:{frontend_port}",
                "evidenceRoot": str(evidence),
                "frontendProcess": {
                    "pid": frontend.pid,
                    "startToken": real_stack.validate_process_start_token(real_stack.process_start_token(frontend.pid)),
                },
            }, sort_keys=True) + "\n")
            capability.chmod(0o600)
            proof_requests = self.raw_dir / "task284-proof-requests"
            proof_requests.mkdir(mode=0o700)
            self.snapshots = []
            self.bound_targets = {}
            self.bound_deletion_request = None
            self.proof_capability = {
                "nonce": nonce,
                "userA": user_a["user_id"],
                "userB": user_b["user_id"],
            }
            self.proof_query_observations = []
            observer = threading.Thread(
                target=self.observe_snapshot_requests,
                args=(proof_requests, observer_stop),
                daemon=True,
            )
            observer.start()

            playwright_env = {
                **env,
                "MEALSWAPP_TASK284_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": f"http://127.0.0.1:{frontend_port}",
                "MEALSWAPP_TASK284_CAPABILITY_FILE": str(capability),
                "MEALSWAPP_TASK284_CAPABILITY_NONCE": nonce,
                "MEALSWAPP_TASK284_PROOF_REQUEST_DIR": str(proof_requests),
                "MEALSWAPP_TASK284_USER_A_ID": user_a["user_id"],
                "MEALSWAPP_TASK284_USER_A_EMAIL": user_a["email"],
                "MEALSWAPP_TASK284_USER_A_PASSWORD": user_a["password"],
                "MEALSWAPP_TASK284_USER_B_ID": user_b["user_id"],
                "MEALSWAPP_TASK284_USER_B_EMAIL": user_b["email"],
                "MEALSWAPP_TASK284_USER_B_PASSWORD": user_b["password"],
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
                "MEALSWAPP_PHASE08_RESULT_FILE": "browser.json",
                "MEALSWAPP_PHASE08_CRITERIA": ",".join(CRITERIA),
                "MEALSWAPP_PHASE08_EXPECTED_PROJECTS": "real-stack-desktop-chromium",
                "MEALSWAPP_PHASE08_INFRASTRUCTURE_ROOT": INFRASTRUCTURE_ROOT,
                "MEALSWAPP_PHASE08_SYNCHRONIZED_ROOTS": ",".join(SYNCHRONIZED_ROOTS),
                "PLAYWRIGHT_OUTPUT_DIR": str(self.raw_dir / "playwright"),
            }
            browser_failure: BaseException | None = None
            try:
                real_stack.run_command(
                    ["bunx", "playwright", "test", "-c", "playwright.real-stack.config.ts", "tests/task284-private-erasure.spec.ts"],
                    cwd=ROOT / "frontend",
                    env=playwright_env,
                    timeout=self.timeout,
                )
            except BaseException as error:
                browser_failure = error
                self.events.append("browser_product_nonpass")
                diagnostic = "\n".join(
                    value.decode("utf-8", errors="replace") if isinstance(value, bytes) else str(value)
                    for value in (getattr(error, "stdout", None), getattr(error, "stderr", None))
                    if value
                )
                for secret in (
                    user_a["email"], user_a["password"], user_a["user_id"],
                    user_b["email"], user_b["password"], user_b["user_id"], self.run_id,
                ):
                    diagnostic = diagnostic.replace(secret, "[redacted]")
                safe_lines = [
                    line[:400] for line in diagnostic.splitlines()
                    if any(marker in line for marker in ("task284", "Error", "Expected", "Received", "Timeout"))
                ][:120]
                (evidence / "browser-diagnostics.txt").write_text(
                    f"{type(error).__name__}\n" + "\n".join(safe_lines) + "\n"
                )
            finally:
                observer_stop.set()
                observer.join(timeout=10)
                if observer.is_alive():
                    raise RuntimeError("Task 284 proof observer did not stop")
            self.write_backend_proof(evidence)
            self.combine_results(evidence, "BLOCKED" if browser_failure and not isinstance(browser_failure, __import__("subprocess").CalledProcessError) else None)
            self.events.append("task284_results_finalized")
        finally:
            observer_stop.set()
            if observer is not None and observer.is_alive():
                observer.join(timeout=10)
            for reservation in reservations:
                reservation.release()

    def observe_snapshot_requests(self, directory: Path, stop: threading.Event) -> None:
        """Answer private browser handshakes with fixed parameterized DB/cache reads."""
        handled: set[str] = set()
        while not stop.is_set() or any(directory.glob("*.request.json")):
            for request_path in sorted(directory.glob("*.request.json")):
                request_id = request_path.name.removesuffix(".request.json")
                if request_id in handled:
                    continue
                handled.add(request_id)
                response_path = directory / f"{request_id}.response.json"
                try:
                    request = validate_snapshot_request(
                        json.loads(request_path.read_text()), request_id, self.proof_capability
                    )
                    snapshot = self.collect_snapshot(request)
                    self.snapshots.append(snapshot)
                    response_path.write_text(json.dumps(snapshot, sort_keys=True) + "\n")
                    response_path.chmod(0o600)
                except Exception as error:
                    response_path.write_text(json.dumps({
                        "schema": "mealswapp.task284-snapshot-error.v1",
                        "error": type(error).__name__,
                        "detail": str(error).replace("\n", " ")[:240],
                    }) + "\n")
                    response_path.chmod(0o600)
                finally:
                    request_path.unlink(missing_ok=True)
            stop.wait(0.05)

    def collect_snapshot(self, request: dict[str, object]) -> dict[str, object]:
        """Collect one identity-free logical snapshot from owned PostgreSQL and Redis."""
        def read(sql: str, parameters: tuple[str, ...]) -> str:
            if not parameters or sql.count("%s") != len(parameters) or re.search(
                r"\b(?:INSERT|UPDATE|DELETE|ALTER|DROP|TRUNCATE|CREATE)\b", sql, re.I
            ):
                raise ValueError("proof query is not fixed read-only parameterized SQL")
            self.proof_query_observations.append({
                "readOnly": True,
                "parameterCount": len(parameters),
            })
            return task283.read_only_psql(self.target, self.database, sql, parameters)

        def count(table: str, column: str, value: str) -> int:
            allowed = {
                ("users", "id"), ("oauth_identities", "user_id"), ("user_sessions", "user_id"),
                ("password_reset_tokens", "user_id"), ("user_profiles", "user_id"),
                ("saved_items", "user_id"), ("saved_diets", "user_id"),
                ("search_history", "user_id"), ("consent_records", "user_id"),
                ("entitlements", "user_id"), ("usage_windows", "user_id"),
                ("mutation_idempotency_keys", "user_id"), ("custom_food_items", "owner_id"),
            }
            if (table, column) not in allowed:
                raise ValueError("proof count target is not allow-listed")
            return int(read(f"SELECT count(*) FROM {table} WHERE {column}=%s::uuid", (value,)))

        def row_hash(table: str, item_id: str) -> str:
            if table not in {"custom_food_items", "food_items"}:
                raise ValueError("proof hash target is not allow-listed")
            return read(
                f"SELECT COALESCE(md5(row_to_json(subject)::text),'missing') FROM (SELECT * FROM {table} WHERE id=%s::uuid) subject",
                (item_id,),
            )

        def custom_count(user_id: str, active_only: bool) -> int:
            suffix = " AND deleted_at IS NULL" if active_only else ""
            return int(read(f"SELECT count(*) FROM custom_food_items WHERE owner_id=%s::uuid{suffix}", (user_id,)))

        user_a = str(request["userA"])
        user_b = str(request["userB"])
        targets = {key: str(request[key]) for key in ("itemA", "deletedItemA", "itemB", "globalItem")}
        if not self.bound_targets:
            if request["phase"] != "before" or request["deletionRequest"] is not None:
                raise ValueError("first snapshot must bind the pre-deletion run fixtures")
            bindings = (
                ("itemA", "custom_food_items", "owner_id", user_a, "Task 284 owner A retained", False),
                ("deletedItemA", "custom_food_items", "owner_id", user_a, "Task 284 owner A selected deletion", True),
                ("itemB", "custom_food_items", "owner_id", user_b, "Task 284 owner B survivor", False),
                ("globalItem", "food_items", None, None, "Task 284 global survivor", False),
            )
            for key, table, owner_column, owner, name, deleted in bindings:
                owner_predicate = "" if owner_column is None else f" AND {owner_column}=%s::uuid"
                deleted_predicate = (
                    "" if table == "food_items"
                    else " AND deleted_at IS NOT NULL" if deleted else " AND deleted_at IS NULL"
                )
                parameters = (targets[key],) if owner is None else (targets[key], owner)
                matches = int(read(
                    f"SELECT count(*) FROM {table} WHERE id=%s::uuid{owner_predicate} AND name=%s{deleted_predicate}",
                    (*parameters, name),
                ))
                if matches != 1:
                    raise ValueError(f"{key} is not the exact run-owned fixture")
            self.bound_targets = targets
        elif targets != self.bound_targets:
            raise ValueError("snapshot targets differ from the run-owned fixture binding")

        deletion_request = request["deletionRequest"]
        if request["phase"] == "before" and deletion_request is not None:
            raise ValueError("pre-deletion snapshot cannot target a deletion request")
        if isinstance(deletion_request, str):
            if self.bound_deletion_request is None:
                matches = int(read(
                    """SELECT count(*) FROM data_deletion_requests
                    WHERE id=%s::uuid
                      AND (user_id=%s::uuid OR (user_id IS NULL AND status='completed'))
                      AND (SELECT count(*) FROM data_deletion_requests)=1""",
                    (deletion_request, user_a),
                ))
                if matches != 1:
                    raise ValueError("deletion request is not bound to run-owned user A")
                self.bound_deletion_request = deletion_request
            elif deletion_request != self.bound_deletion_request:
                raise ValueError("deletion request differs from the run-owned fixture binding")
        deletion = {
            "deletionStatus": None,
            "receiptPresent": False,
            "receiptPseudonymous": False,
            "retryCount": 0,
        }
        if isinstance(deletion_request, str):
            deletion.update(json.loads(read(
                """SELECT json_build_object(
                    'deletionStatus',status,
                    'receiptPresent',receipt_id IS NOT NULL,
                    'receiptPseudonymous',user_id IS NULL AND receipt_id IS NOT NULL,
                    'retryCount',retry_count
                ) FROM data_deletion_requests WHERE id=%s::uuid""",
                (deletion_request,),
            ) or "{}"))
        owned_tables = {
            "users": count("users", "id", user_a),
            "oauthIdentities": count("oauth_identities", "user_id", user_a),
            "sessions": count("user_sessions", "user_id", user_a),
            "passwordResetTokens": count("password_reset_tokens", "user_id", user_a),
            "profiles": count("user_profiles", "user_id", user_a),
            "savedItems": count("saved_items", "user_id", user_a),
            "savedDiets": count("saved_diets", "user_id", user_a),
            "history": count("search_history", "user_id", user_a),
            "consent": count("consent_records", "user_id", user_a),
            "entitlements": count("entitlements", "user_id", user_a),
            "usageWindows": count("usage_windows", "user_id", user_a),
            "mutationIdempotency": count("mutation_idempotency_keys", "user_id", user_a),
            "customItemsRetained": custom_count(user_a, False),
            "savedDietEntries": int(read(
                "SELECT count(*) FROM saved_diet_meal_entries e JOIN saved_diets d ON d.id=e.saved_diet_id WHERE d.user_id=%s::uuid",
                (user_a,),
            )),
            "customItemClassifications": int(read(
                "SELECT count(*) FROM custom_food_item_classifications c JOIN custom_food_items i ON i.id=c.custom_food_item_id WHERE i.owner_id=%s::uuid",
                (user_a,),
            )),
        }
        return {
            "schema": "mealswapp.task284-snapshot.v1",
            "phase": request["phase"],
            "a": {
                "userExists": count("users", "id", user_a) == 1,
                "customItems": custom_count(user_a, True),
                **owned_tables,
                **deletion,
            },
            "b": {
                "userExists": count("users", "id", user_b) == 1,
                "customItems": custom_count(user_b, True),
                "customItemsRetained": custom_count(user_b, False),
                "rowHash": row_hash("custom_food_items", str(request["itemB"])),
            },
            "global": {
                "rowCount": int(read("SELECT count(*) FROM food_items WHERE id=%s::uuid", (str(request["globalItem"]),))),
                "rowHash": row_hash("food_items", str(request["globalItem"])),
            },
            "cache": {
                "a": self.cache_leak_count((user_a, targets["itemA"], targets["deletedItemA"])),
                "b": self.cache_leak_count((user_b, targets["itemB"])),
            },
        }

    def cache_leak_count(self, identities: tuple[str, ...]) -> int:
        """Count fixture identities in every key and string value of the run-owned Redis."""
        if not identities or any(not UUID_RE.fullmatch(value) for value in identities):
            raise ValueError("cache proof identity is invalid")
        keys = real_stack.run_command(
            ["docker", "exec", self.container, "redis-cli", "--scan"], timeout=10
        ).stdout.splitlines()
        leaks = sum(any(identity in key for identity in identities) for key in keys)
        for key in keys:
            kind = real_stack.run_command(
                ["docker", "exec", self.container, "redis-cli", "TYPE", key], timeout=10
            ).stdout.strip()
            if kind == "string":
                value = real_stack.run_command(
                    ["docker", "exec", self.container, "redis-cli", "GET", key], timeout=10
                ).stdout
                leaks += any(identity in value for identity in identities)
        return leaks

    def write_backend_proof(self, evidence: Path) -> None:
        """Write sanitized independent proof and mark mismatches without retaining IDs."""
        before = next((value for value in self.snapshots if value["phase"] == "before"), None)
        after = next((value for value in reversed(self.snapshots) if value["phase"] == "after"), None)
        failures: list[str] = []
        if before is None or after is None:
            failures.append("required_snapshot_missing")
        else:
            expected_a = {
                key: 0 for key in before["a"]
                if key not in {"deletionStatus", "receiptPresent", "receiptPseudonymous", "retryCount"}
            }
            expected_a.update({
                "userExists": False, "deletionStatus": "completed",
                "receiptPresent": True, "receiptPseudonymous": True,
                "retryCount": before["a"]["retryCount"],
            })
            expected_after = {
                "a": expected_a,
                "b": before["b"],
                "global": before["global"],
                "cache": {"a": 0, "b": before["cache"]["b"]},
            }
            failures = task283.compare_expected(expected_after, after)
        backend = evidence / "backend"
        backend.mkdir(exist_ok=True)
        redis_inspect = real_stack.docker_inspect(self.container) or {}
        worker_observed = (
            real_stack.process_start_token(self.worker_identity["pid"])
            == self.worker_identity["startToken"]
        )
        (backend / "task284-proof.json").write_text(json.dumps({
            "schema": "mealswapp.task284-proof.v1",
            "before": before,
            "after": after,
            "assertionFailures": failures,
            "databaseTransactionReadOnly": bool(self.proof_query_observations) and all(
                item["readOnly"] for item in self.proof_query_observations
            ),
            "queriesParameterized": bool(self.proof_query_observations) and all(
                item["parameterCount"] > 0 for item in self.proof_query_observations
            ),
            "queryObservationCount": len(self.proof_query_observations),
            "cacheTargetRunOwned": real_stack.inspect_labels(redis_inspect).get(real_stack.REDIS_RUN_LABEL) == self.run_id,
            "workerProcess": "production" if worker_observed else "missing",
        }, indent=2, sort_keys=True) + "\n")

    @staticmethod
    def write_synthetic_browser(evidence: Path, status: str) -> None:
        """Emit every mapped criterion when Playwright cannot finalize."""
        (evidence / "browser.json").write_text(json.dumps({
            "projects": [],
            "results": [{"criterionId": criterion, "status": status} for criterion in CRITERIA],
        }, indent=2) + "\n")

    @staticmethod
    def combine_results(evidence: Path, failure_status: str | None = None) -> None:
        """Normalize exactly Task 284 criteria and attach independent proof."""
        try:
            values = json.loads((evidence / "browser.json").read_text()).get("results", [])
        except (OSError, ValueError, AttributeError):
            values = []
        by_id = {value.get("criterionId"): value for value in values if isinstance(value, dict)}
        proof = evidence / "backend/task284-proof.json"
        proof_failed = True
        if proof.is_file():
            proof_failed = bool(json.loads(proof.read_text()).get("assertionFailures"))
        results = []
        for criterion in CRITERIA:
            source = by_id.get(criterion, {})
            status = source.get("status")
            if failure_status is not None and status == "PASS":
                status = failure_status
            if status not in {"PASS", "FAIL", "BLOCKED"}:
                status = "BLOCKED"
            if proof_failed:
                status = "FAIL" if proof.is_file() else "BLOCKED"
            result = {
                "criterionId": criterion,
                "status": status,
                "requestIds": sorted({
                    value for value in source.get("requestIds", [])
                    if isinstance(value, str) and REQUEST_ID_RE.fullmatch(value)
                }),
                "evidence": [{"type": "backend", "path": "backend/task284-proof.json"}] if proof.is_file() else [],
                "backendEvidence": sorted({
                    value for value in source.get("backendEvidence", [])
                    if isinstance(value, str) and BACKEND_RE.fullmatch(value) and value.split("=", 1)[0] in BACKEND_KEYS
                }),
            }
            for item in source.get("evidence", []):
                if (
                    isinstance(item, dict) and item.get("type") in {"playwright", "backend"}
                    and isinstance(item.get("path"), str) and not item["path"].startswith(("/", "\\"))
                    and item not in result["evidence"]
                ):
                    result["evidence"].append(item)
            if status != "PASS":
                requirement = next(key for key, criteria in REQUIREMENT_CRITERIA.items() if criterion in criteria)
                proposed = source.get("rootCauseId")
                result["rootCauseId"] = (
                    proposed if proposed in SYNCHRONIZED_ROOTS and proposed != INFRASTRUCTURE_ROOT
                    else ROOTS[requirement]
                )
            results.append(result)
        (evidence / "results.json").write_text(json.dumps({"results": results}, indent=2, sort_keys=True) + "\n")
        for requirement, criteria in REQUIREMENT_CRITERIA.items():
            (evidence / f"{requirement}.json").write_text(json.dumps({
                "results": [item for item in results if item["criterionId"] in criteria]
            }, indent=2, sort_keys=True) + "\n")


def finalize_reports(run_id: str, evidence: Path, defer_report: bool) -> int:
    """Finalize Task 280 shards with FAIL taking precedence over BLOCKED."""
    statuses = {item["status"] for item in json.loads((evidence / "results.json").read_text())["results"]}
    expected = 1 if "FAIL" in statuses else 2 if "BLOCKED" in statuses else 0
    if defer_report:
        return expected
    codes = []
    for requirement in sorted(REQUIREMENT_CRITERIA):
        codes.append(__import__("subprocess").run([
            sys.executable,
            str(ROOT / "scripts/phase08_acceptance.py"),
            "report",
            "--run-id", f"task284-{run_id}-{requirement.lower()}",
            "--result", str(evidence / f"{requirement}.json"),
            "--evidence-root", str(evidence),
            "--requirement", requirement,
        ], cwd=ROOT, check=False).returncode)
    return 1 if 1 in codes else 2 if 2 in codes else 0


def finalize_failure_reports(harness: Task284Harness, defer_report: bool) -> int:
    """Materialize all Task 284 rows after setup, timeout, or signal failure."""
    evidence = harness.artifacts / "acceptance"
    evidence.mkdir(parents=True, exist_ok=True)
    if not (evidence / "browser.json").is_file():
        harness.write_synthetic_browser(evidence, "BLOCKED")
    harness.combine_results(evidence, "BLOCKED")
    return finalize_reports(harness.run_id, evidence, defer_report)


def main(argv: list[str] | None = None) -> int:
    """Allocate the owned stack, run acceptance, report, and tear down."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--timeout-seconds", type=float, default=180)
    parser.add_argument("--defer-report", action="store_true")
    args = parser.parse_args(argv)
    harness: Task284Harness | None = None
    controller = real_stack.SignalController()
    previous = real_stack.install_signal_handlers(controller)
    try:
        real_stack.validate_environment("development")
        target = real_stack.PostgresTarget.parse(
            __import__("os").environ.get(
                "MEALSWAPP_E2E_POSTGRES_ADMIN_URL",
                "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres",
            )
        )
        harness = Task284Harness(target, timeout=args.timeout_seconds, signal_controller=controller)
        harness.run()
        return finalize_reports(harness.run_id, harness.artifacts / "acceptance", args.defer_report)
    except BaseException as error:
        if harness is None:
            print(f"Task 284 infrastructure blocked before ownership: {type(error).__name__}", file=sys.stderr)
            return 2
        print(f"Task 284 real-stack run failed safely: {type(error).__name__}", file=sys.stderr)
        try:
            return finalize_failure_reports(harness, args.defer_report)
        except BaseException as finalizer_error:
            print(f"Task 284 report finalization failed: {type(finalizer_error).__name__}", file=sys.stderr)
            return 1
    finally:
        real_stack.restore_signal_handlers(previous)


if __name__ == "__main__":
    raise SystemExit(main())
