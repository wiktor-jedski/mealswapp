import subprocess
import time
import json
import re

# --- CONFIGURATION ---
MAX_CONCURRENT_CONTAINERS = 4
RUN_AGENT_SCRIPT = "./run_agent.sh"
REPO_DIR = ".."
MAIN_BRANCH = "phase-01"


def sync_repo():
    """Pulls the latest task list updates from the remote repository."""
    print(f"🔄 Syncing latest task list from origin/{MAIN_BRANCH}...")
    result = subprocess.run(
        ["git", "pull", "origin", MAIN_BRANCH, "--rebase", "--autostash"],
        cwd=REPO_DIR,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"⚠️ Warning: Failed to sync repo: {result.stderr}")
        return False
    return True


def extract_json_from_output(text):
    """Extracts JSON from LLM output, handling markdown formatting."""
    match = re.search(r"```json\s*(\{.*?\})\s*```", text, re.DOTALL)
    if match:
        return json.loads(match.group(1))
    # Fallback: try parsing the whole text if no markdown tags
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return None


def get_next_task(active_tasks):
    """
    Uses opencode to read the task list and determine the next best action.
    active_tasks: list of string IDs currently being worked on (e.g. ['PHASE1-TASK1'])
    """
    print("Asking Orchestrator for the next task")

    # Format active tasks so the AI knows what to ignore
    ignore_list = ", ".join(active_tasks) if active_tasks else "None"

    prompt = f"""read docs/implementation/orchestrator-prompt.md
    IGNORE_LIST = [{ignore_list}]
    PHASE-ID = phase-01
    """

    # Run opencode as a subprocess to analyze the task list
    result = subprocess.run(
        ["opencode", "run", prompt], capture_output=True, text=True, cwd=REPO_DIR
    )

    if result.returncode != 0:
        print(f"Error calling opencode orchestrator: {result.stderr}")
        return None

    return extract_json_from_output(result.stdout)


def main():
    # Dictionary to keep track of running processes: {"PARENT-CHILD": subprocess.Popen}
    active_processes = {}

    print(
        f"Starting AI Dev Orchestrator (Max concurrent tasks: {MAX_CONCURRENT_CONTAINERS})..."
    )

    while True:
        # 1. Clean up finished containers
        finished_tasks = []
        for task_key, proc in active_processes.items():
            if proc.poll() is not None:  # Process has terminated
                print(
                    f"✅ Container for {task_key} finished with exit code {proc.returncode}"
                )
                finished_tasks.append(task_key)

        for task_key in finished_tasks:
            del active_processes[task_key]

        # 2. Spawn new containers if we have capacity
        if len(active_processes) < MAX_CONCURRENT_CONTAINERS:
            sync_repo()

            # Fetch next task from AI
            active_tasks = list(active_processes.keys())
            print(f"Current active tasks: {active_tasks}")
            next_task = get_next_task(active_tasks)

            if next_task and next_task.get("child"):
                parent = next_task["parent"]
                child = next_task["child"]
                action = next_task["action"]
                task_key = f"{parent}-{child}"

                if task_key not in active_processes:
                    print(f"Spawning container: {task_key} | Action: {action}")

                    # Spawn the background process (non-blocking)
                    proc = subprocess.Popen(
                        [RUN_AGENT_SCRIPT, parent, child, action],
                        stdout=subprocess.DEVNULL,  # Optionally redirect to a log file per task
                        stderr=subprocess.STDOUT,
                    )

                    active_processes[task_key] = proc
                else:
                    # Failsafe: AI suggested a task already in the queue
                    print(f"AI suggested already active task {task_key}. Waiting...")
                    time.sleep(10)
            else:
                print(
                    "No actionable tasks right now. Waiting for current tasks to finish or dependencies to resolve..."
                )
                time.sleep(15)  # Wait a bit before asking AI again to save tokens
        else:
            # We are at max capacity
            time.sleep(10)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n🛑 Shutting down orchestrator...")
