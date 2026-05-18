import re
import sys
from collections import Counter
from pathlib import Path

requirements = Path("docs/requirements/01_SOFT_REQ_SPEC.md").read_text()
architecture = Path("docs/architecture/01_SOFT_ARCH_DESIGN.md").read_text()

req_list = re.findall(r"^## \[(SW-REQ-\d+)\]", requirements, re.MULTILINE)
missing = []

for req, count in sorted(Counter(req_list).items()):
    if count > 1:
        print(f"{req} DUPLICATE")

for req in sorted(set(req_list)):
    if req not in architecture:
        missing.append(req)
        print(f"{req} MISSING")

if missing or any(count > 1 for count in Counter(req_list).values()):
    sys.exit(1)
