import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { mockDesktop } from './mockDesktop';

const now = '2026-11-14T00:00:00.000Z';
const statusID = 652787832;
const releaseID = 4128974838;
const installedID = 2767927275;
const locale = (lang: string) => JSON.parse(readFileSync(new URL(`../public/locales/${lang}.json`, import.meta.url), 'utf8')).platform;
const ja = locale('ja');
const image = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j1ioAAAAASUVORK5CYII=';
const announcement = { id: 'fixture-pinned', title: '固定の運営告知', intro: '公開告知の本文', image,
  url: 'https://register.bassaps.com/announcements#fixture-pinned', time: '2026-11-14', timestamp: 1794614400, type: 'notice', pinned: true };

function statusFixture() {
  return { overallStatus: 'operational', generatedAt: now, updatedAt: now,
    services: ['portal', 'account', 'download', 'docs', 'm7h-http', 'm7h-game'].map(id => ({
      id, name: id, enabled: id !== 'docs' && id !== 'm7h-game', monitoringType: id === 'm7h-game' ? 'udp' : 'http',
      status: id === 'docs' || id === 'm7h-game' ? 'not_monitored' : 'operational',
      lastCheckedAt: id === 'docs' || id === 'm7h-game' ? null : now, responseTimeMs: null, samples24H: 1, hasIncident: false,
    })) };
}
function releaseFixture(assetState = 'not_provided') {
  return { latest: { id: 1, tag: 'First', version: 'v9.0.0', name: 'Fixture release', publishedAt: now,
    url: 'https://github.com/BassaPS/bassaPSLauncher/releases/tag/First' } as { id: number; tag: string; version: string; name: string; publishedAt: string; url: string } | null,
    recommendedAsset: assetState === 'available' ? { id: 2, name: 'BASSAPSLauncher-fixture-windows-x64.zip', size: 1024,
      url: 'https://github.com/BassaPS/bassaPSLauncher/releases/download/First/BASSAPSLauncher-fixture-windows-x64.zip', digest: 'sha256:' + 'a'.repeat(64) } : null,
    assetState, fetchedAt: now, stale: false, downloadUrl: 'https://download.bassaps.com/' };
}

async function setup(page: Page, options: {
  status?: ReturnType<typeof statusFixture>; release?: ReturnType<typeof releaseFixture>;
  statusFails?: boolean; releaseFails?: boolean; installedFails?: boolean; pending?: boolean; lang?: string;
} = {}) {
  await page.clock.install({ time: new Date(now) });
  await mockDesktop(page, true, [announcement]);
  if (options.lang) await page.addInitScript(lang => localStorage.setItem('i18nextLng', lang), options.lang);
  const calls: number[] = [], opened: string[] = [];
  const control = { statusFails: options.statusFails ?? false, releaseFails: options.releaseFails ?? false };
  let finish: (() => Promise<void>) | undefined;
  await page.route('**/wails/runtime**', route => {
    const url = new URL(route.request().url());
    const body = route.request().method() === 'POST' ? route.request().postDataJSON() : {
      object: Number(url.searchParams.get('object')), args: JSON.parse(url.searchParams.get('args') || '{}'),
    };
    if (body.object !== 0) return route.fallback();
    const id = body.args.methodID; calls.push(id);
    if (id === statusID) {
      if (options.pending && calls.filter(id => id === statusID).length === 1) {
        finish = () => route.fulfill({ json: options.status ?? statusFixture() });
        return;
      }
      return control.statusFails ? route.fulfill({ status: 503, body: 'mock failure' }) : route.fulfill({ json: options.status ?? statusFixture() });
    }
    if (id === releaseID) return control.releaseFails ? route.fulfill({ status: 503, body: 'mock failure' }) : route.fulfill({ json: options.release ?? releaseFixture() });
    if (id === installedID && options.installedFails) return route.fulfill({ json: [false, ''] });
    if (id === 1132232877) { opened.push(body.args.args[0]); return route.fulfill({ json: [true, ''] }); }
    return route.fallback();
  });
  return { calls, opened, control, finish: () => finish?.() };
}
const card = (page: Page) => page.getByRole('region', { name: ja.statusTitle });
const releaseCard = (page: Page) => page.getByRole('region', { name: ja.releaseTitle });

test('Home loading, recovery and game controls stay independent; no Release subscription', async ({ page }) => {
  const backend = await setup(page, { pending: true });
  await page.goto('/');
  await expect(card(page).getByText(ja.loading)).toBeVisible();
  await expect(page.getByRole('button', { name: 'ゲーム開始', exact: true })).toBeEnabled();
  await backend.finish();
  await expect(card(page).locator('[data-service="m7h-http"] strong')).toHaveText(ja.states.operational);
  await expect(card(page).locator('[data-service="m7h-game"] strong')).toHaveText(ja.states.not_monitored);
  await expect(card(page).locator('[data-service="docs"] strong')).toHaveText(ja.states.not_monitored);
  await expect(card(page).getByText(ja.insufficient, { exact: false }).first()).toBeVisible();
  await expect(page.getByRole('button', { name: /固定の運営告知/ })).toBeVisible();
  expect(backend.calls.filter(id => id === releaseID)).toHaveLength(0);
  backend.control.statusFails = true;
  await card(page).getByRole('button', { name: ja.refresh }).click();
  await expect(card(page).getByText(ja.statusError)).toBeVisible();
  await expect(page.getByRole('button', { name: 'ゲーム開始', exact: true })).toBeEnabled();
  await page.screenshot({ path: test.info().outputPath('home-api-failure.png'), fullPage: true });
  backend.control.statusFails = false;
  await card(page).getByRole('button', { name: ja.refresh }).click();
  await expect(card(page).getByText(ja.statusError)).toHaveCount(0);
});

for (const state of ['operational', 'degraded', 'partial_outage', 'major_outage', 'maintenance', 'unknown', 'not_monitored']) {
  test(`Home preserves ${state} and Docs enabled from DTO`, async ({ page }) => {
    const data = statusFixture(); data.overallStatus = state;
    const docs = data.services.find(s => s.id === 'docs')!;
    Object.assign(docs, { enabled: state !== 'not_monitored', status: state, lastCheckedAt: now, samples24H: 21 });
    await setup(page, { status: data }); await page.goto('/');
    await expect(card(page).locator('[data-service="docs"] strong')).toHaveText(ja.states[state]);
    await expect(card(page).locator('.bassaps-platform-overall strong')).toHaveText(ja.states[state]);
    if (['operational', 'not_monitored', 'major_outage', 'maintenance'].includes(state)) {
      await page.screenshot({ path: test.info().outputPath(`home-${state}.png`), fullPage: true });
    }
  });
}

for (const variant of ['boundary', 'old', 'no-observation', 'future', 'maintenance', 'incident']) {
  test(`Home observation ${variant}`, async ({ page }) => {
    const data = statusFixture(), row = data.services.find(s => s.id === 'm7h-http')!;
    row.lastCheckedAt = variant === 'no-observation' ? null : new Date(Date.parse(now) + (variant === 'future' ? 120000 : -(variant === 'boundary' ? 450000 : 450001))).toISOString();
    if (variant === 'maintenance') row.status = 'maintenance';
    if (variant === 'incident') { row.status = 'partial_outage'; row.hasIncident = true; }
    await setup(page, { status: data });
    if (variant === 'boundary') await page.addInitScript(time => { Date.now = () => time; }, Date.parse(now));
    await page.goto('/');
    const target = card(page).locator('[data-service="m7h-http"]');
    await expect(target.locator('strong')).toHaveText(ja.states[variant === 'boundary' ? 'operational' : variant === 'maintenance' ? 'maintenance' : variant === 'incident' ? 'partial_outage' : 'unknown']);
    if (variant === 'no-observation') await expect(target).toContainText(ja.noObservation);
    if (['maintenance', 'incident'].includes(variant)) {
      await expect(target).toContainText(ja.oldObservation); await expect(target).toContainText(ja.declared);
    }
    if (variant === 'boundary') {
      await page.evaluate(time => { Date.now = () => time; }, Date.parse(now) + 1);
      await page.clock.runFor(1);
      await expect(target.locator('strong')).toHaveText(ja.states.unknown);
    }
  });
}

for (const mode of ['not_provided', 'none', 'available', 'empty', 'stale', 'expired', 'error', 'installed-error']) {
  test(`About ${mode}, independent resources and disabled updater`, async ({ page }) => {
    const release = releaseFixture(mode);
    if (mode === 'empty') release.latest = null;
    if (mode === 'stale') release.stale = true;
    if (mode === 'expired') release.fetchedAt = new Date(Date.parse(now) - 604800000).toISOString();
    const backend = await setup(page, { release, releaseFails: mode === 'error', installedFails: mode === 'installed-error' });
    await page.goto('/about');
    const region = releaseCard(page);
    await expect(region.getByText(mode === 'installed-error' ? ja.installedError : '1.1.2', { exact: true })).toBeVisible();
    if (mode === 'expired') {
      await expect(region.getByText(ja.expired)).toBeVisible(); await expect(region.getByText('v9.0.0')).toHaveCount(0);
    } else if (mode === 'error') {
      await expect(region.getByText(ja.releaseError)).toBeVisible(); await expect(region.getByText('v9.0.0')).toHaveCount(0);
    } else if (mode === 'empty') await expect(region.getByText(ja.noRelease)).toBeVisible();
    else {
      await expect(region.getByText('v9.0.0', { exact: true })).toBeVisible();
      await expect(region.getByText('タグ: First')).toBeVisible();
      if (mode === 'not_provided') await expect(region.getByText(ja.noAssetInfo)).toBeVisible();
      if (mode === 'none') await expect(region.getByText(ja.noAsset)).toBeVisible();
      if (mode === 'available') { await expect(region.getByText(/BASSAPSLauncher-fixture-windows-x64.zip/)).toBeVisible(); await expect(region.getByText(ja.digest, { exact: false })).toBeVisible(); }
      if (mode === 'stale') await expect(region.getByText(ja.oldRelease)).toBeVisible();
      await expect(region.locator('time')).toHaveAttribute('datetime', release.fetchedAt);
    }
    expect(backend.calls.filter(id => id === statusID)).toHaveLength(0);
    expect(backend.calls.filter(id => id === releaseID)).toHaveLength(1);
    await expect(page.getByRole('dialog', { name: /アップデート/ })).toHaveCount(0);
    await region.getByRole('button', { name: 'Download Center' }).click();
    expect(backend.opened).toEqual(['https://download.bassaps.com/']);
    if (mode === 'error') {
      backend.control.releaseFails = false;
      await region.getByRole('button', { name: ja.refresh }).click();
      await expect(region.getByText('v9.0.0')).toBeVisible();
    }
  });
}

test('Release becomes stale then expires while held; failed refresh never fakes checked time', async ({ page }) => {
  const backend = await setup(page); await page.goto('/about');
  await expect(releaseCard(page).getByText('v9.0.0')).toBeVisible();
  await page.clock.runFor(600000);
  await expect(releaseCard(page).getByText(ja.oldRelease)).toBeVisible();
  backend.control.releaseFails = true;
  await releaseCard(page).getByRole('button', { name: ja.refresh }).click();
  await expect(releaseCard(page).getByText(ja.releaseError)).toBeVisible();
  await expect(releaseCard(page).locator('time')).toHaveAttribute('datetime', now);
  await page.clock.fastForward(604800000);
  await expect(releaseCard(page).getByText(ja.expired)).toBeVisible();
  await expect(releaseCard(page).getByText('v9.0.0')).toHaveCount(0);
  expect(backend.calls.filter(id => id === releaseID)).toHaveLength(2);
});

test('all fixed service links and Releases use OpenURL; navigation shares resources', async ({ page }) => {
  const backend = await setup(page); await page.goto('/');
  await expect(card(page).locator('[data-service="m7h-http"] strong')).toHaveText(ja.states.operational);
  const nav = page.locator('.bassaps-sidebar').getByRole('navigation', { name: ja.services });
  for (const id of ['portal', 'news', 'download', 'docs', 'status', 'account']) await nav.getByRole('button', { name: ja.links[id], exact: true }).click();
  expect(backend.opened).toEqual(['https://bassaps.com/', 'https://bassaps.com/news', 'https://download.bassaps.com/', 'https://docs.bassaps.com/', 'https://status.bassaps.com/', 'https://register.bassaps.com/']);
  await page.locator('.bassaps-sidebar').getByRole('link', { name: 'このアプリについて' }).click();
  await expect(releaseCard(page).getByText('v9.0.0')).toBeVisible();
  await releaseCard(page).getByRole('button', { name: 'GitHub Releases' }).click();
  expect(backend.opened.at(-1)).toBe('https://github.com/BassaPS/bassaPSLauncher/releases');
  await page.locator('.bassaps-sidebar').getByRole('link', { name: 'ホーム', exact: true }).click();
  await expect(card(page)).toBeVisible();
  expect(backend.calls.filter(id => id === statusID)).toHaveLength(1);
  expect(backend.calls.filter(id => id === releaseID)).toHaveLength(1);
});

for (const lang of ['ja', 'en', 'zh', 'ko', 'vi', 'fr']) {
  test(`locale ${lang} / fallback renders new text`, async ({ page }) => {
    const strings = locale(lang === 'fr' ? 'en' : lang);
    await setup(page, { lang }); await page.goto('/about');
    await expect(page.getByRole('region', { name: strings.releaseTitle })).toBeVisible();
    await expect(page.getByText(strings.noAssetInfo, { exact: true })).toBeVisible();
    await expect(page.locator('body')).not.toContainText('platform.');
    if (lang === 'en') await page.screenshot({ path: test.info().outputPath('about-en.png'), fullPage: true });
  });
}

for (const [width, height, theme] of [[320,560,'light'],[768,720,'dark'],[1280,720,'light'],[1280,720,'dark'],[1920,1080,'system']] as const) {
  test(`layout ${width}x${height} ${theme}, focus and screenshots`, async ({ page }) => {
    const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
    await page.setViewportSize({ width, height }); await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'reduce' });
    await setup(page, { release: releaseFixture('available') }); await page.goto('/');
    await page.getByRole('banner').getByLabel('カラーテーマ').selectOption(theme);
    await expect(page.locator('html')).toHaveAttribute('data-theme', theme === 'light' ? 'bassaps-light' : 'bassaps');
    await expect(card(page).locator('[data-service="m7h-http"] strong')).toHaveText(ja.states.operational);
    await expect(page.getByRole('button', { name: 'ゲーム開始', exact: true })).toBeEnabled();
    const overflow = await page.evaluate(() => ({ inner: innerWidth, viewport: visualViewport?.width, client: document.documentElement.clientWidth, elements: [...document.querySelectorAll('*')].filter(e => e.scrollWidth > e.clientWidth + 1).map(e => ({ tag: e.tagName, cls: e.className, client: e.clientWidth, scroll: e.scrollWidth, rect: e.getBoundingClientRect().right })).slice(0, 15) }));
    expect(await page.evaluate(() => document.documentElement.scrollWidth), JSON.stringify(overflow)).toBeLessThanOrEqual(width);
    await page.evaluate(() => { (document.activeElement as HTMLElement | null)?.blur(); window.scrollTo(0, 0); });
    await page.screenshot({ path: test.info().outputPath(`home-window-${width}-${theme}.png`), animations: 'disabled' });
    await page.screenshot({ path: test.info().outputPath(`home-${width}-${theme}.png`), fullPage: true, animations: 'disabled' });
    if (width < 761) {
      await page.getByRole('button', { name: 'メニューを開く' }).click();
      await expect(page.getByRole('dialog', { name: 'ナビゲーション' }).getByRole('navigation', { name: ja.services })).toBeVisible();
      await page.getByRole('dialog', { name: 'ナビゲーション' }).getByRole('link', { name: 'このアプリについて' }).click();
    } else await page.locator('.bassaps-sidebar').getByRole('link', { name: 'このアプリについて' }).click();
    await expect(releaseCard(page).getByText(/BASSAPSLauncher-fixture/)).toBeVisible();
    const link = releaseCard(page).getByRole('button', { name: 'Download Center' });
    await link.focus(); await page.keyboard.press('Tab'); await expect(releaseCard(page).getByRole('button', { name: 'GitHub Releases' })).toBeFocused();
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width);
    await page.screenshot({ path: test.info().outputPath(`about-${width}-${theme}.png`), fullPage: true, animations: 'disabled' });
    expect(errors).toEqual([]);
  });
}

test('existing NewsWidget, official/server tabs, image, ordering, read state and offline are preserved', async ({ page }) => {
  await setup(page);
  let offline = false;
  await page.route('**/wails/runtime**', route => {
    const body = route.request().postDataJSON();
    if (body.object !== 0) return route.fallback();
    if (body.args.methodID === 1638005996) return route.fulfill({ json: [true, [{ ...announcement, id: 'official', title: '公式ニュース', pinned: false }], ''] });
    if (body.args.methodID === 3670222014) return route.fulfill({ json: offline ? [false, [], 'fixture offline'] : [true, [announcement, { ...announcement, id: 'ordinary', title: '通常のお知らせ', pinned: false }], ''] });
    return route.fallback();
  });
  await page.goto('/'); await expect(page.getByRole('button', { name: /固定の運営告知/ })).toBeVisible();
  await page.goto('/news'); await expect(page.getByRole('heading', { name: '公式ニュース' })).toBeVisible();
  await page.getByRole('button', { name: 'サーバー', exact: true }).click();
  await expect(page.locator('h3').first()).toHaveText('固定の運営告知');
  await expect(page.getByRole('img', { name: '固定の運営告知' })).toBeVisible();
  await expect(page.getByText('2026-11-14', { exact: true }).first()).toBeVisible();
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem('news-storage')!).state.lastReadServerTimestamp)).toBe(announcement.timestamp);
  offline = true; await page.getByRole('button', { name: '更新', exact: true }).click();
  await expect(page.getByText('fixture offline', { exact: true }).first()).toBeVisible();
});

for (const target of ['hoyotoon', 'custom', 'local_test']) {
  test(`initial Status failure does not gate ${target} game launch (mock only)`, async ({ page }) => {
    await setup(page, { statusFails: true });
    await page.addInitScript(target => {
      const saved = JSON.parse(localStorage.getItem('setting-storage')!);
      Object.assign(saved.state, { serverTarget: target, patchTargetUrl: 'http://fixture.invalid:21000' });
      localStorage.setItem('setting-storage', JSON.stringify(saved));
    }, target);
    const localID = Number(readFileSync(new URL('../bindings/cyrene-launcher/internal/march7thhoney-service/march7thhoneyservice.ts', import.meta.url), 'utf8').split('export function StartLocalServer(')[1].match(/ByID\((\d+)/)![1]);
    let launched = '';
    await page.route('**/wails/runtime**', route => {
      const body = route.request().postDataJSON();
      if (body.object !== 0) return route.fallback();
      if (body.args.methodID === localID) return route.fulfill({ json: [true, ''] });
      if (body.args.methodID === 3347991480) { launched = body.args.args[1]; return route.fulfill({ json: [true, ''] }); }
      return route.fallback();
    });
    await page.goto('/'); await expect(card(page).getByText(ja.statusError)).toBeVisible();
    await expect(card(page).locator('.bassaps-platform-overall')).toHaveCount(0);
    await page.getByRole('button', { name: 'ゲーム開始', exact: true }).click();
    await expect.poll(() => launched).toBe(target === 'custom' ? 'http://fixture.invalid:21000' : target === 'local_test' ? 'http://127.0.0.1:21000' : 'http://hsr.bassaps.com:21000');
  });
}

test('existing language switch and native minimize/close dispatch remain intact (mock only)', async ({ page }) => {
  await setup(page);
  const native: number[] = [];
  await page.route('**/wails/runtime**', route => {
    const body = route.request().postDataJSON();
    if (body.object === 0 && [1985260791, 4287907331].includes(body.args.methodID)) {
      native.push(body.args.methodID); return route.fulfill({ json: [true, ''] });
    }
    return route.fallback();
  });
  await page.goto('/about');
  await page.getByRole('banner').getByRole('button', { name: '言語 JA', exact: true }).click();
  await page.getByRole('button', { name: 'English', exact: true }).click();
  await expect(page.getByRole('region', { name: locale('en').releaseTitle })).toBeVisible();
  await page.getByRole('banner').getByRole('button', { name: '最小化', exact: true }).click();
  await expect.poll(() => native).toContain(1985260791);
  await page.getByRole('banner').getByRole('button', { name: '閉じる', exact: true }).click();
  await page.locator('dialog.bassaps-modal-frame[open]').getByRole('button', { name: 'Close', exact: true }).click();
  await expect.poll(() => native).toContain(4287907331);
});
