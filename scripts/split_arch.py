import os
import re
import sys
import argparse


def extract_arch_components(input_file):
    if not os.path.exists(input_file):
        print(f"Error: File '{input_file}' not found.")
        sys.exit(1)

    try:
        with open(input_file, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading file: {e}")
        sys.exit(1)

    # Directory to store the files
    output_dir = "docs/architecture/"
    os.makedirs(output_dir, exist_ok=True)

    # Regex explanation:
    # 1. ^## \[(ARCH-\d+)\] - (.+)  -> Matches start of line ## [ARCH-ID] - Title
    # 2. ([\s\S]*?)                 -> Captures the body (non-greedy)
    # 3. (?=\n## \[ARCH|\n## \d+\.|\Z) -> Lookahead: Stops capturing when it hits:
    #                                     - The next ARCH component
    #                                     - A numbered section (e.g., ## 3. Interface...)
    #                                     - End of file
    pattern = re.compile(
        r"^## \[(ARCH-\d+)\] - (.+)\n([\s\S]*?)(?=\n## \[ARCH|\n## \d+\.|\Z)",
        re.MULTILINE,
    )

    matches = pattern.findall(content)

    if not matches:
        print("No [ARCH-XXX] components found. Check formatting.")
        print("Expected format: ## [ARCH-001] - Component Name")
        return

    print(f"Found {len(matches)} components. Processing...")

    for arch_id, title, body in matches:
        # Create a clean Markdown string for the component
        file_content = f"# [{arch_id}] - {title}\n{body.rstrip()}"

        # Remove any trailing "---" if present from the split
        if file_content.strip().endswith("---"):
            file_content = file_content.strip()[:-3].strip()

        filename = f"{arch_id}.md"
        output_path = os.path.join(output_dir, filename)

        try:
            with open(output_path, "w", encoding="utf-8") as out_f:
                out_f.write(file_content)
                # Ensure a newline at end of file
                out_f.write("\n")
            print(f"✓ Created {output_path}")
        except IOError as e:
            print(f"Error writing {filename}: {e}")

    print(f"\nSuccess! Extracted {len(matches)} files to directory: ./{output_dir}")


def main():
    parser = argparse.ArgumentParser(
        description="Extract ARCH components from a markdown file."
    )
    parser.add_argument(
        "filename", nargs="?", help="The path to the markdown file (e.g., design.md)"
    )

    args = parser.parse_args()

    target_file = args.filename

    # If no argument passed, ask interactively
    if not target_file:
        target_file = input("Please enter the path to the markdown file: ").strip()

    # Remove quotes if user dragged/dropped file in terminal
    target_file = target_file.strip("'\"")

    extract_arch_components(target_file)


if __name__ == "__main__":
    main()

