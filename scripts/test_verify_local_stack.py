#!/usr/bin/env python3

# Implements DESIGN-005 RepositoryInterfaces isolated local-stack verification.

import importlib.util
import unittest
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


if __name__ == "__main__":
    unittest.main()
