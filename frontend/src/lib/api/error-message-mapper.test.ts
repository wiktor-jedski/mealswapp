import { expect, test } from "bun:test";

import { mapErrorMessage } from "./error-message-mapper";

// Implements DESIGN-017 ErrorMessageMapper runtime-safe unknown-envelope verification.

test("unknown Daily Diet envelopes use a fixed status fallback", () => {
	expect(mapErrorMessage("daily_diet", 503, { error: "redis://secret" })).toEqual({
		category: "dependency",
		code: "daily_diet_unavailable",
		message: "Saved daily diets are temporarily unavailable. Please try again shortly.",
		retryable: true
	});
	expect(mapErrorMessage("daily_diet", 418, null)).toEqual({
		category: "unknown",
		code: "daily_diet_request_failed",
		message: "Something went wrong. Please try again.",
		retryable: false
	});
	expect(mapErrorMessage("optimization", 418, [])).toEqual({
		category: "unknown",
		code: "optimization_request_failed",
		message: "Optimization could not be completed. Please try again.",
		retryable: true
	});
});

test("approved error codes retain fixed policy through one table-driven mapper", () => {
	const cases = [
		{
			scope: "daily_diet",
			status: 400,
			error: { category: "validation", code: "validation_failed", retryable: false },
			expected: { category: "validation", code: "validation_failed", message: "Saved daily diet request could not be processed. Please review it and try again.", retryable: false }
		},
		{
			scope: "optimization",
			status: 401,
			error: { category: "auth", code: "unauthorized", retryable: false },
			expected: { category: "auth", code: "unauthorized", message: "Your session expired. Please sign in and try again.", retryable: false }
		},
		{
			scope: "optimization",
			status: 403,
			error: { category: "entitlement", code: "entitlement_denied", retryable: false },
			expected: { category: "entitlement", code: "entitlement_denied", message: "An active trial or paid subscription is required for optimization.", retryable: false }
		},
		{
			scope: "daily_diet",
			status: 503,
			error: { category: "dependency", code: "daily_diet_unavailable", retryable: true },
			expected: { category: "dependency", code: "daily_diet_unavailable", message: "Saved daily diets are temporarily unavailable. Please try again shortly.", retryable: true }
		},
		{
			scope: "optimization",
			status: 429,
			error: { category: "rate_limit", code: "optimization_rate_limited", retryable: true },
			expected: { category: "rate_limit", code: "optimization_rate_limited", message: "Too many optimization requests. Please wait and try again.", retryable: true }
		},
		{
			scope: "optimization",
			status: 504,
			error: { category: "timeout", code: "request_timeout", retryable: true },
			expected: { category: "timeout", code: "request_timeout", message: "Optimization took too long. Please try again.", retryable: true }
		}
	] as const;

	for (const item of cases) {
		expect(mapErrorMessage(item.scope, item.status, {
			status: "error",
			requestId: `req-${item.status}`,
			error: { ...item.error, message: "postgres redis provider credential=https://internal" }
		})).toEqual({ ...item.expected, requestId: `req-${item.status}` });
	}
});

test("malformed fields and hostile text never cross the mapper boundary", () => {
	const malformed = [
		{ category: "database", code: "queue_unavailable", message: "safe", retryable: true },
		{ category: "dependency", code: "redis_secret", message: "safe", retryable: true },
		{ category: "dependency", code: "queue_unavailable", message: null, retryable: true },
		{ category: "dependency", code: "queue_unavailable", message: "safe", retryable: "true" }
	];
	for (const error of malformed) {
		expect(mapErrorMessage("optimization", 503, { requestId: "bad request id", error })).toEqual({
			category: "dependency",
			code: "queue_unavailable",
			message: "The optimization queue is temporarily unavailable. Please try again.",
			retryable: true
		});
	}
	expect(mapErrorMessage("optimization", 400, {
		error: { category: "validation", code: "validation_failed", message: "safe", retryable: "false" }
	})).toEqual({
		category: "validation",
		code: "optimization_invalid_request",
		message: "Optimization request could not be processed. Please review it and try again.",
		retryable: false
	});

	for (const message of [
		"Error at worker.ts:42\npostgres redis stripe api_key=secret",
		"https://internal.example/provider?credential=secret",
		"x".repeat(10_000),
		"control\u0000character"
	]) {
		const mapped = mapErrorMessage("daily_diet", 400, {
			requestId: "x".repeat(121),
			error: { category: "validation", code: "validation_failed", message, retryable: false, requestId: "req\nsecret" }
		});
		expect(mapped).toEqual({
			category: "validation",
			code: "validation_failed",
			message: "Saved daily diet request could not be processed. Please review it and try again.",
			retryable: false
		});
	}
});

test("request IDs accept only bounded printable correlation tokens", () => {
	const accepted = `r${"x".repeat(119)}`;
	expect(mapErrorMessage("optimization", 503, { requestId: accepted })).toMatchObject({ requestId: accepted });
	for (const requestId of ["", `r${"x".repeat(120)}`, "request id", "request\nsecret", "request\u0000secret"]) {
		expect(mapErrorMessage("optimization", 503, { requestId }).requestId).toBeUndefined();
	}
});

test("Daily Diet 403 and 404 remain ownership-safe and indistinguishable", () => {
	for (const status of [403, 404]) {
		expect(mapErrorMessage("daily_diet", status, {
			requestId: `req-${status}`,
			error: { category: "security", code: "cross_user_access", message: "another user's diet", retryable: true }
		})).toEqual({
			category: "security",
			code: "daily_diet_unavailable",
			message: "Saved daily diet is unavailable.",
			retryable: false,
			requestId: `req-${status}`
		});
	}
});
