#!/usr/bin/env python3

# Implements DESIGN-010 RouteHandler local development process lifecycle regression test.

import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PROCESS_HELPERS = ROOT / "scripts" / "dev-processes.sh"


class StartDevProcessTests(unittest.TestCase):
    def test_cleanup_stops_child_process_spawned_by_managed_command(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            child_pid_file = Path(temporary_directory) / "child.pid"
            script = r'''
                set -euo pipefail
                source "$1"
                leader_pid=""
                trap 'stop_dev_process "$leader_pid"' EXIT
                start_dev_process "$2" bash -c '
                    sleep 300 &
                    child_pid=$!
                    printf "%s\n" "$child_pid" > "$1"
                    wait "$child_pid"
                ' _ "$3"
                leader_pid=$DEV_PROCESS_PID

                for _ in {1..100}; do
                    [[ -s "$3" ]] && break
                    sleep 0.01
                done
                child_pid=$(<"$3")
                [[ "$(ps -o pgid= -p "$leader_pid" | tr -d " ")" == "$leader_pid" ]]
                [[ "$(ps -o pgid= -p "$child_pid" | tr -d " ")" == "$leader_pid" ]]

                stop_dev_process "$leader_pid"
                wait "$leader_pid" 2>/dev/null || true

                for _ in {1..100}; do
                    child_state=$(ps -o stat= -p "$child_pid" 2>/dev/null || true)
                    [[ -z "$child_state" || "$child_state" == Z* ]] && exit 0
                    sleep 0.01
                done
                exit 1
            '''

            subprocess.run(
                ["bash", "-c", script, "_", str(PROCESS_HELPERS), str(ROOT), str(child_pid_file)],
                check=True,
                timeout=5,
            )


if __name__ == "__main__":
    unittest.main()
