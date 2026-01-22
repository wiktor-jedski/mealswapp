import re
import sys
import os

def extract_static_aspects(file_path):
    if not os.path.exists(file_path):
        print(f"Error: The file '{file_path}' was not found.")
        return

    try:
        with open(file_path, 'r', encoding='utf-8') as f:
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
    pattern = r'\|\s*\*\*Static Aspects\*\*\s*\|\s*([^|]+)\s*\|'

    # findall returns a list of the contents in Capture Group 1
    matches = re.findall(pattern, content)

    for match in matches:
        # 'match' is now a string like: "SearchController, AutocompleteRanker, ..."
        
        # Split by comma
        items = match.split(',')
        
        # Clean whitespace and add to list
        for item in items:
            clean_item = item.strip()
            if clean_item:
                extracted_components.append(clean_item)

    return extracted_components

if __name__ == "__main__":
    # Check if filename argument is provided
    if len(sys.argv) < 2:
        print("Usage: python extract_components.py <filename>")
    else:
        input_file = sys.argv[1]
        results = extract_static_aspects(input_file)
        
        if results is not None:
            print(f"Found {len(results)} components:")
            print(results)