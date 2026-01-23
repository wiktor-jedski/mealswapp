import sys
import re
import os


def clean_filename(title):
    """
    Converts a title string into a filename-safe string.
    Replaces spaces/symbols with underscores.
    """
    # Replace non-alphanumeric characters (excluding spaces) with empty string
    clean = re.sub(r"[^\w\s-]", "", title)
    # Replace spaces with underscores
    clean = re.sub(r"[-\s]+", "_", clean)
    return clean.strip("_")


def extract_phases(input_filename):
    if not os.path.exists(input_filename):
        print(f"Error: File '{input_filename}' not found.")
        return

    # Regex to identify "## Phase X: Title"
    phase_header_pattern = re.compile(r"^## Phase (\d+): (.+)")
    # Regex to identify any "## Header" (to detect end of phases)
    any_h2_pattern = re.compile(r"^## .+")

    current_file = None

    try:
        with open(input_filename, "r", encoding="utf-8") as f:
            lines = f.readlines()

        print(f"Processing '{input_filename}'...")

        for line in lines:
            phase_match = phase_header_pattern.match(line)
            h2_match = any_h2_pattern.match(line)

            # 1. Check if this line starts a NEW Phase
            if phase_match:
                # Close previous file if open
                if current_file:
                    current_file.close()

                phase_num = phase_match.group(1).zfill(2)  # e.g., "1" -> "01"
                phase_title = phase_match.group(2).strip()
                safe_title = clean_filename(phase_title)

                output_filename = f"{phase_num}_{safe_title}.md"
                print(f"Creating: {output_filename}")

                current_file = open(output_filename, "w", encoding="utf-8")
                current_file.write(line)  # Write the header to the new file
                continue

            # 2. Check if this line starts a section that is NOT a Phase
            # (e.g., "## Verification Checklist")
            if current_file and h2_match and not phase_match:
                print("End of phases detected. Stopping extraction.")
                current_file.close()
                current_file = None
                continue

            # 3. If a file is currently open, write the content line
            if current_file:
                current_file.write(line)

        # Final cleanup
        if current_file:
            current_file.close()

        print("Extraction complete.")

    except Exception as e:
        print(f"An error occurred: {e}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python extract_phases.py <markdown_file>")
    else:
        extract_phases(sys.argv[1])

