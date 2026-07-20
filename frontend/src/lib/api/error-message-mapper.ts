import type { AppError } from "./generated";

// Implements DESIGN-017 ErrorMessageMapper shared runtime-safe client error projection.

/** Generated-contract client domains with an approved error-message policy. */
export type ErrorMessageScope = "daily_diet" | "optimization";

type ErrorRule = Readonly<Omit<AppError, "requestId">>;

const DAILY_DIET_FALLBACKS: Readonly<Record<number, ErrorRule>> = {
	400: rule("validation", "daily_diet_invalid_request", "Saved daily diet request could not be processed. Please review it and try again.", false),
	401: rule("auth", "session_expired", "Your session expired. Please sign in and try again.", false),
	403: rule("security", "daily_diet_unavailable", "Saved daily diet is unavailable.", false),
	404: rule("security", "daily_diet_unavailable", "Saved daily diet is unavailable.", false),
	409: rule("validation", "daily_diet_invalid_request", "Saved daily diet request could not be processed. Please review it and try again.", false),
	422: rule("validation", "daily_diet_invalid_request", "Saved daily diet request could not be processed. Please review it and try again.", false),
	429: rule("rate_limit", "daily_diet_rate_limited", "Too many saved daily-diet requests. Please wait and try again.", true),
	500: rule("server", "internal_error", "Saved daily diets are temporarily unavailable. Please try again.", true),
	503: rule("dependency", "daily_diet_unavailable", "Saved daily diets are temporarily unavailable. Please try again shortly.", true),
	504: rule("timeout", "request_timeout", "The saved daily-diet request took too long. Please try again.", true)
};

const OPTIMIZATION_FALLBACKS: Readonly<Record<number, ErrorRule>> = {
	400: rule("validation", "optimization_invalid_request", "Optimization request could not be processed. Please review it and try again.", false),
	401: rule("auth", "session_expired", "Your session expired. Please sign in and try again.", false),
	403: rule("entitlement", "entitlement_denied", "An active trial or paid subscription is required for optimization.", false),
	404: rule("validation", "optimization_not_found", "This optimization is no longer available. Please submit again.", true),
	409: rule("validation", "optimization_invalid_request", "Optimization request could not be processed. Please review it and try again.", false),
	410: rule("validation", "result_expired", "This optimization result has expired. Submit again for a fresh result.", true),
	422: rule("validation", "solver_infeasible", "No meal combination matched these macro targets. Try a wider tolerance.", false),
	429: rule("rate_limit", "optimization_rate_limited", "Too many optimization requests. Please wait and try again.", true),
	500: rule("server", "internal_error", "Optimization could not be completed. Please try again.", true),
	503: rule("dependency", "queue_unavailable", "The optimization queue is temporarily unavailable. Please try again.", true),
	504: rule("timeout", "request_timeout", "Optimization took too long. Please try again.", true)
};

const APPROVED_RULES: Readonly<Record<string, ErrorRule>> = Object.fromEntries([
	...approved("daily_diet", 400, DAILY_DIET_FALLBACKS[400]!, ["invalid_json", "validation_failed", "idempotency_key_required"]),
	...approved("daily_diet", 401, DAILY_DIET_FALLBACKS[401]!, ["unauthorized", "session_expired"]),
	...approved("daily_diet", 409, DAILY_DIET_FALLBACKS[409]!, ["conflict", "idempotency_key_conflict"]),
	...approved("daily_diet", 429, DAILY_DIET_FALLBACKS[429]!, ["daily_diet_rate_limited"]),
	...approved("daily_diet", 500, DAILY_DIET_FALLBACKS[500]!, ["internal_error"]),
	...approved("daily_diet", 503, DAILY_DIET_FALLBACKS[503]!, ["daily_diet_unavailable"]),
	...approved("daily_diet", 504, DAILY_DIET_FALLBACKS[504]!, ["request_timeout"]),
	...approved("optimization", 400, OPTIMIZATION_FALLBACKS[400]!, ["invalid_json", "validation_failed", "idempotency_key_required"]),
	...approved("optimization", 401, OPTIMIZATION_FALLBACKS[401]!, ["unauthorized", "session_expired"]),
	...approved("optimization", 403, OPTIMIZATION_FALLBACKS[403]!, ["entitlement_denied"]),
	...approved("optimization", 404, OPTIMIZATION_FALLBACKS[404]!, ["not_found", "optimization_not_found"]),
	...approved("optimization", 409, OPTIMIZATION_FALLBACKS[409]!, ["idempotency_key_conflict"]),
	...approved("optimization", 410, OPTIMIZATION_FALLBACKS[410]!, ["result_expired"]),
	...approved("optimization", 422, OPTIMIZATION_FALLBACKS[422]!, ["solver_infeasible"]),
	...approved("optimization", 429, OPTIMIZATION_FALLBACKS[429]!, ["optimization_rate_limited"]),
	...approved("optimization", 500, OPTIMIZATION_FALLBACKS[500]!, ["internal_error"]),
	...approved("optimization", 503, OPTIMIZATION_FALLBACKS[503]!, ["optimization_unavailable", "queue_unavailable"]),
	...approved("optimization", 504, OPTIMIZATION_FALLBACKS[504]!, ["request_timeout"])
]);

/** Maps an untrusted API error envelope to fixed, user-safe client text. */
export function mapErrorMessage(scope: ErrorMessageScope, status: number, envelope: unknown): AppError {
	const fallback = fallbackFor(scope, status);
	const body = isObject(envelope) ? envelope : undefined;
	const source = body && isObject(body.error) ? body.error : undefined;

	// Daily Diet intentionally makes forbidden and missing resources indistinguishable.
	const approvedError = scope === "daily_diet" && (status === 403 || status === 404)
		? undefined
		: approvedSourceError(scope, status, source);
	const mapped = approvedError ?? { ...fallback };
	const requestId = safeRequestId(source?.requestId) ?? safeRequestId(body?.requestId);
	return requestId ? { ...mapped, requestId } : mapped;
}

function approvedSourceError(scope: ErrorMessageScope, status: number, source: Record<string, unknown> | undefined): ErrorRule | undefined {
	if (
		!source ||
		typeof source.category !== "string" ||
		typeof source.code !== "string" ||
		typeof source.message !== "string" ||
		typeof source.retryable !== "boolean"
	) return undefined;
	const approvedRule = APPROVED_RULES[`${scope}:${status}:${source.code}`];
	return approvedRule?.category === source.category
		? { ...approvedRule, code: source.code, retryable: source.retryable }
		: undefined;
}

function fallbackFor(scope: ErrorMessageScope, status: number): ErrorRule {
	const fallback = scope === "daily_diet" ? DAILY_DIET_FALLBACKS[status] : OPTIMIZATION_FALLBACKS[status];
	if (fallback) return fallback;
	return scope === "daily_diet"
		? rule("unknown", "daily_diet_request_failed", "Something went wrong. Please try again.", false)
		: rule("unknown", "optimization_request_failed", "Optimization could not be completed. Please try again.", true);
}

function approved(scope: ErrorMessageScope, status: number, base: ErrorRule, codes: readonly string[]): Array<[string, ErrorRule]> {
	return codes.map((code) => [`${scope}:${status}:${code}`, { ...base, code }]);
}

function rule(category: AppError["category"], code: string, message: string, retryable: boolean): ErrorRule {
	return { category, code, message, retryable };
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function safeRequestId(value: unknown): string | undefined {
	return typeof value === "string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,119}$/.test(value) ? value : undefined;
}
