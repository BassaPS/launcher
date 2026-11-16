import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';

function method(file: string, name: string): number {
  const source = readFileSync(new URL(file, import.meta.url), 'utf8');
  const section = source.slice(source.indexOf(`export function ${name}(`));
  const match = section.match(/\$Call.ByID\((\d+)/);
  if (!match) throw new Error(`Missing generated method ${name}`);
  return Number(match[1]);
}
const platformBinding = '../bindings/cyrene-launcher/internal/platform-service/platformservice.ts';
const statusID = method(platformBinding, 'GetStatus');
const releaseID = method(platformBinding, 'GetLatestRelease');
const installedID = method('../bindings/cyrene-launcher/internal/app-service/appservice.ts', 'GetCurrentLauncherVersion');
const time = '2026-11-13T12:00:00.000Z';

function status() {
  return { overallStatus: 'operational', updatedAt: time, generatedAt: time,
    services: ['portal', 'account', 'download', 'docs', 'm7h-http', 'm7h-game'].map(id => ({
      id, name: id, monitoringType: id === 'm7h-game' ? 'udp' : 'http', enabled: id !== 'docs' && id !== 'm7h-game',
      status: id === 'docs' || id === 'm7h-game' ? 'not_monitored' : 'operational',
      lastCheckedAt: id === 'm7h-game' ? null : time, responseTimeMs: null, samples24H: 1, hasIncident: false,
    })) };
}
function release(assetState = 'not_provided') {
  return { latest: { id: 1, tag: 'First', version: 'v1.1.2', name: 'Launcher', publishedAt: time,
    url: 'https://github.com/BassaPS/launcher/releases/tag/First' },
    recommendedAsset: assetState === 'available' ? { id: 2, name: 'Launcher.zip', size: 100,
      url: 'https://github.com/BassaPS/launcher/releases/download/First/Launcher.zip', digest: null } : null,
    assetState, fetchedAt: time, stale: false, downloadUrl: 'https://download.bassaps.com/' };
}

async function mock(page: Page, options: { assetState?: string; releaseFails?: boolean; delayedStatus?: boolean } = {}) {
  const calls: number[] = [];
  const cancellations: string[] = [];
  let finishFirst: (() => Promise<void>) | undefined;
  await page.clock.install({ time: new Date(time) });
  await page.route('**/wails/custom.js', route => route.fulfill({ status: 404, body: '' }));
  await page.route('**/wails/runtime**', async route => {
    const body = route.request().postDataJSON();
    const args = body.args;
    if (body.object !== 0) {
      if (args?.['call-id']) cancellations.push(args['call-id']);
      return route.fulfill({ json: null });
    }
    const id = args.methodID;
    calls.push(id);
    if (id === statusID) {
      if (options.delayedStatus && calls.filter(value => value === statusID).length === 1) {
        const old = status(); old.overallStatus = 'major_outage';
        finishFirst = () => route.fulfill({ json: old });
        return;
      }
      return route.fulfill({ json: status() });
    }
    if (id === releaseID) return options.releaseFails ? route.fulfill({ status: 503, body: 'fixture error' }) : route.fulfill({ json: release(options.assetState) });
    if (id === installedID) return route.fulfill({ json: [true, '1.1.2'] });
    throw new Error(`Unexpected binding call ${id}`);
  });
  return { calls, cancellations, finishFirst: () => finishFirst?.() };
}

async function state(page: Page) {
  return JSON.parse(await page.getByTestId('consumer-0').textContent() ?? '{}');
}

test('StrictMode and two consumers share calls and preserve legacy release metadata', async ({ page }) => {
  const backend = await mock(page);
  await page.goto('/__platform-test.html');
  await expect.poll(async () => (await state(page)).release.phase).toBe('success');
  const data = await state(page);
  expect(data.release.data.assetState).toBe('not_provided');
  expect(data.installedVersion.data).toBe('1.1.2');
  expect(data.services.find((s: { id: string }) => s.id === 'm7h-game').status).toBe('not_monitored');
  expect(data.services.find((s: { id: string }) => s.id === 'docs').status).toBe('not_monitored');
  expect(backend.calls.filter(id => id === statusID)).toHaveLength(1);
  expect(backend.calls.filter(id => id === releaseID)).toHaveLength(1);
  await page.getByRole('button', { name: 'one', exact: true }).click();
  await page.clock.runFor(5);
  expect(backend.cancellations).toHaveLength(0);
  await page.getByRole('button', { name: 'hidden', exact: true }).click();
  await page.clock.runFor(60_000);
  expect(backend.calls.filter(id => id === statusID)).toHaveLength(1);
  await page.getByRole('button', { name: 'visible', exact: true }).click();
  await page.clock.runFor(5);
  await expect.poll(() => backend.calls.filter(id => id === statusID).length).toBe(2);
  expect(backend.calls.filter(id => id === releaseID)).toHaveLength(1);
});

for (const assetState of ['none', 'available']) {
  test(`new release DTO preserves ${assetState}`, async ({ page }) => {
    await mock(page, { assetState });
    await page.goto('/__platform-test.html');
    await expect.poll(async () => (await state(page)).release.data?.assetState).toBe(assetState);
    await page.getByRole('button', { name: 'retry release', exact: true }).first().click();
    await expect.poll(async () => (await state(page)).release.phase).toBe('success');
  });
}

test('release failure is independent of status and installed version', async ({ page }) => {
  await mock(page, { releaseFails: true });
  await page.goto('/__platform-test.html');
  await expect.poll(async () => (await state(page)).release.error).toBe(true);
  expect((await state(page)).status.phase).toBe('success');
  expect((await state(page)).installedVersion.data).toBe('1.1.2');
});

test('unmount cancellation prevents a late response replacing a newer call', async ({ page }) => {
  const backend = await mock(page, { delayedStatus: true });
  await page.goto('/__platform-test.html');
  await expect.poll(() => backend.calls.filter(id => id === statusID).length).toBe(1);
  await page.getByRole('button', { name: 'none', exact: true }).click();
  await page.clock.runFor(5);
  await expect.poll(() => backend.cancellations.length).toBeGreaterThan(0);
  await page.getByRole('button', { name: 'two', exact: true }).click();
  await page.clock.runFor(5);
  await page.getByRole('button', { name: 'retry status', exact: true }).first().click();
  await expect.poll(async () => (await state(page)).status.phase).toBe('success');
  await backend.finishFirst();
  await page.clock.runFor(5);
  expect((await state(page)).overallStatus).toBe('operational');
});

test('hidden elapsed time re-evaluates retained status and release freshness', async ({ page }) => {
  await mock(page);
  await page.goto('/__platform-test.html');
  await expect.poll(async () => (await state(page)).release.phase).toBe('success');
  await page.getByRole('button', { name: 'hidden', exact: true }).click();
  await page.clock.runFor(600_000);
  await page.getByRole('button', { name: 'visible', exact: true }).click();
  await expect.poll(async () => (await state(page)).release.phase).toBe('stale');
  expect((await state(page)).overallStatus).toBe('unknown');
  await page.getByRole('button', { name: 'hidden', exact: true }).click();
  await page.clock.runFor(7 * 86_400_000);
  await page.getByRole('button', { name: 'visible', exact: true }).click();
  await expect.poll(async () => (await state(page)).release.expired).toBe(true);
  expect((await state(page)).release.data).toBe(null);
});
