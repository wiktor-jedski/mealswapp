#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector capacity-gate verification.

import importlib.util
import unittest
from collections import Counter
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("verify-optimization-capacity.py")
SPEC = importlib.util.spec_from_file_location("capacity_check", MODULE_PATH)
assert SPEC and SPEC.loader
capacity = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(capacity)


class CapacityGateTests(unittest.TestCase):
    def test_p95_uses_ceiling_rank_at_default_32_sample_boundary(self):
        self.assertEqual(capacity.p95([0.1] * 30 + [2.1, 2.2]), 2.1)

    def test_p95_latency_violation_at_threshold_fails_gate(self):
        report = self.valid_report()
        report["submission"]["p95LatencySeconds"] = 2.0
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_rejects_absent_readiness_and_queue_evidence(self):
        report = self.valid_report()
        report["readiness"].update({"samples": 0, "validSamples": 0})
        report["queueEvidence"]["samples"] = 0
        report["queueWorkerEvidence"] = Counter()
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_rejects_continuously_degraded_readiness(self):
        report = self.valid_report()
        report["readiness"].update({"samples": 4, "validSamples": 0})
        report["readiness"]["statuses"] = Counter({"503": 4})
        report["queueEvidence"]["samples"] = 0
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_rejects_malformed_readiness_evidence(self):
        malformed = {
            "status": 200,
            "checks": {"redis": "ok", "worker": "ok", "optimization_queue": "ok"},
            "queue": {"depth": 1, "oldestQueuedAgeSeconds": float("nan"), "oldestPendingAgeSeconds": 0},
        }
        self.assertFalse(capacity.readiness_sample_is_valid(malformed))
        report = self.valid_report()
        report["readiness"].update({"samples": 1, "validSamples": 0})
        report["queueEvidence"]["samples"] = 0
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_rejects_readiness_monitor_failure(self):
        report = self.valid_report()
        report["readiness"]["monitorErrors"] = ["URLError"]
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_rejects_missing_poll_samples(self):
        report = self.valid_report()
        report["poll"]["samples"] = 0
        self.assertFalse(capacity.capacity_gate_passes(report, 32))

    def test_gate_accepts_complete_readiness_and_queue_worker_evidence(self):
        self.assertTrue(capacity.capacity_gate_passes(self.valid_report(), 32))

    @staticmethod
    def valid_report():
        return {
            "submission": {"statuses": Counter({"202": 32}), "p95LatencySeconds": 1.1, "samples": 32},
            "poll": {"p95LatencySeconds": 1.2, "samples": 32},
            "readiness": {
                "statuses": Counter({"200": 2}),
                "samples": 2,
                "validSamples": 2,
                "monitorErrors": [],
                "monitorAlive": False,
            },
            "queueWorkerEvidence": Counter({"ok|ok|ok": 2}),
            "queueEvidence": {"samples": 2},
        }


if __name__ == "__main__":
    unittest.main()
