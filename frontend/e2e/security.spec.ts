import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { mockDesktop } from './mockDesktop';

const strings = (language: string) => JSON.parse(readFileSync(new URL(`../public/locales/${language}.json`, import.meta.url), 'utf8')).platform;
const cases: [string, string, string][] = [
  ['en', 'en', 'en'], ['ja', 'ja', 'ja'], ['zh', 'zh', 'zh'], ['ko', 'ko', 'ko'], ['vi', 'vi', 'vi'],
  ['region', 'ja-JP', 'ja'], ['script-region', 'zh-Hant-TW', 'zh'], ['underscore', 'en_US', 'en'],
  ['fallback', 'fr', 'en'], ['unknown', 'unknown-language', 'en'], ['parent', '../', 'en'],
  ['encoded', '%2e%2e%2f', 'en'], ['double-encoded', '%252e%252e%252f', 'en'],
  ['external', 'https://fixture.invalid/locales/ja', 'en'], ['file', 'file:///fixture/ja', 'en'],
  ['data', 'data:application/json,{}', 'en'], ['query', 'ja&ns=private', 'en'], ['stored-object', '{"lng":"ja"}', 'en'],
  ['long', 'ja-' + 'x'.repeat(100_000), 'en'],
];

for (const [name, value, language] of cases) {
  test(`bundled translations: ${name} cannot create a locale URL`, async ({ page }) => {
    const external: string[] = [], locales: string[] = [], errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    await mockDesktop(page);
    await page.addInitScript(value => {
      localStorage.setItem('i18nextLng', value);
      const saved = JSON.parse(localStorage.getItem('setting-storage')!);
      saved.state.locale = value;
      localStorage.setItem('setting-storage', JSON.stringify(saved));
    }, value);
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      if (url.protocol === 'http:' || url.protocol === 'https:') {
        if (url.origin !== 'http://127.0.0.1:4188') { external.push(url.href); return route.abort(); }
        if (url.pathname.startsWith('/locales/')) { locales.push(url.pathname); return route.abort(); }
      }
      return route.fallback();
    });
    const query = value.length < 200 ? '?lng=' + encodeURIComponent(value) + '&ns=' + encodeURIComponent('../https://fixture.invalid/private') : '';
    await page.goto('/about' + query);
    await expect(page.getByRole('region', { name: strings(language).releaseTitle })).toBeVisible();
    await expect(page.getByText(strings(language).noRelease, { exact: true })).toBeVisible();
    await expect(page.locator('body')).not.toContainText('platform.');
    await page.reload();
    await expect(page.getByRole('region', { name: strings(language).releaseTitle })).toBeVisible();
    expect(await page.evaluate(() => localStorage.getItem('i18nextLng'))).toBe(language);
    expect(locales).toEqual([]);
    expect(external).toEqual([]);
    expect(errors).toEqual([]);
  });
}

test('the shipped locale files remain available at their existing public paths', async ({ request }) => {
  for (const language of ['en', 'ja', 'zh', 'ko', 'vi']) {
    const response = await request.get(`/locales/${language}.json`);
    expect(response.ok()).toBe(true);
    expect(await response.json()).toEqual(JSON.parse(readFileSync(new URL(`../public/locales/${language}.json`, import.meta.url), 'utf8')));
  }
});
