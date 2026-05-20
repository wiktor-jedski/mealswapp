import { ApiClientError } from '../api/client';
import type { AppError, ErrorCategory } from '../api/types';

export interface RetryPolicy {
  maxAttempts: number;
  baseDelayMs: number;
  maxDelayMs: number;
  jitterMs: number;
  retryableCategories: ErrorCategory[];
}

export interface DegradedFeature {
  name: string;
  component: string;
  reason: string;
  active: boolean;
  since: string;
}

export interface ErrorBoundaryState {
  hasError: boolean;
  error?: AppError;
  degradedFeatures: DegradedFeature[];
}

export interface ErrorMessageRule {
  code: string;
  category: ErrorCategory;
  userMessage: string;
  logLevel: 'info' | 'warn' | 'error';
}

export const defaultRetryPolicy: RetryPolicy = {
  maxAttempts: 3,
  baseDelayMs: 500,
  maxDelayMs: 5000,
  jitterMs: 0,
  retryableCategories: ['network', 'timeout', 'dependency', 'server']
};

const rules: ErrorMessageRule[] = [
  { category: 'validation', code: 'validation_error', userMessage: 'Check the highlighted fields and try again.', logLevel: 'info' },
  { category: 'auth', code: 'unauthorized', userMessage: 'Sign in to continue.', logLevel: 'info' },
  { category: 'auth', code: 'forbidden', userMessage: 'You do not have access to this action.', logLevel: 'warn' },
  { category: 'entitlement', code: 'entitlement_required', userMessage: 'Upgrade your plan to use this feature.', logLevel: 'info' },
  { category: 'network', code: 'offline', userMessage: 'You are offline. Cached results are shown when available.', logLevel: 'warn' },
  { category: 'network', code: 'network_error', userMessage: 'Network connection failed. Check your connection and retry.', logLevel: 'warn' },
  { category: 'timeout', code: 'timeout', userMessage: 'The request took too long. Try again.', logLevel: 'warn' },
  { category: 'dependency', code: 'rate_limited', userMessage: 'Too many requests. Wait a moment before retrying.', logLevel: 'warn' },
  { category: 'dependency', code: 'dependency_unavailable', userMessage: 'Search is temporarily degraded. Try again shortly.', logLevel: 'warn' },
  { category: 'server', code: 'internal_error', userMessage: 'Something went wrong on our side. Try again shortly.', logLevel: 'error' },
  { category: 'unknown', code: 'unknown_error', userMessage: 'Something went wrong. Try again.', logLevel: 'error' }
];

export function classifyClientError(error: unknown): AppError {
  if (error instanceof ApiClientError) {
    return {
      category: error.category,
      code: error.code,
      message: error.message,
      retryable: error.retryable,
      requestId: error.requestId,
      fields: error.fields,
      cause: error.cause
    };
  }
  if (isAppError(error)) {
    return error;
  }
  if (error instanceof DOMException && error.name === 'AbortError') {
    return { category: 'timeout', code: 'timeout', message: 'Request timed out', retryable: true, cause: error };
  }
  return { category: 'unknown', code: 'unknown_error', message: 'Unknown client error', retryable: false, cause: error };
}

export function mapErrorMessage(error: AppError): string {
  return matchingRule(error).userMessage;
}

export function errorLogLevel(error: AppError): ErrorMessageRule['logLevel'] {
  return matchingRule(error).logLevel;
}

export function shouldRetry(error: AppError, policy: RetryPolicy = defaultRetryPolicy, attempt: number): boolean {
  return attempt < policy.maxAttempts && error.retryable && policy.retryableCategories.includes(error.category);
}

export function nextRetryDelay(policy: RetryPolicy = defaultRetryPolicy, attempt: number): number {
  const exponential = policy.baseDelayMs * 2 ** Math.max(0, attempt - 1);
  return Math.min(policy.maxDelayMs, exponential + policy.jitterMs);
}

export function createErrorBoundaryState(): ErrorBoundaryState {
  return { hasError: false, degradedFeatures: [] };
}

export function captureBoundaryError(state: ErrorBoundaryState, error: unknown): ErrorBoundaryState {
  return { ...state, hasError: true, error: classifyClientError(error) };
}

export function resetBoundary(state: ErrorBoundaryState): ErrorBoundaryState {
  return { ...state, hasError: false, error: undefined };
}

export function markFeatureDegraded(state: ErrorBoundaryState, feature: Omit<DegradedFeature, 'active' | 'since'>, now = new Date()): ErrorBoundaryState {
  const degradedFeature: DegradedFeature = { ...feature, active: true, since: now.toISOString() };
  return {
    ...state,
    degradedFeatures: [degradedFeature, ...state.degradedFeatures.filter((candidate) => candidate.name !== feature.name)]
  };
}

function matchingRule(error: AppError): ErrorMessageRule {
  return (
    rules.find((rule) => rule.category === error.category && rule.code === error.code) ??
    rules.find((rule) => rule.category === error.category) ??
    rules[rules.length - 1]
  );
}

function isAppError(error: unknown): error is AppError {
  return Boolean(error && typeof error === 'object' && 'category' in error && 'code' in error && 'message' in error);
}
