// Service Worker - 简单缓存策略
const CACHE_VERSION = 'v1';
const CACHE_NAME = 'valuation-' + CACHE_VERSION;

// 静态资源缓存
const STATIC_URLS = [
  '/',
  '/static/manifest.json',
  '/static/icon.svg',
];

self.addEventListener('install', e => {
  e.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(STATIC_URLS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE_NAME).map(k => caches.delete(k)))
    ).then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);

  // 实时数据接口: 网络优先, 失败回退缓存
  if (url.pathname === '/lookup' || url.pathname === '/search' || url.pathname === '/industries') {
    e.respondWith(
      fetch(e.request).then(r => {
        const copy = r.clone();
        caches.open(CACHE_NAME).then(c => c.put(e.request, copy));
        return r;
      }).catch(() => caches.match(e.request))
    );
    return;
  }

  // 其他: 缓存优先, 后台更新
  e.respondWith(
    caches.match(e.request).then(cached => {
      const fetched = fetch(e.request).then(r => {
        const copy = r.clone();
        caches.open(CACHE_NAME).then(c => c.put(e.request, copy));
        return r;
      }).catch(() => cached);
      return cached || fetched;
    })
  );
});
