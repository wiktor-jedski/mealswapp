// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
	| "validation"
	| "auth"
	| "entitlement"
	| "network"
	| "timeout"
	| "server"
	| "dependency"
	| "unknown";

export interface AppError {
	category: ErrorCategory;
	code: string;
	message: string;
	retryable: boolean;
	requestId?: string;
}

export interface Envelope<TData extends Record<string, unknown> = Record<string, unknown>> {
	status: string;
	requestId: string;
	data?: TData;
	error?: AppError | null;
}

export interface HealthData extends Record<string, unknown> {
	service: string;
}

export interface ReadinessData extends Record<string, unknown> {
	checks: Record<string, string>;
}

export interface CSRFTokenData extends Record<string, unknown> {
	csrfToken: string;
}
