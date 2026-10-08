// 校园版 PWA Service Worker（本项目新增）：静态资源缓存 + 离线兜底。
const CACHE = "campus-memos-v1";
const PRECACHE = ["/"];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.addAll(PRECACHE)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  // API 请求不缓存，保证数据实时性。
  if (url.pathname.startsWith("/api/") || url.pathname.includes(".")) {
    if (url.pathname.startsWith("/api/")) return;
  }
  if (event.request.method !== "GET") return;
  event.respondWith(
    caches.match(event.request).then(
      (cached) =>
        cached ||
        fetch(event.request)
          .then((resp) => {
            if (resp.ok && (url.pathname === "/" || url.pathname.startsWith("/assets/"))) {
              const clone = resp.clone();
              caches.open(CACHE).then((c) => c.put(event.request, clone));
            }
            return resp;
          })
          .catch(() => caches.match("/")),
    ),
  );
});
