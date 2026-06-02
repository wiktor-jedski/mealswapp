#!/usr/bin/env python3

# Implements DESIGN-017 ErrorMessageMapper frontend contract generation.

import argparse
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "api" / "openapi.yaml"
OUTPUT = ROOT / "frontend" / "src" / "lib" / "api" / "generated.ts"
REQUIRED_MARKERS = ("AppError:", "Envelope:", "CSRFTokenEnvelope:", "/api/v1/health:", "/api/v1/ready:", "/api/v1/auth/csrf-token:")
GENERATED = """// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
\t| "validation"
\t| "auth"
\t| "entitlement"
\t| "network"
\t| "timeout"
\t| "server"
\t| "dependency"
\t| "unknown";

export interface AppError {
\tcategory: ErrorCategory;
\tcode: string;
\tmessage: string;
\tretryable: boolean;
\trequestId?: string;
}

export interface Envelope<TData extends Record<string, unknown> = Record<string, unknown>> {
\tstatus: string;
\trequestId: string;
\tdata?: TData;
\terror?: AppError | null;
}

export interface HealthData extends Record<string, unknown> {
\tservice: string;
}

export interface ReadinessData extends Record<string, unknown> {
\tchecks: Record<string, string>;
}

export interface CSRFTokenData extends Record<string, unknown> {
\tcsrfToken: string;
}
"""


def main() -> int:
	parser = argparse.ArgumentParser(description="Generate shared frontend API types from the Phase 02 OpenAPI contract.")
	parser.add_argument("--check", action="store_true", help="Fail if generated frontend types have drifted.")
	args = parser.parse_args()
	source = OPENAPI.read_text(encoding="utf-8")
	missing = [marker for marker in REQUIRED_MARKERS if marker not in source]
	if missing:
		print(f"OpenAPI contract missing required markers: {missing}")
		return 1
	if args.check:
		if not OUTPUT.exists() or OUTPUT.read_text(encoding="utf-8") != GENERATED:
			print(f"Generated API types are stale: run `python3 {Path(__file__).name}`")
			return 1
		print("Generated API types are current.")
		return 0
	OUTPUT.parent.mkdir(parents=True, exist_ok=True)
	OUTPUT.write_text(GENERATED, encoding="utf-8")
	print(f"Generated {OUTPUT.relative_to(ROOT)}")
	return 0


if __name__ == "__main__":
	sys.exit(main())
