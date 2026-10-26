/* منظومة تراخيص الآبار بالخارجة — service worker (offline app shell) */
const VERSION='3d891a79b1';const SHELL='wells-shell-'+VERSION;const RT='wells-rt-v1';
const CORE=['./','./index.html','./manifest.webmanifest','./icon-192.png','./icon-512.png','./icon-maskable-192.png','./icon-maskable-512.png','./apple-touch-icon.png','./favicon.png'];
self.addEventListener('install',e=>{e.waitUntil(caches.open(SHELL).then(c=>c.addAll(CORE)).then(()=>self.skipWaiting()))});
self.addEventListener('activate',e=>{e.waitUntil(caches.keys().then(ks=>Promise.all(ks.filter(k=>k.startsWith('wells-shell-')&&k!==SHELL).map(k=>caches.delete(k)))).then(()=>self.clients.claim()))});
const CDN=/^https:\/\/(fonts\.googleapis\.com|fonts\.gstatic\.com|cdnjs\.cloudflare\.com)\//;
function timeout(ms,p){return new Promise((res,rej)=>{const t=setTimeout(()=>rej(new Error('timeout')),ms);p.then(v=>{clearTimeout(t);res(v)},e=>{clearTimeout(t);rej(e)})})}
self.addEventListener('fetch',e=>{const req=e.request;if(req.method!=='GET')return;const url=new URL(req.url);
  if(req.mode==='navigate'&&url.origin===location.origin){
    e.respondWith((async()=>{const c=await caches.open(SHELL);try{const r=await timeout(4000,fetch(req));if(r&&r.ok){c.put('./index.html',r.clone())}return r}catch(_){return (await c.match('./index.html'))||(await c.match('./'))||Response.error()}})());return}
  if(url.origin===location.origin){e.respondWith(caches.match(req,{ignoreSearch:true}).then(m=>m||fetch(req).then(r=>{if(r&&r.ok){const cp=r.clone();caches.open(SHELL).then(c=>c.put(req,cp))}return r})));return}
  if(CDN.test(req.url)){e.respondWith(caches.open(RT).then(async c=>{const m=await c.match(req);const net=fetch(req).then(r=>{if(r&&(r.ok||r.type==='opaque'))c.put(req,r.clone());return r}).catch(()=>m);return m||net}));return}
  /* Supabase, maps, WhatsApp and everything else: always network */
});
self.addEventListener('message',e=>{if(e.data==='skip')self.skipWaiting()});
