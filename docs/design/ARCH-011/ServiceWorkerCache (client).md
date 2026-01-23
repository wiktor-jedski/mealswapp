# ServiceWorkerCache (client)

**Traceability:** ARCH-011

## 1. Data Structures & Types

### 1.1 Cache Entry Interface

```typescript
interface CacheEntry {
  url: string;
  timestamp: number;
  expiresAt: number;
  cacheControlMaxAge: number;
  response: Response;
  metadata: CacheMetadata;
}

interface CacheMetadata {
  contentType: string;
  contentLength: number;
  etag: string;
  isStale: boolean;
}
```

### 1.2 Cache Configuration

```typescript
interface CacheConfig {
  maxEntries: number;
  maxSizeMB: number;
  defaultTTL: number;
  staleWhileRevalidate: boolean;
  cacheImagesOnly: boolean;
}

const DEFAULT_CACHE_CONFIG: CacheConfig = {
  maxEntries: 500,
  maxSizeMB: 100,
  defaultTTL: 24 * 60 * 60 * 1000,
  staleWhileRevalidate: true,
  cacheImagesOnly: true
};
```

### 1.3 Image URL Pattern

```typescript
const IMAGE_URL_PATTERNS = [
  /\/images\/food\//,
  /\.(jpg|jpeg|png|webp|avif)(\?.*)?$/i
];
```

### 1.4 Service Worker Message Types

```typescript
type ServiceWorkerMessageType =
  | 'CACHE_URLS'
  | 'CLEAR_CACHE'
  | 'PURGE_URLS'
  | 'GET_CACHE_SIZE'
  | 'GET_CACHE_ENTRIES'
  | 'OFFLINE_STATUS';

interface ServiceWorkerMessage {
  type: ServiceWorkerMessageType;
  payload: Record<string, unknown>;
  timestamp: number;
}
```

### 1.5 Offline State

```typescript
interface OfflineState {
  isOnline: boolean;
  lastOnlineAt: number | null;
  cachedRequestsCount: number;
  stalenessIndicator: boolean;
}
```

## 2. Logic & Algorithms

### 2.1 Service Worker Registration

**Step 1:** Check if Service Workers are supported
```typescript
function isServiceWorkerSupported(): boolean {
  return 'serviceWorker' in navigator;
}
```

**Step 2:** Register the Service Worker file
```typescript
async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!isServiceWorkerSupported()) {
    console.warn('Service Workers not supported');
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js', {
      scope: '/'
    });
    
    console.log('Service Worker registered:', registration.scope);
    
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;
      if (newWorker) {
        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            dispatchEvent(new CustomEvent('sw-update-available'));
          }
        });
      }
    });
    
    return registration;
  } catch (error) {
    console.error('Service Worker registration failed:', error);
    return null;
  }
}
```

**Step 3:** Initialize ServiceWorkerCache on app load
```typescript
async function initializeServiceWorkerCache(): Promise<void> {
  const registration = await registerServiceWorker();
  
  if (registration) {
    await setupPushNotificationListener(registration);
    await syncCacheOnActivate(registration);
  }
  
  setupOnlineStatusListener();
}
```

### 2.2 Image Caching Strategy (Cache-First)

**Step 1:** Intercept fetch requests in Service Worker
```typescript
self.addEventListener('fetch', (event: FetchEvent) => {
  const url = new URL(event.request.url);
  
  if (shouldCacheRequest(url, event.request)) {
    event.respondWith(handleCachableRequest(event.request));
  }
});

function shouldCacheRequest(url: URL, request: Request): boolean {
  if (!isImageRequest(url)) {
    return false;
  }
  
  if (isSameOrigin(url) === false) {
    return false;
  }
  
  return true;
}

function isImageRequest(url: URL): boolean {
  const pathname = url.pathname;
  return IMAGE_URL_PATTERNS.some(pattern => pattern.test(pathname));
}
```

**Step 2:** Implement cache-first strategy
```typescript
async function handleCachableRequest(request: Request): Promise<Response> {
  const cache = await getImageCache();
  const cachedResponse = await cache.match(request);
  
  if (cachedResponse) {
    const metadata = await getEntryMetadata(request.url);
    
    if (metadata && !isExpired(metadata.expiresAt)) {
      updateAccessTime(request.url);
      return cachedResponse;
    }
    
    if (shouldRevalidateStale(request, metadata)) {
      return await revalidateAndCache(request, cache, cachedResponse);
    }
    
    return cachedResponse;
  }
  
  return await fetchAndCache(request, cache);
}

async function revalidateAndCache(
  request: Request,
  cache: Cache,
  staleResponse: Response
): Promise<Response> {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const clonedResponse = networkResponse.clone();
      await cache.put(request, clonedResponse);
      updateEntryMetadata(request.url, networkResponse);
    }
    
    return networkResponse;
  } catch {
    return staleResponse;
  }
}
```

**Step 3:** Respect Cache-Control headers
```typescript
function parseCacheControl(header: string | null): CacheControlParams {
  if (!header) {
    return { maxAge: DEFAULT_CACHE_CONFIG.defaultTTL };
  }
  
  const directives = header.split(',').reduce((acc, directive) => {
    const [key, value] = directive.trim().split('=');
    acc[key.trim()] = value ? parseInt(value, 10) : true;
    return acc;
  }, {} as Record<string, number | boolean>);
  
  return {
    maxAge: (directives['max-age'] as number) || DEFAULT_CACHE_CONFIG.defaultTTL,
    noCache: directives['no-cache'] as boolean,
    noStore: directives['no-store'] as boolean,
    mustRevalidate: directives['must-revalidate'] as boolean
  };
}

function isExpired(expiresAt: number): boolean {
  return Date.now() > expiresAt;
}
```

### 2.3 Cache Management

**Step 1:** Open or create image cache
```typescript
async function getImageCache(): Promise<Cache> {
  return await caches.open(IMAGE_CACHE_NAME);
}

const IMAGE_CACHE_NAME = 'mealswapp-images-v1';
```

**Step 2:** Pre-cache known image URLs
```typescript
async function preCacheImageUrls(urls: string[]): Promise<void> {
  const cache = await getImageCache();
  
  const cachePromises = urls.map(async (url) => {
    try {
      const response = await fetch(url, { mode: 'cors' });
      
      if (response.ok) {
        await cache.put(url, response);
        await setEntryMetadata(url, response);
      }
    } catch (error) {
      console.warn(`Failed to pre-cache image: ${url}`, error);
    }
  });
  
  await Promise.allSettled(cachePromises);
}
```

**Step 3:** Evict old entries using LRU policy
```typescript
async function evictOldEntries(): Promise<void> {
  const cache = await getImageCache();
  const keys = await cache.keys();
  
  const entries: Array<{ url: string; metadata: CacheMetadata | null; lastAccess: number }> = [];
  
  for (const request of keys) {
    const metadata = await getEntryMetadata(request.url);
    const lastAccess = await getLastAccessTime(request.url);
    
    entries.push({
      url: request.url,
      metadata,
      lastAccess: lastAccess || 0
    });
  }
  
  entries.sort((a, b) => a.lastAccess - b.lastAccess);
  
  const toEvict = entries.slice(0, Math.max(0, entries.length - DEFAULT_CACHE_CONFIG.maxEntries));
  
  for (const entry of toEvict) {
    await cache.delete(entry.url);
    await deleteEntryMetadata(entry.url);
  }
}
```

### 2.4 Cache Invalidation via Push Notifications

**Step 1:** Register for push notifications
```typescript
async function setupPushNotificationListener(
  registration: ServiceWorkerRegistration
): Promise<void> {
  registration.addEventListener('push', async (event: PushEvent) => {
    const data = event.data?.json();
    
    if (data?.type === 'CACHE_INVALIDATION') {
      event.waitUntil(invalidateCache(data.urls));
    }
  });
}

async function invalidateCache(urls: string[]): Promise<void> {
  const cache = await getImageCache();
  
  for (const url of urls) {
    await cache.delete(url);
    await deleteEntryMetadata(url);
  }
  
  dispatchEvent(new CustomEvent('cache-invalidated', { detail: urls }));
}
```

**Step 2:** Send purge command from server
```typescript
async function purgeStaleImages(imageUrls: string[]): Promise<void> {
  const registration = await navigator.serviceWorker.ready;
  
  if (registration.active) {
    registration.active.postMessage({
      type: 'PURGE_URLS',
      payload: { urls: imageUrls },
      timestamp: Date.now()
    });
  }
}
```

### 2.5 Offline Mode Handling

**Step 1:** Monitor online/offline status
```typescript
function setupOnlineStatusListener(): void {
  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);
}

function handleOnline(): void {
  const state = getOfflineState();
  state.isOnline = true;
  state.lastOnlineAt = Date.now();
  setOfflineState(state);
  
  dispatchEvent(new CustomEvent('app-online'));
}

function handleOffline(): void {
  const state = getOfflineState();
  state.isOnline = false;
  setOfflineState(state);
  
  dispatchEvent(new CustomEvent('app-offline'));
}
```

**Step 2:** Serve cached content when offline
```typescript
async function getCachedContent<T>(
  url: string,
  fallback: () => Promise<T>
): Promise<T> {
  if (navigator.onLine) {
    return await fallback();
  }
  
  const cache = await getImageCache();
  const response = await cache.match(url);
  
  if (response) {
    const metadata = await getEntryMetadata(url);
    
    if (metadata) {
      metadata.isStale = true;
      await setEntryMetadata(url, metadata);
    }
    
    return await response.json() as T;
  }
  
  throw new Error('Offline and no cached version available');
}
```

**Step 3:** Display staleness indicator
```typescript
function checkStaleness(url: string): boolean {
  const metadata = getEntryMetadataSync(url);
  
  if (!metadata) {
    return false;
  }
  
  const now = Date.now();
  const stalenessThreshold = metadata.expiresAt + (24 * 60 * 60 * 1000);
  
  return now > stalenessThreshold;
}
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Description | Trigger | Handling |
|-------------|-------------|---------|----------|
| `ServiceWorkerRegistrationFailed` | Service Worker failed to register | Browser security restrictions, network issues | Fallback to non-cached mode, log error |
| `CacheQuotaExceeded` | Cache storage quota exceeded | Storage limit reached | Trigger LRU eviction, log warning |
| `CacheDeleteFailed` | Failed to delete cached entry | Corrupted cache entry | Remove from metadata, continue |
| `NetworkUnavailable` | No network and no cached response | Device offline, no prior cache | Show offline banner, return error |
| `CacheCorrupted` | Cached response is corrupted | Storage corruption | Delete entry, fetch fresh copy |
| `StaleResponseServed` | Serving stale cached content | Cache expired, network unavailable | Show staleness indicator to user |
| `PushNotificationFailed` | Push notification registration failed | Browser不支持, permission denied | Fallback to periodic sync |

### 3.2 State Transitions

```
Initial State
    ↓
[Service Worker Registered] → [Cache Ready]
    ↓                    ↓
[Online] ─────────────→ [Offline]
    ↓                    ↓
[Fetch from Network]  [Serve from Cache]
    ↓                    ↓
[Cache Updated] ─────→ [Stale Served]
    ↓                    ↓
[Ready] ←────────────── [Back Online]
```

### 3.3 Error Recovery Strategies

**Service Worker Registration Failure:**
```typescript
async function handleRegistrationFailure(error: Error): Promise<void> {
  console.error('SW registration failed:', error);
  
  localStorage.setItem('sw-registration-failed', 'true');
  
  dispatchEvent(new CustomEvent('sw-registration-failed', {
    detail: { message: error.message }
  }));
}
```

**Cache Quota Exceeded:**
```typescript
async function handleQuotaExceeded(): Promise<void> {
  console.warn('Cache quota exceeded, running eviction');
  
  await evictOldEntries();
  
  const cache = await getImageCache();
  const keys = await cache.keys();
  
  if (keys.length >= DEFAULT_CACHE_CONFIG.maxEntries) {
    dispatchEvent(new CustomEvent('cache-warning', {
      detail: { message: 'Cache is full, some images may not be cached' }
    }));
  }
}
```

**Network Unavailable with No Cache:**
```typescript
async function handleCacheMiss(url: string): Promise<never> {
  const state = getOfflineState();
  state.cachedRequestsCount++;
  setOfflineState(state);
  
  dispatchEvent(new CustomEvent('cache-miss-offline', {
    detail: { url }
  }));
  
  throw new OfflineCacheMissError(`No cached version of ${url} available while offline`);
}
```

## 4. Component Interfaces

### 4.1 Public API

```typescript
class ServiceWorkerCache {
  private static instance: ServiceWorkerCache;
  private config: CacheConfig;
  private isRegistered: boolean = false;
  
  static getInstance(): ServiceWorkerCache {
    if (!ServiceWorkerCache.instance) {
      ServiceWorkerCache.instance = new ServiceWorkerCache();
    }
    return ServiceWorkerCache.instance;
  }
  
  async initialize(): Promise<void>;
  
  async cacheUrls(urls: string[]): Promise<void>;
  
  async getCachedUrl(url: string): Promise<Response | null>;
  
  async clearCache(): Promise<void>;
  
  async purgeUrls(urls: string[]): Promise<void>;
  
  getCacheSize(): Promise<number>;
  
  getCacheEntries(): Promise<CacheEntry[]>;
  
  isOnline(): boolean;
  
  isReady(): boolean;
}
```

### 4.2 Internal Functions

```typescript
async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null>;

function isServiceWorkerSupported(): boolean;

async function getImageCache(): Promise<Cache>;

async function handleCachableRequest(request: Request): Promise<Response>;

function shouldCacheRequest(url: URL, request: Request): boolean;

async function fetchAndCache(request: Request, cache: Cache): Promise<Response>;

async function revalidateAndCache(
  request: Request,
  cache: Cache,
  staleResponse: Response
): Promise<Response>;

function parseCacheControl(header: string | null): CacheControlParams;

async function evictOldEntries(): Promise<void>;

async function setupPushNotificationListener(
  registration: ServiceWorkerRegistration
): Promise<void>;

async function invalidateCache(urls: string[]): Promise<void>;

function setupOnlineStatusListener(): Promise<void>;

function getOfflineState(): OfflineState;

function setOfflineState(state: OfflineState): void;

async function getEntryMetadata(url: string): Promise<CacheMetadata | null>;

async function setEntryMetadata(url: string, response: Response): Promise<void>;

async function deleteEntryMetadata(url: string): Promise<void>;

async function updateEntryMetadata(url: string, response: Response): Promise<void>;
```

### 4.3 Event Handlers

```typescript
function handleFetch(event: FetchEvent): void;

function handleInstall(event: InstallEvent): void;

function handleActivate(event: ActivateEvent): void;

function handlePush(event: PushEvent): void;

function handleSync(event: SyncEvent): void;

function handleMessage(event: ExtendableMessageEvent): void;
```

### 4.4 Configuration Constants

```typescript
const IMAGE_CACHE_NAME = 'mealswapp-images-v1';
const METADATA_CACHE_NAME = 'mealswapp-metadata-v1';
const DEFAULT_CACHE_CONFIG: CacheConfig = {
  maxEntries: 500,
  maxSizeMB: 100,
  defaultTTL: 24 * 60 * 60 * 1000,
  staleWhileRevalidate: true,
  cacheImagesOnly: true
};
const STALE_THRESHOLD_MS = 24 * 60 * 60 * 1000;
```
