import { describe, expect, it } from 'bun:test';
import { ApiClient, ApiClientError } from './client';
import type { Envelope, SearchResponse } from './types';

describe('ApiClient', () => {
  it('posts typed search requests and unwraps success envelopes', async () => {
    const response: SearchResponse = {
      items: [],
      totalCount: 0,
      page: 1,
      pageSize: 10,
      similarityScores: [],
      warnings: []
    };
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      baseUrl: 'https://api.example.test/api/v1',
      fetch: recordingFetch(calls, okEnvelope(response))
    });

    const result = await client.search({ query: 'tofu', mode: 'single', page: 1 });

    expect(result).toEqual(response);
    expect(calls[0].url).toBe('https://api.example.test/api/v1/search');
    expect(calls[0].init.method).toBe('POST');
    expect(JSON.parse(calls[0].init.body as string)).toEqual({ query: 'tofu', mode: 'single', page: 1 });
  });

  it('builds autocomplete query strings', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope([{ itemId: '1', label: 'Tofu', exactMatch: true, levenshteinDistance: 0, length: 4, rank: 1 }]))
    });

    const result = await client.autocomplete('red lentils', 5);

    expect(result[0].label).toBe('Tofu');
    expect(calls[0].url).toBe('/api/v1/autocomplete?query=red+lentils&limit=5');
  });

  it('raises typed API errors from failure envelopes', async () => {
    const client = new ApiClient({
      fetch: recordingFetch([], {
        ok: false,
        status: 503,
        statusText: 'Service Unavailable',
        body: {
          success: false,
          error: {
            category: 'dependency',
            code: 'dependency_unavailable',
            message: 'Search repository unavailable',
            retryable: true,
            requestId: 'req-1'
          }
        }
      })
    });

    try {
      await client.search({ query: 'tofu', mode: 'single', page: 1 });
      throw new Error('expected API error');
    } catch (error) {
      expect(error).toBeInstanceOf(ApiClientError);
      const apiError = error as ApiClientError;
      expect(apiError.category).toBe('dependency');
      expect(apiError.code).toBe('dependency_unavailable');
      expect(apiError.retryable).toBe(true);
      expect(apiError.requestId).toBe('req-1');
    }
  });

  it('maps fetch failures to retryable network errors', async () => {
    const client = new ApiClient({
      fetch: (async () => {
        throw new Error('offline');
      }) as unknown as typeof fetch
    });

    try {
      await client.getProfile();
      throw new Error('expected network error');
    } catch (error) {
      expect(error).toBeInstanceOf(ApiClientError);
      const apiError = error as ApiClientError;
      expect(apiError.category).toBe('network');
      expect(apiError.retryable).toBe(true);
    }
  });

  it('syncs theme preference through the profile endpoint', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({ id: 'user-1', metadata: { themePreference: 'dark' } }))
    });

    const profile = await client.updateThemePreference('dark');

    expect(profile.metadata?.themePreference).toBe('dark');
    expect(calls[0].url).toBe('/api/v1/profile');
    expect(calls[0].init.method).toBe('PATCH');
    expect(JSON.parse(calls[0].init.body as string)).toEqual({ metadata: { themePreference: 'dark' } });
  });

  it('exposes account export and deletion entry points', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({ status: 'deleted' }))
    });

    await client.exportAccountData('json');
    await client.deleteAccount();

    expect(calls[0].url).toBe('/api/v1/profile/export?format=json');
    expect(calls[1].url).toBe('/api/v1/profile');
    expect(calls[1].init.method).toBe('DELETE');
  });

  it('exposes entitlement, subscription status, checkout, and portal entry points', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({
        userId: 'user-1',
        tier: 'free',
        status: 'active',
        searchLimitPer24h: 3,
        allowedModes: ['single']
      }))
    });

    const entitlement = await client.getEntitlement();
    expect(entitlement.tier).toBe('free');
    expect(calls[0].url).toBe('/api/v1/subscription/entitlement');

    const statusClient = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({
        entitlement,
        billingState: 'active',
        plans: [{ id: 'paid_monthly', tier: 'paid', interval: 'monthly', priceCents: 300 }]
      }))
    });
    const status = await statusClient.getSubscriptionStatus();
    expect(status.plans?.[0].priceCents).toBe(300);
    expect(calls[1].url).toBe('/api/v1/subscription/status');

    const checkoutClient = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({ id: 'cs_test', url: 'https://stripe.test/checkout' }))
    });
    await checkoutClient.createCheckoutSession('price_monthly', 'https://app.test/success', 'https://app.test/cancel');
    expect(calls[2].url).toBe('/api/v1/subscription/checkout');
    expect(calls[2].init.method).toBe('POST');
    expect(JSON.parse(calls[2].init.body as string)).toEqual({
      priceId: 'price_monthly',
      successUrl: 'https://app.test/success',
      cancelUrl: 'https://app.test/cancel'
    });

    const portalClient = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({ url: 'https://stripe.test/portal' }))
    });
    await portalClient.createCustomerPortalSession('https://app.test/account');
    expect(calls[3].url).toBe('/api/v1/subscription/portal');
    expect(JSON.parse(calls[3].init.body as string)).toEqual({ returnUrl: 'https://app.test/account' });
  });

  it('submits optimization jobs and polls job status', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({
        jobId: 'job-1',
        pollUrl: '/api/v1/optimization/jobs/job-1',
        status: 'queued'
      }))
    });

    const submitted = await client.submitOptimizationJob({
      originalMeals: [{ id: 'meal-1', name: 'Oats', quantity: 100 }],
      targetMacros: { protein: 90, carbs: 160, fat: 55 },
      excludedIds: ['meal-9'],
      tolerancePercent: 10
    });

    expect(submitted.jobId).toBe('job-1');
    expect(calls[0].url).toBe('/api/v1/optimization/jobs');
    expect(calls[0].init.method).toBe('POST');

    const statusClient = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({
        jobId: 'job-1',
        userId: 'user-1',
        request: {
          originalMeals: [{ id: 'meal-1', name: 'Oats', quantity: 100 }],
          targetMacros: { protein: 90, carbs: 160, fat: 55 },
          excludedIds: [],
          tolerancePercent: 10
        },
        status: 'completed',
        progress: 100,
        createdAt: '2026-05-20T12:00:00Z',
        result: [{ meals: [{ itemId: 'tofu', quantity: 200 }], macros: { protein: 92, carbs: 150, fat: 52 }, calories: 620, similarityScore: 0.7 }]
      }))
    });

    const job = await statusClient.getOptimizationJob('job-1');
    expect(job.status).toBe('completed');
    expect(job.result?.[0].meals[0].itemId).toBe('tofu');
    expect(calls[1].url).toBe('/api/v1/optimization/jobs/job-1');
  });

  it('exposes admin workflow endpoints', async () => {
    const calls: RequestRecord[] = [];
    const client = new ApiClient({
      fetch: recordingFetch(calls, okEnvelope({ candidates: [], page: 1, pageSize: 10 }))
    });

    await client.adminExternalSearch('tofu', 'all', 1, 10);
    await client.adminListItems('tof', 2, 10);
    await client.adminCreateItem({ name: 'Tofu', physicalState: 'solid', servingUnit: 'gram', servingSize: 100 });
    await client.adminTransitionItem('food-1', 'approve');
    await client.adminListTags('diet');
    await client.adminAssignTag('food-1', 'tag-1');
    await client.adminMergeTags('tag-old', 'tag-1');
    await client.adminListUsers('user', 1, 10);
    await client.adminDisableUser('user-1');
    await client.adminResetUserLockout('user-1');
    await client.adminUserAudit('user-1');

    expect(calls[0].url).toBe('/api/v1/admin/external-search?query=tofu&provider=all&page=1&pageSize=10');
    expect(calls[1].url).toBe('/api/v1/admin/items?query=tof&page=2&pageSize=10');
    expect(calls[2].url).toBe('/api/v1/admin/items');
    expect(calls[2].init.method).toBe('POST');
    expect(calls[3].url).toBe('/api/v1/admin/items/food-1/approve');
    expect(calls[4].url).toBe('/api/v1/admin/tags?kind=diet');
    expect(calls[5].url).toBe('/api/v1/admin/items/food-1/tags');
    expect(calls[6].url).toBe('/api/v1/admin/tags/merge');
    expect(calls[7].url).toBe('/api/v1/admin/users?query=user&page=1&pageSize=10');
    expect(calls[8].url).toBe('/api/v1/admin/users/user-1/disable');
    expect(calls[9].url).toBe('/api/v1/admin/users/user-1/reset-lockout');
    expect(calls[10].url).toBe('/api/v1/admin/users/user-1/audit?page=1&pageSize=10');
  });
});

interface RequestRecord {
  url: string;
  init: RequestInit;
}

interface MockResponse {
  ok: boolean;
  status: number;
  statusText: string;
  body: unknown;
}

function okEnvelope<T>(data: T): MockResponse {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    body: { success: true, data } satisfies Envelope<T>
  };
}

function recordingFetch(calls: RequestRecord[], response: MockResponse): typeof fetch {
  return (async (input: string | URL | Request, init?: RequestInit) => {
    calls.push({ url: String(input), init: init ?? {} });
    return {
      ok: response.ok,
      status: response.status,
      statusText: response.statusText,
      json: async () => response.body
    } as Response;
  }) as typeof fetch;
}
