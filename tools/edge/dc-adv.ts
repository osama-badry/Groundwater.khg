// =====================================================================
// منظومة تراخيص الآبار بالخارجة — الفحص العميق لتاريخ البئر (dc-adv)
// سلاسل شهرية منذ 2016 لثلاث مناطق: داخل حدود الأرض · حلقة الصحراء المحيطة (مرجع) · نقطة البئر (30 م)
// Sentinel-2: نسبة الزراعة (NDVI) · الرطوبة (NDMI) · التربة العارية (BSI) · المياه المكشوفة (MNDWI)
// Sentinel-1: الرادار VV (رطوبة التربة والري حتى بدون زراعة)
// 8 طلبات فقط لكل فحص (حماية حصة Copernicus) · يقف فوراً عند 429/503
// جميع الحقوق محفوظة للمصمم والمطور - م/ أسامه حسن بدري - 01145060007
// =====================================================================
import { createClient } from "npm:@supabase/supabase-js@2.45.4";
const CORS = { "Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type, x-dev-key", "Access-Control-Allow-Methods": "POST, OPTIONS" };
const TOKEN_URL = "https://identity.dataspace.copernicus.eu/auth/realms/CDSE/protocol/openid-connect/token";
const STATS_URL = "https://sh.dataspace.copernicus.eu/api/v1/statistics";
const CRS = "http://www.opengis.net/def/crs/OGC/1.3/CRS84";
const json = (o: unknown, st = 200) => new Response(JSON.stringify(o), { status: st, headers: { ...CORS, "Content-Type": "application/json" } });
let tok: { v: string; exp: number } | null = null;
async function cdseToken() {
  if (tok && Date.now() < tok.exp - 60000) return tok.v;
  const id = Deno.env.get("CDSE_CLIENT_ID"), sec = Deno.env.get("CDSE_CLIENT_SECRET");
  if (!id || !sec) throw new Error("no_cdse_secret");
  const r = await fetch(TOKEN_URL, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ grant_type: "client_credentials", client_id: id, client_secret: sec }) });
  if (!r.ok) throw new Error("cdse_auth_" + r.status);
  const j = await r.json(); tok = { v: j.access_token, exp: Date.now() + (j.expires_in || 600) * 1000 }; return tok.v;
}
async function stats(t: string, body: unknown) {
  let last = "";
  for (let k = 0; k < 3; k++) {
    if (k) await new Promise((r) => setTimeout(r, 2000 * k));
    const r = await fetch(STATS_URL, { method: "POST", headers: { Authorization: "Bearer " + t, "Content-Type": "application/json" }, body: JSON.stringify(body) });
    if (r.ok) return ((await r.json()).data || []) as any[];
    last = "stats_" + r.status + ":" + (await r.text()).slice(0, 160);
    if (r.status === 429 || r.status === 503) throw new Error("cdse_busy");
    if (r.status < 500) break;
  }
  throw new Error(last);
}
const S2 = (l1c: boolean) => `//VERSION=3
function setup(){return{input:[{bands:["B02","B03","B04","B08","B8A","B11",${l1c ? "" : "\"SCL\","}"dataMask"]}],output:[{id:"ix",bands:5,sampleType:"FLOAT32"},{id:"dataMask",bands:1}]}}
function evaluatePixel(s){var ok=s.dataMask;${l1c ? "if(s.B02>0.3&&s.B03>0.3)ok=0;" : "if(s.SCL==3||s.SCL==8||s.SCL==9||s.SCL==10||s.SCL==0||s.SCL==1)ok=0;"}
var nd=(s.B08-s.B04)/(s.B08+s.B04+1e-6),nm=(s.B8A-s.B11)/(s.B8A+s.B11+1e-6),bs=((s.B11+s.B04)-(s.B08+s.B02))/((s.B11+s.B04)+(s.B08+s.B02)+1e-6),mw=(s.B03-s.B11)/(s.B03+s.B11+1e-6);
return{ix:[nd>=0.25?1:0,nd,nm,bs,mw>0.1?1:0],dataMask:[ok]}}`;
const S1 = `//VERSION=3
function setup(){return{input:[{bands:["VV","dataMask"]}],output:[{id:"vv",bands:1,sampleType:"FLOAT32"},{id:"dataMask",bands:1}]}}
function evaluatePixel(s){return{vv:[10*Math.log(Math.max(s.VV,1e-5))/Math.LN10],dataMask:[s.dataMask]}}`;
const ring = (P: number[][]) => { const r = P.map((p) => [Number(p[1]), Number(p[0])]); if (r[0][0] !== r[r.length - 1][0] || r[0][1] !== r[r.length - 1][1]) r.push(r[0]); return r; };
const area = (r: number[][]) => { let s = 0; for (let i = 0; i < r.length - 1; i++) s += r[i][0] * r[i + 1][1] - r[i + 1][0] * r[i][1]; return s / 2; };

export async function handle(b: any) {
  const P: number[][] = Array.isArray(b.poly) ? b.poly : [];
  if (P.length < 3 || P.length > 400 || P.some((p) => !isFinite(+p[0]) || !isFinite(+p[1]) || +p[0] < 20 || +p[0] > 32 || +p[1] < 24 || +p[1] > 37)) return json({ error: "bad_poly" }, 400);
  const la0 = +b.well?.la, lo0 = +b.well?.lo; const R = ring(P);
  const lons = R.map((p) => p[0]), lats = R.map((p) => p[1]); const cla = (Math.min(...lats) + Math.max(...lats)) / 2, kx = 111320 * Math.cos(cla * Math.PI / 180), ky = 110540;
  if ((Math.max(...lons) - Math.min(...lons)) * kx > 6000 || (Math.max(...lats) - Math.min(...lats)) * ky > 6000) return json({ error: "too_big" }, 400);
  const m = 300, dx = m / kx, dy = m / ky; const x0 = Math.min(...lons) - dx, x1 = Math.max(...lons) + dx, y0 = Math.min(...lats) - dy, y1 = Math.max(...lats) + dy;
  const hole = area(R) > 0 ? R.slice().reverse() : R;
  const G: Record<string, unknown> = { land: { type: "Polygon", coordinates: [R] }, ring: { type: "Polygon", coordinates: [[[x0, y0], [x1, y0], [x1, y1], [x0, y1], [x0, y0]], hole] } };
  const wla = isFinite(la0) ? la0 : cla, wlo = isFinite(lo0) ? lo0 : (Math.min(...lons) + Math.max(...lons)) / 2; const h = 15;
  G.well = { type: "Polygon", coordinates: [[[wlo - h / kx, wla - h / ky], [wlo + h / kx, wla - h / ky], [wlo + h / kx, wla + h / ky], [wlo - h / kx, wla + h / ky], [wlo - h / kx, wla - h / ky]]] };
  const to = new Date(Date.now() - 3 * 864e5).toISOString().slice(0, 10);
  const t = await cdseToken();
  const res = 10;
  const s2 = (geom: unknown, from: string, to_: string, l1c: boolean) => stats(t, {
    input: { bounds: { geometry: geom, properties: { crs: CRS } }, data: [{ type: l1c ? "sentinel-2-l1c" : "sentinel-2-l2a", dataFilter: { maxCloudCoverage: 60 } }] },
    aggregation: { timeRange: { from: from + "T00:00:00Z", to: to_ + "T23:59:59Z" }, aggregationInterval: { of: "P1M" }, resx: res / kx, resy: res / ky, evalscript: S2(l1c) },
    calculations: { default: {} },
  });
  const s1 = (geom: unknown) => stats(t, {
    input: { bounds: { geometry: geom, properties: { crs: CRS } }, data: [{ type: "sentinel-1-grd", dataFilter: { acquisitionMode: "IW", polarization: "DV", resolution: "HIGH" }, processing: { backCoeff: "SIGMA0_ELLIPSOID", orthorectify: true, demInstance: "COPERNICUS" } }] },
    aggregation: { timeRange: { from: "2016-01-01T00:00:00Z", to: to + "T23:59:59Z" }, aggregationInterval: { of: "P1M" }, resx: 20 / kx, resy: 20 / ky, evalscript: S1 },
    calculations: { default: {} },
  });
  const pick = (rows: any[]) => rows.map((it) => { const B = it?.outputs?.ix?.bands; const s = B?.B0?.stats; if (!s || !(s.sampleCount > (s.noDataCount || 0))) return null; const valid = 1 - (s.noDataCount || 0) / s.sampleCount; const g = (k: string) => { const v = B?.[k]?.stats?.mean; return isFinite(v) ? +Number(v).toFixed(4) : null; }; return [String(it.interval?.from || "").slice(0, 7), g("B0"), g("B1"), g("B2"), g("B3"), g("B4"), +valid.toFixed(2)]; }).filter((x) => x && (x[6] as number) > 0.3);
  const pv = (rows: any[]) => rows.map((it) => { const s = it?.outputs?.vv?.bands?.B0?.stats; return s && s.sampleCount > (s.noDataCount || 0) && isFinite(s.mean) ? [String(it.interval?.from || "").slice(0, 7), +Number(s.mean).toFixed(2)] : null; }).filter(Boolean);
  const jobs: Record<string, Promise<any[]>> = {};
  for (const g of ["land", "ring", "well"]) { jobs[g + "A"] = s2(G[g], "2016-01-01", "2018-12-31", true); jobs[g + "B"] = s2(G[g], "2019-01-01", to, false); }
  jobs.vvL = s1(G.land); jobs.vvR = s1(G.ring);
  const keys = Object.keys(jobs); const out: Record<string, any[]> = {}; const errs: string[] = [];
  const rs = await Promise.allSettled(keys.map((k) => jobs[k]));
  rs.forEach((r, i) => { if (r.status === "fulfilled") out[keys[i]] = r.value; else errs.push(keys[i] + ":" + String((r.reason as Error)?.message || r.reason).slice(0, 90)); });
  if (errs.some((e) => /cdse_busy/.test(e))) return json({ error: "cdse_busy" }, 503);
  const S: Record<string, any[]> = {};
  for (const g of ["land", "ring", "well"]) S[g] = [...pick(out[g + "A"] || []), ...pick(out[g + "B"] || [])];
  return json({ ok: true, cols: ["m", "veg", "ndvi", "ndmi", "bsi", "wat", "valid"], s2: S, vv: { land: pv(out.vvL || []), ring: pv(out.vvR || []) }, errs, to });
}

Deno.serve(async (req) => {
  if (req.method === "OPTIONS") return new Response("ok", { headers: CORS });
  try {
    const auth = req.headers.get("Authorization") || "";
    const sb = createClient(Deno.env.get("SUPABASE_URL")!, Deno.env.get("SUPABASE_ANON_KEY")!, { global: { headers: { Authorization: auth, "x-dev-key": req.headers.get("x-dev-key") || "" } } });
    const { data: role, error: re } = await sb.rpc("my_role");
    if (re || role === null || ![0, 1, 2, 4, 5, 7].includes(Number(role))) return json({ error: "forbidden" }, 403);
    return await handle(await req.json());
  } catch (e) {
    const m = String((e as Error).message || e);
    return json({ error: m.split(":")[0], detail: m.slice(0, 200) }, /cdse_busy/.test(m) ? 503 : 500);
  }
});
