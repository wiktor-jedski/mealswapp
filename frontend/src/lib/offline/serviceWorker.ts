export const serviceWorkerPath = '/sw.js';
export const serviceWorkerScope = '/';
export const serviceWorkerCacheLimits = {
  shell: 20,
  images: 60,
  api: 30
} as const;

export interface ServiceWorkerRegistrationTarget {
  serviceWorker?: {
    register: (scriptURL: string, options?: RegistrationOptions) => Promise<unknown>;
  };
}

export interface ServiceWorkerRegistrationState {
  supported: boolean;
  registered: boolean;
  scriptURL: string;
  error?: string;
}

export async function registerServiceWorker(target: ServiceWorkerRegistrationTarget = globalThis.navigator ?? {}): Promise<ServiceWorkerRegistrationState> {
  if (!target.serviceWorker) {
    return { supported: false, registered: false, scriptURL: serviceWorkerPath };
  }
  try {
    await target.serviceWorker.register(serviceWorkerPath, { scope: serviceWorkerScope });
    return { supported: true, registered: true, scriptURL: serviceWorkerPath };
  } catch (error) {
    return {
      supported: true,
      registered: false,
      scriptURL: serviceWorkerPath,
      error: error instanceof Error ? error.message : 'service worker registration failed'
    };
  }
}

export function shouldCacheShellAsset(pathname: string): boolean {
  return pathname === '/' || pathname === '/index.html' || pathname.startsWith('/assets/');
}

export function shouldCacheImage(pathname: string): boolean {
  return pathname.startsWith('/static/similarity/') || pathname.startsWith('/images/') || pathname.includes('/placeholder');
}

export function shouldCacheAPIGet(pathname: string, method: string): boolean {
  return method.toUpperCase() === 'GET' && (pathname.startsWith('/api/v1/search') || pathname.startsWith('/api/v1/autocomplete') || pathname.startsWith('/api/v1/admin/external-search'));
}

export function staleHeader(stale: boolean): Record<string, string> {
  return stale ? { 'X-MealSwapp-Stale': 'true' } : {};
}
