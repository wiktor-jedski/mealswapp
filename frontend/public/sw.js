const VERSION = 'mealswapp-sw-v1';
const SHELL_CACHE = `${VERSION}:shell`;
const IMAGE_CACHE = `${VERSION}:images`;
const API_CACHE = `${VERSION}:api`;
const SHELL_ASSETS = ['/', '/index.html'];
const CACHE_LIMITS = {
  [SHELL_CACHE]: 20,
  [IMAGE_CACHE]: 60,
  [API_CACHE]: 30
};

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(SHELL_CACHE)
      .then((cache) => cache.addAll(SHELL_ASSETS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => !key.startsWith(VERSION)).map((key) => caches.delete(key))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  if (event.data && event.data.type === 'PURGE_USER_CACHE') {
    event.waitUntil(purgeUserCaches());
  }
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') {
    return;
  }
  const url = new URL(request.url);
  if (shouldHandleShell(url.pathname, request.destination)) {
    event.respondWith(networkFirst(request, SHELL_CACHE));
    return;
  }
  if (shouldHandleImage(url.pathname, request.destination)) {
    event.respondWith(cacheFirst(request, IMAGE_CACHE));
    return;
  }
  if (shouldHandleAPI(url.pathname)) {
    event.respondWith(networkFirst(request, API_CACHE));
  }
});

function shouldHandleShell(pathname, destination) {
  return pathname === '/' || pathname === '/index.html' || pathname.startsWith('/assets/') || destination === 'script' || destination === 'style';
}

function shouldHandleImage(pathname, destination) {
  return destination === 'image' || pathname.startsWith('/static/similarity/') || pathname.startsWith('/images/') || pathname.includes('/placeholder');
}

function shouldHandleAPI(pathname) {
  return pathname.startsWith('/api/v1/search') || pathname.startsWith('/api/v1/autocomplete') || pathname.startsWith('/api/v1/admin/external-search');
}

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(request);
  if (cached) {
    return markStale(cached);
  }
  try {
    const response = await fetch(request);
    if (response.ok) {
      await cache.put(request, response.clone());
      await trimCache(cache, CACHE_LIMITS[cacheName]);
    }
    return response;
  } catch {
    return cached || Response.error();
  }
}

async function networkFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  try {
    const response = await fetch(request);
    if (response.ok) {
      await cache.put(request, response.clone());
      await trimCache(cache, CACHE_LIMITS[cacheName]);
    }
    return response;
  } catch {
    const cached = await cache.match(request);
    if (cached) {
      return markStale(cached);
    }
    if (request.mode === 'navigate') {
      const shell = await cache.match('/');
      if (shell) {
        return markStale(shell);
      }
    }
    return Response.error();
  }
}

async function purgeUserCaches() {
  const keys = await caches.keys();
  await Promise.all(keys.filter((key) => key.startsWith(VERSION)).map((key) => caches.delete(key)));
}

async function trimCache(cache, maxEntries) {
  if (!maxEntries) {
    return;
  }
  const keys = await cache.keys();
  const overflow = keys.length - maxEntries;
  if (overflow <= 0) {
    return;
  }
  await Promise.all(keys.slice(0, overflow).map((request) => cache.delete(request)));
}

function markStale(response) {
  const headers = new Headers(response.headers);
  headers.set('X-MealSwapp-Stale', 'true');
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers
  });
}
