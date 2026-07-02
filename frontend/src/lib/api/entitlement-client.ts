import type { QueryFunctionContext } from "@tanstack/query-core";
import type { CreateMutationOptions, CreateQueryOptions } from "@tanstack/svelte-query";

import {
	BILLING_CHECKOUT_ENDPOINT,
	BILLING_ENTITLEMENT_ENDPOINT,
	buildCheckoutCreateRequestInit,
	buildEntitlementStatusRequestInit,
	type AppError,
	type CheckoutCreateRequest,
	type CheckoutSessionData,
	type CheckoutSessionEnvelope,
	type EntitlementStatusData,
	type EntitlementStatusEnvelope,
	type Envelope,
	type IdempotencyKey
} from "./generated";

// Implements DESIGN-001 SearchView current user entitlement state over generated billing contracts.
// Implements DESIGN-017 ErrorMessageMapper recoverable billing and entitlement error mapping.

/** Stable query-key namespace for current-user billing entitlement status. */
const ENTITLEMENT_QUERY_NAMESPACE = "billing-entitlement" as const;

/** Stable mutation-key namespace for hosted checkout creation. */
const CHECKOUT_MUTATION_NAMESPACE = "billing-checkout" as const;

/** Current-user entitlement query key; it has no user id because credentials identify the session. */
export type EntitlementQueryKey = readonly [typeof ENTITLEMENT_QUERY_NAMESPACE];

/** Checkout creation mutation key used by TanStack Query mutation state. */
export type CheckoutMutationKey = readonly [typeof CHECKOUT_MUTATION_NAMESPACE];

/**
 * Variables accepted by the checkout mutation. `idempotencyKey` is optional so UI code can
 * normally let the client generate one, while tests and replay flows can supply a known key.
 *
 * @remarks Implements DESIGN-001 SearchView checkout creation mutation variables.
 */
export interface CheckoutMutationVariables {
	request: CheckoutCreateRequest;
	csrfToken?: string;
	idempotencyKey?: IdempotencyKey;
	signal?: AbortSignal;
}

/**
 * Error thrown by the entitlement client for non-2xx billing envelopes. Carries a generated
 * {@link AppError} plus a `recoverable` flag for UI gates that can offer checkout or retry.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper recoverable billing client error contract.
 */
export class EntitlementClientError extends Error {
	readonly appError: AppError;
	readonly status: number;
	readonly recoverable: boolean;

	constructor(appError: AppError, status: number, recoverable: boolean) {
		super(appError.message);
		this.name = "EntitlementClientError";
		this.appError = appError;
		this.status = status;
		this.recoverable = recoverable;
	}
}

/**
 * GETs the generated entitlement endpoint with cookies included and decodes the generated
 * {@link EntitlementStatusEnvelope} to its data payload.
 *
 * @remarks Implements DESIGN-001 SearchView current user entitlement fetch.
 */
export async function fetchEntitlementStatus(signal?: AbortSignal): Promise<EntitlementStatusData> {
	const response = await fetch(BILLING_ENTITLEMENT_ENDPOINT, buildEntitlementStatusRequestInit({ signal }));
	return decodeEntitlementStatusResponse(response);
}

/**
 * POSTs checkout creation with an idempotency key and decodes the generated checkout envelope.
 *
 * @remarks Implements DESIGN-001 SearchView hosted checkout creation client.
 */
export async function createCheckoutSession(
	request: CheckoutCreateRequest,
	options: { csrfToken?: string; idempotencyKey?: IdempotencyKey; signal?: AbortSignal } = {}
): Promise<CheckoutSessionData> {
	const idempotencyKey = options.idempotencyKey ?? generateCheckoutIdempotencyKey();
	const response = await fetch(
		BILLING_CHECKOUT_ENDPOINT,
		buildCheckoutCreateRequestInit(request, idempotencyKey, {
			csrfToken: options.csrfToken,
			signal: options.signal
		})
	);
	return decodeCheckoutSessionResponse(response);
}

/**
 * Builds the current-user entitlement TanStack Query options with a stable key and no anonymous
 * retry loop; 401 is surfaced immediately so anonymous users can continue without churn.
 *
 * @remarks Implements DESIGN-001 SearchView TanStack Query entitlement state.
 */
export function buildEntitlementQueryOptions(): CreateQueryOptions<
	EntitlementStatusData,
	EntitlementClientError,
	EntitlementStatusData,
	EntitlementQueryKey
> {
	return {
		queryKey: [ENTITLEMENT_QUERY_NAMESPACE],
		staleTime: 60_000,
		gcTime: 5 * 60_000,
		retry: false,
		queryFn: (context) => fetchEntitlementQueryFn(context)
	};
}

/**
 * Builds checkout mutation options. The generated idempotency key is pinned onto the mutation
 * variables so TanStack retries replay the same request key instead of creating duplicate sessions.
 *
 * @remarks Implements DESIGN-001 SearchView checkout creation mutation state and retry behavior.
 */
export function buildCheckoutMutationOptions(): CreateMutationOptions<
	CheckoutSessionData,
	EntitlementClientError,
	CheckoutMutationVariables
> {
	return {
		mutationKey: [CHECKOUT_MUTATION_NAMESPACE],
		retry: (failureCount, error) => error.status === 503 && error.recoverable && failureCount < 1,
		mutationFn: (variables) => {
			variables.idempotencyKey = variables.idempotencyKey ?? generateCheckoutIdempotencyKey();
			return createCheckoutSession(variables.request, {
				csrfToken: variables.csrfToken,
				idempotencyKey: variables.idempotencyKey,
				signal: variables.signal
			});
		}
	};
}

/**
 * Generates a client-side checkout idempotency key with a stable prefix for observability.
 *
 * @remarks Implements DESIGN-001 SearchView checkout idempotency-key generation.
 */
export function generateCheckoutIdempotencyKey(): IdempotencyKey {
	const cryptoValue = globalThis.crypto;
	if (cryptoValue && typeof cryptoValue.randomUUID === "function") {
		return `checkout-${cryptoValue.randomUUID()}`;
	}
	return `checkout-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

async function fetchEntitlementQueryFn(
	context: QueryFunctionContext<EntitlementQueryKey>
): Promise<EntitlementStatusData> {
	return fetchEntitlementStatus(context.signal);
}

async function decodeEntitlementStatusResponse(response: Response): Promise<EntitlementStatusData> {
	const envelope = await readJsonEnvelope(response);
	if (!response.ok) {
		throw mapBillingError(envelope, response.status);
	}
	if (!hasData(envelope)) {
		throw malformedEnvelopeError(response.status, envelope?.requestId);
	}
	return envelope.data as EntitlementStatusData;
}

async function decodeCheckoutSessionResponse(response: Response): Promise<CheckoutSessionData> {
	const envelope = await readJsonEnvelope(response);
	if (!response.ok) {
		throw mapBillingError(envelope, response.status);
	}
	if (!hasData(envelope)) {
		throw malformedEnvelopeError(response.status, envelope?.requestId);
	}
	return envelope.data as CheckoutSessionData;
}

async function readJsonEnvelope(response: Response): Promise<Envelope | null> {
	try {
		const body = (await response.json()) as unknown;
		return readEnvelope(body);
	} catch {
		return null;
	}
}

function readEnvelope(body: unknown): Envelope | null {
	if (typeof body !== "object" || body === null) {
		return null;
	}
	return body as Envelope;
}

function hasData(envelope: Envelope | null): envelope is EntitlementStatusEnvelope | CheckoutSessionEnvelope {
	return Boolean(envelope && typeof envelope === "object" && envelope.data);
}

function mapBillingError(envelope: Envelope | null, status: number): EntitlementClientError {
	const appError = mapBillingAppError(envelope?.error ?? null, status);
	if (!appError.requestId && envelope?.requestId) {
		appError.requestId = envelope.requestId;
	}
	return new EntitlementClientError(appError, status, isRecoverableBillingError(appError, status));
}

function malformedEnvelopeError(status: number, requestId: string | undefined): EntitlementClientError {
	const appError: AppError = {
		category: "server",
		code: "malformed_envelope",
		message: "Billing status is temporarily unavailable. Please try again shortly.",
		retryable: true
	};
	if (requestId) {
		appError.requestId = requestId;
	}
	return new EntitlementClientError(appError, status, true);
}

function mapBillingAppError(envelopeError: AppError | null, status: number): AppError {
	const fallback = fallbackForBillingStatus(status);
	const appError: AppError = {
		category: envelopeError?.category ?? categoryForBillingStatus(status),
		code: envelopeError?.code ?? codeForBillingStatus(status),
		message: safeMessage(envelopeError?.message) ? envelopeError.message : fallback,
		retryable: envelopeError?.retryable ?? retryableForBillingStatus(status)
	};
	if (envelopeError?.requestId) {
		appError.requestId = envelopeError.requestId;
	}
	return appError;
}

function categoryForBillingStatus(status: number): AppError["category"] {
	switch (status) {
		case 401:
			return "auth";
		case 402:
		case 409:
			return "entitlement";
		case 400:
		case 422:
			return "validation";
		case 503:
			return "dependency";
		default:
			return "unknown";
	}
}

function codeForBillingStatus(status: number): string {
	switch (status) {
		case 401:
			return "anonymous_session";
		case 402:
			return "billing_payment_required";
		case 409:
			return "checkout_idempotency_conflict";
		case 400:
			return "checkout_invalid_request";
		case 422:
			return "checkout_validation_failed";
		case 503:
			return "entitlement_unavailable";
		default:
			return "unknown_error";
	}
}

function retryableForBillingStatus(status: number): boolean {
	return status === 503;
}

function isRecoverableBillingError(appError: AppError, status: number): boolean {
	return status === 402 || status === 409 || status === 503 || appError.retryable;
}

function fallbackForBillingStatus(status: number): string {
	switch (status) {
		case 401:
			return "Sign in to view your billing status.";
		case 402:
			return "Update billing to continue using this search mode.";
		case 409:
			return "Checkout could not be replayed. Please try again.";
		case 503:
			return "Billing status is temporarily unavailable. Please try again shortly.";
		default:
			return "Billing request could not be completed. Please try again.";
	}
}

function safeMessage(message: string | undefined): message is string {
	if (!message) {
		return false;
	}
	return !message.includes("\n") && !message.includes("http://") && !message.includes("https://");
}
