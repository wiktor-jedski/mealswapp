import subprocess
import re
import sys
import os
from typing import List
from pathlib import Path


def design_prompt(arch_id: str, static_asset: str) -> str:
    return f"""read docs/design/designer-prompt.md
ARCH-ID: {arch_id} - docs/architecture/{arch_id}.md
generate document docs/design/{arch_id}/{static_asset}.md"""


def planner_prompt(
    phase_plan_path: str, static_aspect_design_path: str, task_list_path: str
) -> str:
    return f"""read docs/implementation/planner-prompt.md
PHASE_PLAN: {phase_plan_path}
STATIC_ASPECT_DESIGN: {static_aspect_design_path}
TASK_LIST: {task_list_path}"""


def planner_ralph(
    phase_plan_path: str,
    task_list_path: str,
    static_aspects: List[str] | None,
):
    if static_aspects is None:
        return
    for static_aspect in static_aspects:
        try:
            subprocess.run(
                [
                    "opencode",
                    "run",
                    planner_prompt(phase_plan_path, static_aspect, task_list_path),
                    "--agent",
                    "build",
                ]
            )
        except Exception as e:
            print(f"Error while running agent: {e}")
            continue

    return


def static_asset_ralph(file_path: str, static_assets: List[str] | None) -> None:
    if static_assets is None:
        return
    arch_id = Path(file_path).stem
    for static_asset in static_assets:
        try:
            subprocess.run(
                [
                    "opencode",
                    "run",
                    design_prompt(arch_id=arch_id, static_asset=static_asset),
                    "--agent",
                    "build",
                ]
            )
        except Exception as e:
            print(f"Error while running agent: {e}")
            continue

    return


def extract_static_aspects_from_arch(file_path):
    if not os.path.exists(file_path):
        print(f"Error: The file '{file_path}' was not found.")
        return

    try:
        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading file: {e}")
        return

    extracted_components = []

    # Regex Explanation:
    # \|                     -> Matches the starting pipe of the table row
    # \s*                    -> Matches optional whitespace
    # \*\*Static Aspects\*\* -> Matches the specific label literal text
    # \s*                    -> Matches optional whitespace
    # \|                     -> Matches the middle separator pipe
    # \s*                    -> Matches optional whitespace
    # ([^|]+)                -> CAPTURE GROUP 1: Matches everything that is NOT a pipe
    # \s*                    -> Matches optional trailing whitespace
    # \|                     -> Matches the closing pipe of the table row
    pattern = r"\|\s*\*\*Static Aspects\*\*\s*\|\s*([^|]+)\s*\|"

    # findall returns a list of the contents in Capture Group 1
    matches = re.findall(pattern, content)

    for match in matches:
        # 'match' is now a string like: "SearchController, AutocompleteRanker, ..."

        # Split by comma
        items = match.split(",")

        # Clean whitespace and add to list
        for item in items:
            clean_item = item.strip()
            if clean_item:
                extracted_components.append(clean_item)

    return extracted_components


def extract_static_aspects_from_phase(filename):
    aspects = []
    target_header = "Static Aspect"

    try:
        with open(filename, "r", encoding="utf-8") as f:
            lines = f.readlines()

        target_col_index = -1
        inside_target_table = False

        for line in lines:
            stripped_line = line.strip()

            # Check if we are finding a header row containing "Static Aspect"
            if target_header in stripped_line and "|" in stripped_line:
                # Split line by pipe
                parts = [p.strip() for p in stripped_line.split("|")]

                # Find the index of the column.
                # Note: valid markdown tables often start with | so index 0 is empty string
                if target_header in parts:
                    target_col_index = parts.index(target_header)
                    inside_target_table = True
                continue

            # If we are currently processing a relevant table
            if inside_target_table:
                # If the line is empty or doesn't start with |, the table has ended
                if not stripped_line.startswith("|"):
                    inside_target_table = False
                    target_col_index = -1
                    continue

                # Skip the separator line (e.g., |:---|---|)
                if "---" in stripped_line:
                    continue

                # Extract the data
                parts = [p.strip() for p in stripped_line.split("|")]

                # Ensure the row has enough columns
                if len(parts) > target_col_index:
                    raw_content = parts[target_col_index]

                    # Clean the content: remove ** bold markers and whitespace
                    clean_content = raw_content.replace("*", "").strip()

                    if clean_content:
                        aspects.append(clean_content)

        return aspects

    except FileNotFoundError:
        print(f"Error: The file '{filename}' was not found.")
        return []
    except Exception as e:
        print(f"An error occurred: {e}")
        return []


if __name__ == "__main__":
    # Check if filename argument is provided
    if len(sys.argv) < 2:
        print("Usage: python extract_components.py <filename>")
    else:
        input_file = sys.argv[1]
        results = extract_static_aspects_from_phase(input_file)

        if results is not None:
            print(f"Found {len(results)} static aspects:")
            print(results)
        planner_ralph(
            phase_plan_path=sys.argv[1],
            task_list_path=sys.argv[2],
            static_aspects=results,
        )
