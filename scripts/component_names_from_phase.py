import sys


def extract_static_aspects(filename):
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
    if len(sys.argv) < 2:
        print("Usage: python extract_aspects.py <filename>")
    else:
        file_path = sys.argv[1]
        result = extract_static_aspects(file_path)
        print(result)

