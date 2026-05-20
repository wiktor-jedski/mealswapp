import { describe, expect, it } from 'bun:test';
import { ApiClientError } from '../api/client';
import {
  captureBoundaryError,
  classifyClientError,
  createErrorBoundaryState,
  errorLogLevel,
  mapErrorMessage,
  markFeatureDegraded,
  nextRetryDelay,
  resetBoundary,
  shouldRetry
} from './errorHandling';

describe('ErrorMessageMapper', () => {
  it('maps validation, auth, entitlement, rate-limit, offline, dependency, and unknown messages', () => {
    expect(mapErrorMessage({ category: 'validation', code: 'validation_error', message: '', retryable: false })).toBe('Check the highlighted fields and try again.');
    expect(mapErrorMessage({ category: 'auth', code: 'unauthorized', message: '', retryable: false })).toBe('Sign in to continue.');
    expect(mapErrorMessage({ category: 'entitlement', code: 'entitlement_required', message: '', retryable: false })).toBe('Upgrade your plan to use this feature.');
    expect(mapErrorMessage({ category: 'dependency', code: 'rate_limited', message: '', retryable: true })).toBe('Too many requests. Wait a moment before retrying.');
    expect(mapErrorMessage({ category: 'network', code: 'offline', message: '', retryable: true })).toBe('You are offline. Cached results are shown when available.');
    expect(mapErrorMessage({ category: 'dependency', code: 'dependency_unavailable', message: '', retryable: true })).toBe('Search is temporarily degraded. Try again shortly.');
    expect(mapErrorMessage({ category: 'unknown', code: 'unknown_error', message: '', retryable: false })).toBe('Something went wrong. Try again.');
  });

  it('classifies API, timeout, and unknown client errors', () => {
    expect(classifyClientError(new ApiClientError({ category: 'auth', code: 'unauthorized', message: 'Unauthorized', retryable: false })).category).toBe('auth');
    expect(classifyClientError(new DOMException('aborted', 'AbortError')).category).toBe('timeout');
    expect(classifyClientError(new Error('boom')).category).toBe('unknown');
  });

  it('exposes retry and log-level decisions', () => {
    const retryable = { category: 'dependency' as const, code: 'dependency_unavailable', message: '', retryable: true };
    const validation = { category: 'validation' as const, code: 'validation_error', message: '', retryable: false };

    expect(shouldRetry(retryable, undefined, 0)).toBe(true);
    expect(shouldRetry(validation, undefined, 0)).toBe(false);
    expect(shouldRetry(retryable, undefined, 3)).toBe(false);
    expect(nextRetryDelay(undefined, 3)).toBe(2000);
    expect(errorLogLevel(retryable)).toBe('warn');
  });
});

describe('ErrorBoundary state', () => {
  it('captures and recovers from boundary errors', () => {
    const captured = captureBoundaryError(createErrorBoundaryState(), new Error('render failed'));
    expect(captured.hasError).toBe(true);
    expect(captured.error?.category).toBe('unknown');

    const recovered = resetBoundary(captured);
    expect(recovered.hasError).toBe(false);
    expect(recovered.error).toBeUndefined();
  });

  it('marks degraded feature state without dropping existing state', () => {
    const degraded = markFeatureDegraded(createErrorBoundaryState(), { name: 'history sync', component: 'SidebarComponent', reason: 'offline' }, new Date('2026-05-20T00:00:00.000Z'));

    expect(degraded.degradedFeatures).toEqual([
      { name: 'history sync', component: 'SidebarComponent', reason: 'offline', active: true, since: '2026-05-20T00:00:00.000Z' }
    ]);
  });
});
