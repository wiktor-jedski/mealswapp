import type {
  AppError,
  AdminAuditHistory,
  AdminFoodItem,
  AdminItemList,
  AdminTag,
  AdminUserDetail,
  AdminUserList,
  CheckoutSession,
  CustomerPortalSession,
  DietOptimizationRequest,
  Entitlement,
  Envelope,
  ExternalProvider,
  ExternalSearchResult,
  OptimizationJob,
  OptimizationSubmitResponse,
  RankedAutocomplete,
  SearchRequest,
  SearchResponse,
  SubscriptionStatus,
  UserProfile
} from './types';
import type { ThemePreference } from '../theme/theme';

export interface ApiClientOptions {
  baseUrl?: string;
  fetch?: typeof fetch;
}

export class ApiClientError extends Error implements AppError {
  category: AppError['category'];
  code: string;
  retryable: boolean;
  requestId?: string;
  fields?: unknown;
  cause?: unknown;

  constructor(error: AppError) {
    super(error.message);
    this.name = 'ApiClientError';
    this.category = error.category;
    this.code = error.code;
    this.retryable = error.retryable;
    this.requestId = error.requestId;
    this.fields = error.fields;
    this.cause = error.cause;
  }
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly fetcher: typeof fetch;

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? '/api/v1';
    this.fetcher = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  search(request: SearchRequest): Promise<SearchResponse> {
    return this.request<SearchResponse>('/search', {
      method: 'POST',
      body: JSON.stringify(request)
    });
  }

  autocomplete(query: string, limit = 10): Promise<RankedAutocomplete[]> {
    const params = new URLSearchParams({ query, limit: String(limit) });
    return this.request<RankedAutocomplete[]>(`/autocomplete?${params.toString()}`);
  }

  getProfile(): Promise<UserProfile> {
    return this.request<UserProfile>('/profile');
  }

  updateProfile(update: Partial<UserProfile>): Promise<UserProfile> {
    return this.request<UserProfile>('/profile', {
      method: 'PATCH',
      body: JSON.stringify(update)
    });
  }

  updateThemePreference(preference: ThemePreference): Promise<UserProfile> {
    return this.updateProfile({
      metadata: { themePreference: preference }
    });
  }

  exportAccountData(format: 'json' | 'csv'): Promise<unknown> {
    return this.request<unknown>(`/profile/export?format=${encodeURIComponent(format)}`);
  }

  deleteAccount(): Promise<{ status: string }> {
    return this.request<{ status: string }>('/profile', {
      method: 'DELETE'
    });
  }

  getEntitlement(): Promise<Entitlement> {
    return this.request<Entitlement>('/subscription/entitlement');
  }

  getSubscriptionStatus(): Promise<SubscriptionStatus> {
    return this.request<SubscriptionStatus>('/subscription/status');
  }

  createCheckoutSession(priceId: string, successUrl: string, cancelUrl: string): Promise<CheckoutSession> {
    return this.request<CheckoutSession>('/subscription/checkout', {
      method: 'POST',
      body: JSON.stringify({ priceId, successUrl, cancelUrl })
    });
  }

  createCustomerPortalSession(returnUrl: string): Promise<CustomerPortalSession> {
    return this.request<CustomerPortalSession>('/subscription/portal', {
      method: 'POST',
      body: JSON.stringify({ returnUrl })
    });
  }

  submitOptimizationJob(request: DietOptimizationRequest): Promise<OptimizationSubmitResponse> {
    return this.request<OptimizationSubmitResponse>('/optimization/jobs', {
      method: 'POST',
      body: JSON.stringify(request)
    });
  }

  getOptimizationJob(jobId: string): Promise<OptimizationJob> {
    return this.request<OptimizationJob>(`/optimization/jobs/${encodeURIComponent(jobId)}`);
  }

  adminExternalSearch(query: string, provider: ExternalProvider, page = 1, pageSize = 10): Promise<ExternalSearchResult> {
    const params = new URLSearchParams({ query, provider, page: String(page), pageSize: String(pageSize) });
    return this.request<ExternalSearchResult>(`/admin/external-search?${params.toString()}`);
  }

  adminListItems(query = '', page = 1, pageSize = 10): Promise<AdminItemList> {
    const params = new URLSearchParams({ query, page: String(page), pageSize: String(pageSize) });
    return this.request<AdminItemList>(`/admin/items?${params.toString()}`);
  }

  adminCreateItem(item: AdminFoodItem): Promise<AdminFoodItem> {
    return this.request<AdminFoodItem>('/admin/items', { method: 'POST', body: JSON.stringify(item) });
  }

  adminUpdateItem(id: string, item: AdminFoodItem): Promise<AdminFoodItem> {
    return this.request<AdminFoodItem>(`/admin/items/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(item) });
  }

  adminTransitionItem(id: string, transition: 'approve' | 'reject' | 'deactivate'): Promise<AdminFoodItem> {
    return this.request<AdminFoodItem>(`/admin/items/${encodeURIComponent(id)}/${transition}`, { method: 'POST' });
  }

  adminListTags(kind: string): Promise<AdminTag[]> {
    return this.request<AdminTag[]>(`/admin/tags?kind=${encodeURIComponent(kind)}`);
  }

  adminUpsertTag(tag: AdminTag): Promise<AdminTag> {
    return this.request<AdminTag>('/admin/tags', { method: 'POST', body: JSON.stringify(tag) });
  }

  adminAssignTag(foodItemId: string, tagId: string): Promise<void> {
    return this.request<void>(`/admin/items/${encodeURIComponent(foodItemId)}/tags`, { method: 'POST', body: JSON.stringify({ tagId }) });
  }

  adminMergeTags(sourceId: string, targetId: string): Promise<void> {
    return this.request<void>('/admin/tags/merge', { method: 'POST', body: JSON.stringify({ sourceId, targetId }) });
  }

  adminListUsers(query = '', page = 1, pageSize = 10): Promise<AdminUserList> {
    const params = new URLSearchParams({ query, page: String(page), pageSize: String(pageSize) });
    return this.request<AdminUserList>(`/admin/users?${params.toString()}`);
  }

  adminGetUser(id: string): Promise<AdminUserDetail> {
    return this.request<AdminUserDetail>(`/admin/users/${encodeURIComponent(id)}`);
  }

  adminDisableUser(id: string): Promise<AdminUserDetail['user']> {
    return this.request<AdminUserDetail['user']>(`/admin/users/${encodeURIComponent(id)}/disable`, { method: 'POST' });
  }

  adminResetUserLockout(id: string): Promise<void> {
    return this.request<void>(`/admin/users/${encodeURIComponent(id)}/reset-lockout`, { method: 'POST' });
  }

  adminUserAudit(id: string, page = 1, pageSize = 10): Promise<AdminAuditHistory> {
    const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
    return this.request<AdminAuditHistory>(`/admin/users/${encodeURIComponent(id)}/audit?${params.toString()}`);
  }

  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    let response: Response;
    try {
      response = await this.fetcher(this.url(path), {
        credentials: 'include',
        ...init,
        headers: {
          Accept: 'application/json',
          ...(init.body ? { 'Content-Type': 'application/json' } : {}),
          ...init.headers
        }
      });
    } catch (cause) {
      throw new ApiClientError({
        category: 'network',
        code: 'network_error',
        message: 'Network request failed',
        retryable: true,
        cause
      });
    }

    const envelope = await this.decodeEnvelope<T>(response);
    if (!response.ok || !envelope.success) {
      throw new ApiClientError(envelope.error ?? fallbackError(response));
    }
    return envelope.data as T;
  }

  private async decodeEnvelope<T>(response: Response): Promise<Envelope<T>> {
    try {
      return (await response.json()) as Envelope<T>;
    } catch (cause) {
      throw new ApiClientError({
        category: 'server',
        code: 'invalid_json',
        message: 'Invalid API response',
        retryable: false,
        cause
      });
    }
  }

  private url(path: string): string {
    if (/^https?:\/\//.test(path)) {
      return path;
    }
    return `${this.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
  }
}

export function createApiClient(options: ApiClientOptions = {}): ApiClient {
  return new ApiClient(options);
}

function fallbackError(response: Response): AppError {
  return {
    category: response.status >= 500 ? 'server' : 'unknown',
    code: 'request_failed',
    message: response.statusText || 'Request failed',
    retryable: response.status >= 500
  };
}
