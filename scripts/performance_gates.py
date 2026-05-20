#!/usr/bin/env python3
from __future__ import annotations

import gzip
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "frontend" / "dist"

MAX_INITIAL_JS_GZIP_BYTES = 150 * 1024
MAX_INITIAL_CSS_GZIP_BYTES = 30 * 1024
MAX_INITIAL_TOTAL_GZIP_BYTES = 200 * 1024


def main() -> int:
    assets_dir = DIST / "assets"
    if not assets_dir.is_dir():
        raise SystemExit("frontend/dist/assets is missing; run frontend production build before performance gates")

    js_gzip = sum(gzip_size(path) for path in assets_dir.glob("*.js"))
    css_gzip = sum(gzip_size(path) for path in assets_dir.glob("*.css"))
    total_gzip = js_gzip + css_gzip + sum(gzip_size(path) for path in DIST.glob("*.html"))

    assert_budget("initial JS gzip", js_gzip, MAX_INITIAL_JS_GZIP_BYTES)
    assert_budget("initial CSS gzip", css_gzip, MAX_INITIAL_CSS_GZIP_BYTES)
    assert_budget("initial total gzip", total_gzip, MAX_INITIAL_TOTAL_GZIP_BYTES)

    print(f"initial_js_gzip_bytes={js_gzip}")
    print(f"initial_css_gzip_bytes={css_gzip}")
    print(f"initial_total_gzip_bytes={total_gzip}")
    return 0


def gzip_size(path: Path) -> int:
    return len(gzip.compress(path.read_bytes(), compresslevel=9))


def assert_budget(label: str, actual: int, budget: int) -> None:
    if actual > budget:
        raise SystemExit(f"{label} exceeded budget: {actual} > {budget}")


if __name__ == "__main__":
    raise SystemExit(main())
