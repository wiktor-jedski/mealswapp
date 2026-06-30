#!/usr/bin/env python3

# Implements DESIGN-001 SearchView and DESIGN-016 LayoutGrid frontend UAT verification.

import contextlib
import argparse
import os
import re
import shutil
import socket
import subprocess
import sys
import time
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FRONTEND = ROOT / "frontend"
ARTIFACT_DIR = Path("/tmp/mealswapp-frontend-verifier")
REQUIRED_TEXT = (
	"Mealswapp",
	"Catalog",
	"Substitution",
	"Daily Diet Alternative",
	"Food search",
)
SEARCH_INPUT_RE = re.compile(r'<input[^>]*id="autocomplete-input"[^>]*>')


def free_port() -> int:
	with contextlib.closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
		sock.bind(("127.0.0.1", 0))
		return int(sock.getsockname()[1])


def frontend_env() -> dict[str, str]:
	return {
		**os.environ,
		"BUN_TMPDIR": str(FRONTEND / ".bun-tmp"),
		"BUN_INSTALL": str(FRONTEND / ".bun-install"),
	}


def start_vite(port: int) -> subprocess.Popen[str]:
	print(f"+ bun run dev -- --host 127.0.0.1 --port {port} --strictPort")
	return subprocess.Popen(
		["bun", "run", "dev", "--", "--host", "127.0.0.1", "--port", str(port), "--strictPort"],
		cwd=FRONTEND,
		text=True,
		env=frontend_env(),
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT,
	)


def stop_process(process: subprocess.Popen[str]) -> None:
	if process.poll() is not None:
		return
	process.terminate()
	try:
		process.wait(timeout=10)
	except subprocess.TimeoutExpired:
		process.kill()
		process.wait(timeout=5)


def wait_for_http(url: str, timeout: float = 30.0) -> None:
	deadline = time.monotonic() + timeout
	last_error: Exception | None = None
	while time.monotonic() < deadline:
		try:
			with urllib.request.urlopen(url, timeout=2) as response:
				if response.status == 200:
					return
		except OSError as exc:
			last_error = exc
		time.sleep(0.5)
	raise TimeoutError(f"{url} did not respond within {timeout:.0f}s: {last_error}")


def chromium() -> str:
	for candidate in ("chromium", "chromium-browser", "google-chrome"):
		path = shutil.which(candidate)
		if path:
			return path
	playwright_browser = playwright_chromium()
	if playwright_browser:
		return playwright_browser
	raise RuntimeError("chromium, chromium-browser, or google-chrome is required for frontend verification")


def playwright_chromium() -> str | None:
	script = "import { chromium } from './node_modules/playwright/index.mjs'; console.log(chromium.executablePath());"
	try:
		result = subprocess.run(
			["node", "--input-type=module", "-e", script],
			cwd=FRONTEND,
			check=True,
			text=True,
			capture_output=True,
			env=frontend_env(),
		)
	except (OSError, subprocess.CalledProcessError):
		return None
	path = result.stdout.strip()
	if path and Path(path).exists():
		return path
	return None


def run_chromium(command: list[str], capture: bool = False) -> subprocess.CompletedProcess[str]:
	print(f"+ {' '.join(command)}")
	return subprocess.run(command, check=True, text=True, capture_output=capture)


def rendered_dom(browser: str, url: str) -> str:
	result = run_chromium(
		[
			browser,
			"--headless=new",
			"--disable-gpu",
			"--no-sandbox",
			"--virtual-time-budget=10000",
			"--dump-dom",
			url,
		],
		capture=True,
	)
	return result.stdout


def assert_shell_dom(dom: str) -> None:
	missing = [text for text in REQUIRED_TEXT if text not in dom]
	if missing:
		raise RuntimeError(f"rendered shell is missing expected text: {', '.join(missing)}")
	search_input = SEARCH_INPUT_RE.search(dom)
	if search_input is None:
		raise RuntimeError("rendered shell is missing the autocomplete search input")
	if "disabled" in search_input.group(0):
		raise RuntimeError("autocomplete search input must not be disabled in Phase 05")
	if 'aria-label="Theme preference"' not in dom:
		raise RuntimeError("rendered shell is missing the theme selector")
	if "data-sidebar-theme-toggle" not in dom:
		raise RuntimeError("rendered shell is missing the sidebar theme toggle")


def capture_screenshot(browser: str, url: str, artifact_dir: Path, name: str, width: int, height: int) -> Path:
	artifact_dir.mkdir(parents=True, exist_ok=True)
	output = artifact_dir / name
	run_chromium(
		[
			browser,
			"--headless=new",
			"--disable-gpu",
			"--no-sandbox",
			f"--window-size={width},{height}",
			"--virtual-time-budget=10000",
			f"--screenshot={output}",
			url,
		]
	)
	if not output.exists() or output.stat().st_size < 1000:
		raise RuntimeError(f"screenshot was not captured correctly: {output}")
	return output


def capture_scenario_screenshots(url: str, artifact_dir: Path, screenshot_stem: str) -> None:
	script = ROOT / "scripts" / "capture-frontend-scenarios.mjs"
	print(f"+ node {script} {url} {artifact_dir} {screenshot_stem}")
	subprocess.run(
		["node", str(script), url, str(artifact_dir), screenshot_stem],
		cwd=ROOT,
		check=True,
		text=True,
		env=frontend_env(),
	)


def main() -> int:
	parser = argparse.ArgumentParser(description="Verify the Mealswapp frontend shell and capture UAT screenshots.")
	parser.add_argument("--artifact-dir", default=str(ARTIFACT_DIR), help="Directory for generated screenshots.")
	parser.add_argument("--screenshot-stem", default="frontend-verification", help="Filename stem for screenshot artifacts.")
	args = parser.parse_args()

	artifact_dir = Path(args.artifact_dir)
	screenshot_stem = Path(args.screenshot_stem).name
	port = free_port()
	url = f"http://127.0.0.1:{port}"
	process = start_vite(port)
	try:
		wait_for_http(url)
		browser = chromium()
		assert_shell_dom(rendered_dom(browser, url))
		desktop = capture_screenshot(browser, url, artifact_dir, f"{screenshot_stem}-desktop.png", 1280, 900)
		mobile = capture_screenshot(browser, url, artifact_dir, f"{screenshot_stem}-mobile.png", 390, 844)
		capture_scenario_screenshots(url, artifact_dir, screenshot_stem)
		print(f"Frontend verification passed. Screenshots: {desktop}, {mobile}")
		return 0
	finally:
		stop_process(process)


if __name__ == "__main__":
	sys.exit(main())
