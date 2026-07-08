const CACHE_NAME = "gigpurse-v2";
const PRECACHE_URLS = ["/", "/icons/icon-192.png", "/icons/icon-512.png"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(PRECACHE_URLS)));
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))))
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin || url.pathname.startsWith("/api/")) return;

  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(() => caches.match(request).then((cached) => cached || caches.match("/")))
    );
    return;
  }

  if (url.pathname.startsWith("/_next/static/") || url.pathname.startsWith("/icons/")) {
    // Network-first, not cache-first: in dev mode Next's chunk URLs are NOT
    // content-hashed (stable paths, reused across rebuilds for Fast
    // Refresh), so a cache-first strategy here would permanently pin the
    // browser to whatever JS was first cached and silently ignore every
    // code change afterward. Network-first keeps the cache as an offline
    // fallback only, without that staleness trap.
    event.respondWith(
      fetch(request)
        .then((res) => {
          const clone = res.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(request, clone));
          return res;
        })
        .catch(() => caches.match(request))
    );
  }
});

// Real OS-level push, distinct from the in-app websocket notifications —
// this fires even when no GigPurse tab is open. Payload shape is set by
// the backend's pushSender: {title, body, link}.
self.addEventListener("push", (event) => {
  let data = { title: "GigPurse", body: "You have a new notification.", link: "/" };
  try {
    data = { ...data, ...event.data.json() };
  } catch {
    // Non-JSON or empty payload — fall back to the generic message above.
  }
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: "/icons/icon-192.png",
      badge: "/icons/icon-192.png",
      data: { link: data.link || "/" },
    })
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const link = event.notification.data?.link || "/";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ("focus" in client) {
          client.navigate(link);
          return client.focus();
        }
      }
      return self.clients.openWindow(link);
    })
  );
});
