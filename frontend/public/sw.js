/*
 * SwasthyaSetu service worker.
 *
 * Offline support is a product requirement, not progressive enhancement
 * (architecture section 11): connectivity is least reliable exactly where and
 * when first-aid guidance matters most.
 *
 * Three caching strategies, chosen per resource by what a stale copy costs:
 *
 *   App shell        cache-first    The UI itself never changes mid-session.
 *   First-aid content stale-while-  Showing yesterday's article instantly
 *                    revalidate     beats showing a spinner; it refreshes in
 *                                   the background.
 *   Everything else  network-first  Doctor availability and triage results
 *                                   must never be served stale -- an offline
 *                                   answer here would be actively unsafe.
 *
 * Bump CACHE_VERSION on any change to this file or the shell; `activate`
 * deletes every cache that does not match.
 */

const CACHE_VERSION = "v1";
const SHELL_CACHE = `swasthyasetu-shell-${CACHE_VERSION}`;
const CONTENT_CACHE = `swasthyasetu-content-${CACHE_VERSION}`;
const CURRENT_CACHES = [SHELL_CACHE, CONTENT_CACHE];

const SHELL_ASSETS = ["/", "/offline", "/manifest.webmanifest", "/icon.svg"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(SHELL_CACHE)
      // Individually, so one 404 cannot fail the whole installation and
      // leave the app with no worker at all.
      .then((cache) =>
        Promise.allSettled(SHELL_ASSETS.map((asset) => cache.add(asset))),
      )
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((key) => !CURRENT_CACHES.includes(key))
            .map((key) => caches.delete(key)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

/** Cached copy served immediately; a fresh copy replaces it in the background. */
async function staleWhileRevalidate(request) {
  const cache = await caches.open(CONTENT_CACHE);
  const cached = await cache.match(request);

  const network = fetch(request)
    .then((response) => {
      if (response.ok) cache.put(request, response.clone());
      return response;
    })
    .catch(() => undefined);

  return cached ?? (await network) ?? Response.error();
}

async function networkFirst(request) {
  try {
    return await fetch(request);
  } catch (error) {
    const cached = await caches.match(request);
    if (cached) return cached;
    throw error;
  }
}

async function navigationHandler(request) {
  try {
    return await fetch(request);
  } catch {
    // Prefer the real page if it was visited before; fall back to the
    // dedicated offline page, then to the cached shell.
    return (
      (await caches.match(request)) ??
      (await caches.match("/offline")) ??
      (await caches.match("/")) ??
      Response.error()
    );
  }
}

self.addEventListener("fetch", (event) => {
  const { request } = event;

  // Only GET is cacheable, and cross-origin responses are not ours to store.
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (request.mode === "navigate") {
    event.respondWith(navigationHandler(request));
    return;
  }

  // Curated first-aid content: the one thing that must survive going offline.
  if (url.pathname.startsWith("/api/v1/articles")) {
    event.respondWith(staleWhileRevalidate(request));
    return;
  }

  // Build output is content-hashed, so a cache hit is always correct.
  if (url.pathname.startsWith("/_next/static/")) {
    event.respondWith(
      caches
        .open(SHELL_CACHE)
        .then(async (cache) => {
          const cached = await cache.match(request);
          if (cached) return cached;
          const response = await fetch(request);
          if (response.ok) cache.put(request, response.clone());
          return response;
        }),
    );
    return;
  }

  event.respondWith(networkFirst(request));
});

// Week 18: drain the queue of symptom reports logged while offline. The
// registration name is fixed here so the client can register the sync with a
// matching tag.
self.addEventListener("sync", (event) => {
  if (event.tag === "sync-symptom-reports") {
    // Implemented in Week 18 alongside the client-side queue.
  }
});
