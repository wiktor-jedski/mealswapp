import type { CreateQueryOptions } from "@tanstack/svelte-query";
import {
	type AppError,
	type CheckoutRequest,
	type CheckoutSessionData,
	type CheckoutSessionEnvelope,
	type EntitlementData,
	type EntitlementEnvelope,
	type Envelope,
	createIdempotencyHeader
} from "./generated";

// Implements DESIGN-001 SearchView TanStack Query entitlement client over generated envelopes.
// Implements DESIGN-017 ErrorMessageMapper safe AppError mapping for 401/402/409/503 envelopes.

/** Base path of the GET entitlement endpoint. */
const ENTITLEMENT_ENDPOINT = "/api/v1/entitlements";

/** Base path of the POST checkout endpoint. */
const CHECKOUT_ENDPOINT = "/api/v1/billing/checkout";

/** Request budget after which the client aborts and surfaces a retryable timeout. */
export const ENTITLEMENT_TIMEOUT_MS = 10_000;

/** Stable query-key namespace prefix so entitlement keys never collide with other queries. */
const ENTITLEMENT_QUERY_NAMESPACE = "entitlement" as const;

/** TanStack query key shape for entitlement queries. */
export type EntitlementQueryKey = readonly [typeof ENTITLEMENT_QUERY_NAMESPACE];

/**
 * Error thrown by the entitlement client when the API returns a non-2xx envelope or the
 * request times out. Carries a user-safe {@link AppError} so callers can render
 * classified messages without touching the raw network exception.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper client error boundary contract.
 */
export class EntitlementClientError extends Error {
	readonly appError: AppError;
	readonly status: number;

	constructor(appError: AppError, status: number) {
		super(appError.message);
		this.name = "EntitlementClientError";
		this.appError = appError;
		this.status = status;
	}
}

/**
 * Maps a server envelope error and HTTP status to a user-safe {@link AppError}, preserving
 * `requestId` and `retryable` from the server while never leaking stack traces or URLs.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper 401/402/409/503 status classification.
 */
export function mapAppError(
	envelopeError: AppError | undefined | null,
	status: number,
	fallbackMessage: string
): AppError {
	const category = categoryForStatus(status);
	const serverCategory = envelopeError?.category;
	const resolvedCategory = isCategory(serverCategory) ? serverCategory : category;
	const retryable = envelopeError?.retryable ?? defaultRetryableFor(resolvedCategory, status);
	const message = looksSafe(envelopeError?.message) ? (envelopeError as { message: string }).message : fallbackMessage;
	const code = envelopeError?.code ?? defaultCodeForStatus(status);

	const appError: AppError = {
		category: resolvedCategory,
		code,
		message,
		retryable
	};

	if (envelopeError?.requestId) {
		appError.requestId = envelopeError.requestId;
	}
	return appError;
}

/**
 * GETs `/api/v1/entitlements` with `credentials: "include"` and the provided AbortSignal,
 * then decodes the generated {@link EntitlementEnvelope} into an {@link EntitlementData}.
 * Non-2xx envelopes throw {@link EntitlementClientError}.
 */
export async function fetchEntitlement(signal?: AbortSignal): Promise<EntitlementData> {
	const response = await fetch(ENTITLEMENT_ENDPOINT, {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal
	});
	return decodeEntitlementResponse(response);
}

/**
 * POSTs a {@link CheckoutRequest} to `/api/v1/billing/checkout` with `credentials: "include"`,
 * automatically generated idempotency key, and the provided AbortSignal.
 * Decodes the generated {@link CheckoutSessionEnvelope} into a {@link CheckoutSessionData}.
 * Non-2xx envelopes throw {@link EntitlementClientError}.
 */
export async function createCheckoutSession(
	request: CheckoutRequest,
	signal?: AbortSignal
): Promise<CheckoutSessionData> {
	const response = await fetch(CHECKOUT_ENDPOINT, {
		method: "POST",
		credentials: "include",
		headers: {
			Accept: "application/json",
			"Content-Type": "application/json",
			...createIdempotencyHeader()
		},
		body: JSON.stringify(request),
		signal
	});
	return decodeCheckoutResponse(response);
}

/**
 * Builds TanStack Query options for the entitlement query backed by the stable query key,
 * a 10-second timeout, and proper retry behavior for auth/entitlement errors.
 */
export function buildEntitlementQueryOptions(
	timeoutMs: number = ENTITLEMENT_TIMEOUT_MS
): CreateQueryOptions<EntitlementData, EntitlementClientError, EntitlementData, EntitlementQueryKey> {
	const queryKey: EntitlementQueryKey = [ENTITLEMENT_QUERY_NAMESPACE];

	return {
		queryKey,
		queryFn: (context) => runEntitlementQueryFn(context.signal, timeoutMs),
		staleTime: 5 * 60 * 1000,
		retry: (failureCount, error) => {
			if (error instanceof EntitlementClientError) {
				return error.appError.retryable && failureCount < 3;
			}
			return failureCount < 3;
		}
	};
}

/**
 * Creates a chained AbortSignal that aborts when the parent aborts or after `timeoutMs` milliseconds,
 * returning a cancel handle that clears the timer and removes the parent listener.
 */
function createTimeoutSignal(parent: AbortSignal, timeoutMs: number): { signal: AbortSignal; cancel: () => void } {
	const controller = new AbortController();
	const onParentAbort = () => {
		controller.abort(parent.reason ?? new DOMException("Aborted", "AbortError"));
	};
	if (parent.aborted) {
		controller.abort(parent.reason);
	} else {
		parent.addEventListener("abort", onParentAbort, { once: true });
	}
	const timer = setTimeout(() => {
		controller.abort(new DOMException("Entitlement timeout", "TimeoutError"));
	}, timeoutMs);
	return {
		signal: controller.signal,
		cancel: () => {
			clearTimeout(timer);
			parent.removeEventListener("abort", onParentAbort);
		}
	};
}

async function runEntitlementQueryFn(parentSignal: AbortSignal, timeoutMs: number): Promise<EntitlementData> {
	const handle = createTimeoutSignal(parentSignal, timeoutMs);
	try {
		return await fetchEntitlement(handle.signal);
	} catch (error) {
		throw mapAbortError(error, handle.signal);
	} finally {
		handle.cancel();
	}
}

/**
 * Converts an `AbortError` caused by the timeout into a retryable {@link EntitlementClientError}.
 */
function mapAbortError(error: unknown, signal: AbortSignal): unknown {
	if (error instanceof DOMException && error.name === "AbortError") {
		const reason = signal.reason;
		if (reason instanceof DOMException && reason.name === "TimeoutError") {
			throw new EntitlementClientError(
				{
					category: "timeout",
					code: "entitlement_timeout",
					message: "Request took too long. Please try again.",
					retryable: true
				},
				408
			);
		}
	}
	throw error;
}

async function decodeEntitlementResponse(response: Response): Promise<EntitlementData> {
	const status = response.status;
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new EntitlementClientError(
			mapAppError(undefined, status, fallbackMessageForCategory(categoryForStatus(status))),
			status
		);
	}

	if (!response.ok) {
		const envelope = readEnvelope(body);
		const fallback = fallbackMessageForCategory(categoryForStatus(status));
		const appError = mapAppError(envelope?.error ?? undefined, status, fallback);
		attachRequestId(appError, envelope?.requestId);
		throw new EntitlementClientError(appError, status);
	}

	const envelope = body as EntitlementEnvelope | null;
	if (!envelope || typeof envelope !== "object" || envelope.data === undefined || envelope.data === null) {
		const appError: AppError = {
			category: "server",
			code: "malformed_envelope",
			message: fallbackMessageForCategory("server"),
			retryable: true
		};
		attachRequestId(appError, envelope?.requestId);
		throw new EntitlementClientError(appError, status);
	}
	return envelope.data;
}

async function decodeCheckoutResponse(response: Response): Promise<CheckoutSessionData> {
	const status = response.status;
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new EntitlementClientError(
			mapAppError(undefined, status, fallbackMessageForCategory(categoryForStatus(status))),
			status
		);
	}

	if (!response.ok) {
		const envelope = readEnvelope(body);
		const fallback = fallbackMessageForCategory(categoryForStatus(status));
		const appError = mapAppError(envelope?.error ?? undefined, status, fallback);
		attachRequestId(appError, envelope?.requestId);
		throw new EntitlementClientError(appError, status);
	}

	const envelope = body as CheckoutSessionEnvelope | null;
	if (!envelope || typeof envelope !== "object" || envelope.data === undefined || envelope.data === null) {
		const appError: AppError = {
			category: "server",
			code: "malformed_envelope",
			message: fallbackMessageForCategory("server"),
			retryable: true
		};
		attachRequestId(appError, envelope?.requestId);
		throw new EntitlementClientError(appError, status);
	}
	return envelope.data;
}

function readEnvelope(body: unknown): Envelope | null {
	if (typeof body !== "object" || body === null) {
		return null;
	}
	const candidate = body as { status?: unknown; requestId?: unknown; error?: unknown };
	if (typeof candidate.status !== "string" && typeof candidate.requestId !== "string" && candidate.error === undefined) {
		return null;
	}
	return candidate as Envelope;
}

function attachRequestId(appError: AppError, requestId: string | undefined): void {
	if (!appError.requestId && typeof requestId === "string" && requestId.length > 0) {
		appError.requestId = requestId;
	}
}

function categoryForStatus(status: number): AppError["category"] {
	switch (status) {
		case 400:
		case 422:
			return "validation";
		case 401:
		case 403:
			return "auth";
		case 402:
			return "entitlement";
		case 409:
			return "server"; // Usually conflict on idempotency key or state
		case 429:
			return "server";
		case 503:
			return "dependency";
		default:
			return "unknown";
	}
}

function defaultRetryableFor(category: AppError["category"], status: number): boolean {
	// 401 should strictly not be retryable to avoid loops
	if (status === 401) return false;
	return category === "server" || category === "dependency" || category === "network" || category === "timeout";
}

function defaultCodeForStatus(status: number): string {
	switch (status) {
		case 400:
			return "invalid_request";
		case 401:
			return "unauthorized";
		case 402:
			return "payment_required";
		case 403:
			return "forbidden";
		case 409:
			return "conflict";
		case 422:
			return "validation_failed";
		case 429:
			return "rate_limited";
		case 503:
			return "dependency_unavailable";
		default:
			return "unknown_error";
	}
}

function fallbackMessageForCategory(category: AppError["category"]): string {
	switch (category) {
		case "validation":
			return "Request could not be processed.";
		case "auth":
			return "Please sign in to continue.";
		case "entitlement":
			return "Your plan does not include this feature. Please upgrade to continue.";
		case "network":
			return "Network is unavailable. Please check your connection and try again.";
		case "timeout":
			return "Request took too long. Please try again.";
		case "server":
			return "Too many requests right now. Please wait a moment and try again.";
		case "dependency":
			return "Service is temporarily unavailable. Please try again shortly.";
		default:
			return "Something went wrong. Please try again.";
	}
}

function isCategory(value: unknown): value is AppError["category"] {
	return (
		value === "validation" ||
		value === "auth" ||
		value === "entitlement" ||
		value === "network" ||
		value === "timeout" ||
		value === "server" ||
		value === "dependency" ||
		value === "unknown"
	);
}

function looksSafe(message: string | undefined): message is string {
	if (typeof message !== "string" || message.length === 0) {
		return false;
	}
	if (message.includes("http://") || message.includes("https://")) {
		return false;
	}
	if (/\.(ts|js|go|rs|py):\d+/.test(message)) {
		return false;
	}
	if (message.includes("\n")) {
		return false;
	}
	if (/\b(stack|panic|goroutine|traceback)\b/i.test(message)) {
		return false;
	}
	return true;
}
