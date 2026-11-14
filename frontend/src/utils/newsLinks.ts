import { NewsService } from "@bindings/cyrene-launcher/internal/news-service";

export type NewsSource = 'register' | 'official';
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

// UI affordance only; Go repeats the authoritative source-specific validation.
function parse(raw: string): URL | null {
  try {
    if (raw.length > 2048 || /[\s\p{Cc}\p{Cf}\\]/u.test(raw) || /%(?:2f|5c|25|2e)/i.test(raw)) return null;
    const u = new URL(raw);
    return u.protocol === 'https:' && !u.username && !u.password && !u.port && u.href === raw ? u : null;
  } catch { return null; }
}

export function canOpenNews(source: NewsSource, id: string, raw: string): boolean {
  const u = parse(raw);
  if (!u) return false;
  if (source === 'register') return uuid.test(id) && raw === `https://register.bassaps.com/announcements#${id}`;
  if (!/^[A-Za-z0-9_-]{1,64}$/.test(id) || u.hash) return false;
  if (u.hostname === 'hoyo.link') return /^\/[A-Za-z0-9_-]{1,64}$/.test(u.pathname) && !u.search;
  if (u.hostname === 'youtu.be') return /^\/[A-Za-z0-9_-]{11}$/.test(u.pathname) && !u.search;
  return u.hostname === 'www.hoyolab.com' && /^\/article\/[0-9]{1,20}$/.test(u.pathname) && (!u.search || /^\?utm_id=[0-9]{1,20}$/.test(u.search));
}

export function canLoadNewsImage(source: NewsSource, id: string, raw: string): boolean {
  const u = parse(raw);
  if (!u || u.search || u.hash) return false;
  if (source === 'register') return uuid.test(id) && u.hostname === 'register.bassaps.com' && /^\/api\/files\/[0-9a-f-]{36}$/.test(u.pathname) && uuid.test(u.pathname.slice('/api/files/'.length));
  return /^[A-Za-z0-9_-]{1,64}$/.test(id) && u.hostname === 'fastcdn.hoyoverse.com' && /^\/static-resource-v2\/[A-Za-z0-9_./-]+\.(png|jpg|jpeg)$/.test(u.pathname);
}

export async function openNews(source: NewsSource, id: string, url: string): Promise<boolean> {
  if (!canOpenNews(source,id,url)) return false;
  try { return (await NewsService.OpenNewsURL(source,id,url))[0]; }
  catch { return false; }
}

export function isNewsRaster(data: string): boolean {
  return data.length <= 4 * 1024 * 1024 + 32 && /^data:image\/(png|jpeg);base64,[A-Za-z0-9+/]+=*$/.test(data);
}
