#!/usr/bin/env python3

"""Repeatable authenticated optimization submission/poll responsiveness check.

The fixture body and cookie are supplied by the operator; the report contains
only status, latency, readiness, and queue/worker evidence. It never writes
request bodies, cookies, user IDs, diet IDs, or job IDs.
"""

# Implements DESIGN-014 MetricsCollector capacity evidence for SW-REQ-080/SW-REQ-082.

from __future__ import annotations

import argparse
import concurrent.futures
import json
import math
import os
import threading
import time
import urllib.error
import urllib.request
import uuid
from collections import Counter
from pathlib import Path
from typing import Any


CAPACITY_LATENCY_LIMIT_SECONDS = 2.0
REQUIRED_READINESS_CHECKS = ("redis", "worker", "optimization_queue")
REQUIRED_QUEUE_FIELDS = ("depth", "oldestQueuedAgeSeconds", "oldestPendingAgeSeconds")


def request_json(url: str, method: str = "GET", headers: dict[str, str] | None = None, body: bytes | None = None, timeout: float = 5.0) -> tuple[int, dict[str, Any], float]:
    request = urllib.request.Request(url, data=body, headers=headers or {}, method=method)
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            payload = response.read()
            status = response.status
    except urllib.error.HTTPError as error:
        payload = error.read()
        status = error.code
    latency = time.perf_counter() - started
    try:
        decoded = json.loads(payload.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        decoded = {}
    return status, decoded if isinstance(decoded, dict) else {}, latency


def p95(values: list[float]) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, math.ceil(len(ordered) * 0.95) - 1))
    return ordered[index]


def readiness_sample_is_valid(sample: dict[str, Any]) -> bool:
    if sample.get("status") != 200:
        return False
    checks = sample.get("checks")
    if not isinstance(checks, dict) or any(checks.get(key) != "ok" for key in REQUIRED_READINESS_CHECKS):
        return False
    queue = sample.get("queue")
    if not isinstance(queue, dict) or any(key not in queue for key in REQUIRED_QUEUE_FIELDS):
        return False
    depth = queue["depth"]
    if isinstance(depth, bool) or not isinstance(depth, int) or depth < 0:
        return False
    return all(_finite_non_negative_number(queue[key]) for key in REQUIRED_QUEUE_FIELDS[1:])


def _finite_non_negative_number(value: Any) -> bool:
    return not isinstance(value, bool) and isinstance(value, (int, float)) and math.isfinite(float(value)) and value >= 0


def readiness_monitor(base_url: str, path: str, stop: threading.Event, samples: list[dict[str, Any]], errors: list[str]) -> None:
    try:
        while not stop.is_set():
            status, payload, latency = request_json(base_url + path)
            data = payload.get("data", {})
            if not isinstance(data, dict):
                data = {}
            checks = data.get("checks", {})
            queue = data.get("queue", {})
            if not isinstance(checks, dict):
                checks = {}
            if not isinstance(queue, dict):
                queue = {}
            samples.append({
                "status": status,
                "latencySeconds": latency,
                "checks": {key: checks.get(key) for key in REQUIRED_READINESS_CHECKS},
                "queue": {key: queue.get(key) for key in REQUIRED_QUEUE_FIELDS},
            })
            stop.wait(0.1)
    except Exception as error:  # noqa: BLE001 - the capacity gate must observe monitor failure and fail closed.
        errors.append(type(error).__name__)
        stop.set()


def submit_and_poll(base_url: str, submit_path: str, body: bytes, headers: dict[str, str], poll_timeout: float, poll_interval: float) -> dict[str, Any]:
    submit_headers = dict(headers)
    submit_headers["Content-Type"] = "application/json"
    submit_headers["Idempotency-Key"] = "capacity-check-" + uuid.uuid4().hex
    status, payload, submit_latency = request_json(base_url + submit_path, "POST", submit_headers, body)
    result: dict[str, Any] = {"submitStatus": status, "submitLatencySeconds": submit_latency, "polls": 0, "pollStatuses": []}
    data = payload.get("data", {})
    poll_path = data.get("pollUrl") if isinstance(data, dict) else None
    if status != 202 or not isinstance(poll_path, str) or not poll_path.startswith("/"):
        return result
    replay_status, replay_payload, replay_latency = request_json(base_url + submit_path, "POST", submit_headers, body)
    replay_data = replay_payload.get("data", {})
    result["replayStatus"] = replay_status
    result["replayLatencySeconds"] = replay_latency
    result["replayMatchesAcknowledgement"] = replay_status == 202 and replay_data == data
    deadline = time.monotonic() + poll_timeout
    while time.monotonic() < deadline:
        poll_status, poll_payload, poll_latency = request_json(base_url + poll_path, "GET", {key: value for key, value in headers.items() if key != "Content-Type"})
        result["polls"] += 1
        result["pollStatuses"].append({"status": poll_status, "latencySeconds": poll_latency})
        poll_data = poll_payload.get("data", {})
        job_status = poll_data.get("status") if isinstance(poll_data, dict) else None
        if job_status in {"completed", "failed", "cancelled"}:
            result["terminalStatus"] = job_status
            return result
        time.sleep(poll_interval)
    result["terminalStatus"] = "poll_timeout"
    return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default=os.getenv("MEALSWAPP_CAPACITY_BASE_URL", "http://localhost:8080"))
    parser.add_argument("--submit-path", default=os.getenv("MEALSWAPP_CAPACITY_SUBMIT_PATH", "/api/v1/optimization/jobs"))
    parser.add_argument("--ready-path", default=os.getenv("MEALSWAPP_CAPACITY_READY_PATH", "/ready"))
    parser.add_argument("--body-file", type=Path, default=None)
    parser.add_argument("--requests", type=int, default=int(os.getenv("MEALSWAPP_CAPACITY_REQUESTS", "32")))
    parser.add_argument("--concurrency", type=int, default=int(os.getenv("MEALSWAPP_CAPACITY_CONCURRENCY", "8")))
    parser.add_argument("--poll-timeout", type=float, default=30.0)
    parser.add_argument("--poll-interval", type=float, default=0.1)
    parser.add_argument("--output", type=Path, default=Path(os.getenv("MEALSWAPP_CAPACITY_OUTPUT", "logs/optimization-capacity.json")))
    return parser.parse_args()


def capacity_gate_passes(report: dict[str, Any], request_count: int) -> bool:
    try:
        submission = report["submission"]
        replay = report["replay"]
        poll = report["poll"]
        readiness = report["readiness"]
        queue_evidence = report["queueEvidence"]
        queue_worker_evidence = report["queueWorkerEvidence"]
        if submission["statuses"].get("202", 0) != request_count or submission["samples"] != request_count:
            return False
        if replay["statuses"].get("202", 0) != request_count or replay["samples"] != request_count or replay["acknowledgementMatches"] != request_count:
            return False
        if poll["samples"] <= 0:
            return False
        if submission["p95LatencySeconds"] >= CAPACITY_LATENCY_LIMIT_SECONDS or replay["p95LatencySeconds"] >= CAPACITY_LATENCY_LIMIT_SECONDS or poll["p95LatencySeconds"] >= CAPACITY_LATENCY_LIMIT_SECONDS:
            return False
        if readiness["monitorAlive"] or readiness["monitorErrors"]:
            return False
        if readiness["samples"] <= 0 or readiness["validSamples"] != readiness["samples"]:
            return False
        if not queue_worker_evidence or queue_evidence["samples"] != readiness["validSamples"]:
            return False
        return True
    except (KeyError, TypeError, ValueError):
        return False


def main() -> int:
    args = parse_args()
    if args.requests <= 0 or args.concurrency <= 0 or args.concurrency > args.requests:
        raise SystemExit("--requests must be positive and --concurrency must be between 1 and --requests")
    cookie = os.getenv("MEALSWAPP_CAPACITY_COOKIE", "")
    csrf = os.getenv("MEALSWAPP_CAPACITY_CSRF_TOKEN", "")
    if not cookie or not csrf:
        raise SystemExit("MEALSWAPP_CAPACITY_COOKIE and MEALSWAPP_CAPACITY_CSRF_TOKEN are required")
    if args.body_file is None:
        body_value = os.getenv("MEALSWAPP_CAPACITY_BODY", "")
        if not body_value:
            raise SystemExit("provide --body-file or MEALSWAPP_CAPACITY_BODY")
        body = body_value.encode("utf-8")
    else:
        body = args.body_file.read_bytes()
    headers = {"Cookie": cookie, "X-CSRF-Token": csrf}
    base_url = args.base_url.rstrip("/")
    ready_samples: list[dict[str, Any]] = []
    monitor_errors: list[str] = []
    stop = threading.Event()
    monitor = threading.Thread(target=readiness_monitor, args=(base_url, args.ready_path, stop, ready_samples, monitor_errors), daemon=True)
    monitor.start()
    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as executor:
        futures = [executor.submit(submit_and_poll, base_url, args.submit_path, body, headers, args.poll_timeout, args.poll_interval) for _ in range(args.requests)]
        results = [future.result() for future in futures]
    stop.set()
    monitor.join(timeout=2)
    elapsed = time.perf_counter() - started

    submission_latencies = [result["submitLatencySeconds"] for result in results]
    replay_latencies = [result["replayLatencySeconds"] for result in results if "replayLatencySeconds" in result]
    poll_latencies = [poll["latencySeconds"] for result in results for poll in result["pollStatuses"]]
    statuses = Counter(str(result.get("submitStatus")) for result in results)
    replay_statuses = Counter(str(result.get("replayStatus")) for result in results if "replayStatus" in result)
    terminal_statuses = Counter(str(result.get("terminalStatus", "not_accepted")) for result in results)
    readiness_statuses = Counter(str(sample["status"]) for sample in ready_samples)
    valid_readiness_samples = [sample for sample in ready_samples if readiness_sample_is_valid(sample)]
    queue_worker_evidence = Counter(
        f"{sample['checks'].get('redis')}|{sample['checks'].get('worker')}|{sample['checks'].get('optimization_queue')}"
        for sample in ready_samples
    )
    queue_samples = [sample["queue"] for sample in valid_readiness_samples]
    report = {
        "requests": args.requests,
        "concurrency": args.concurrency,
        "elapsedSeconds": elapsed,
        "submission": {"statuses": statuses, "p95LatencySeconds": p95(submission_latencies), "samples": len(submission_latencies)},
        "replay": {
            "statuses": replay_statuses,
            "p95LatencySeconds": p95(replay_latencies),
            "samples": len(replay_latencies),
            "acknowledgementMatches": sum(result.get("replayMatchesAcknowledgement") is True for result in results),
        },
        "poll": {"p95LatencySeconds": p95(poll_latencies), "samples": len(poll_latencies)},
        "terminalStatuses": terminal_statuses,
        "readiness": {
            "statuses": readiness_statuses,
            "samples": len(ready_samples),
            "validSamples": len(valid_readiness_samples),
            "monitorErrors": monitor_errors,
            "monitorAlive": monitor.is_alive(),
        },
        "queueWorkerEvidence": queue_worker_evidence,
        "queueEvidence": {
            "samples": len(queue_samples),
            "maxDepth": max((sample["depth"] for sample in queue_samples), default=0),
            "maxOldestQueuedAgeSeconds": max((sample.get("oldestQueuedAgeSeconds", 0) or 0 for sample in queue_samples), default=0),
            "maxOldestPendingAgeSeconds": max((sample.get("oldestPendingAgeSeconds", 0) or 0 for sample in queue_samples), default=0),
        },
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True, default=dict) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2, sort_keys=True, default=dict))
    return 0 if capacity_gate_passes(report, args.requests) else 1


if __name__ == "__main__":
    raise SystemExit(main())
