#!/usr/bin/env python3

# Implements DESIGN-005 RepositoryInterfaces isolated local-stack verification.

import importlib.util
import unittest
from subprocess import CompletedProcess
from unittest.mock import patch
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("verify-local-stack.py")
SPEC = importlib.util.spec_from_file_location("local_stack_check", MODULE_PATH)
assert SPEC and SPEC.loader
local_stack = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(local_stack)


class LocalStackDatabaseIsolationTests(unittest.TestCase):
    def test_destructive_migration_cycle_uses_dedicated_test_database(self):
        self.assertIn("/mealswapp_test?", local_stack.DATABASE_URL)
        self.assertEqual(local_stack.backend_env()["MEALSWAPP_DATABASE_URL"], local_stack.DATABASE_URL)

    def test_missing_test_database_is_created_through_compose_postgres(self):
        calls = []

        def fake_run(command, cwd=local_stack.ROOT, env=None, capture=False):
            calls.append(command)
            return CompletedProcess(command, 0, stdout="" if capture else None, stderr=None)

        with patch.object(local_stack, "run", side_effect=fake_run):
            local_stack.ensure_test_database()

        self.assertEqual(calls, [
            ["docker", "compose", "exec", "-T", "postgres", "psql", "-U", "mealswapp", "-d", "postgres", "-tAc", "SELECT 1 FROM pg_database WHERE datname = 'mealswapp_test'"],
            ["docker", "compose", "exec", "-T", "postgres", "createdb", "-U", "mealswapp", "mealswapp_test"],
        ])

    def test_existing_test_database_is_not_recreated(self):
        with patch.object(
            local_stack,
            "run",
            return_value=CompletedProcess([], 0, stdout="1\n", stderr=None),
        ) as run:
            local_stack.ensure_test_database()

        run.assert_called_once()


if __name__ == "__main__":
    unittest.main()
