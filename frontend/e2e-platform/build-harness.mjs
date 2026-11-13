import { existsSync, writeFileSync, unlinkSync } from 'node:fs';
import { resolve } from 'node:path';
import { build } from 'vite';

if (process.env.ISOLATED_PLATFORM_TESTS !== '1') throw new Error('Run this harness only in an isolated source copy');
const entry = resolve('__platform-test.html');
if (existsSync(entry)) throw new Error('The temporary harness entry already exists');
writeFileSync(entry, '<html><body><div id="root"></div><script type="module" src="/e2e-platform/harness.tsx"></script></body></html>');
try {
  await build({
    configFile: resolve('vite.config.ts'),
    build: { rollupOptions: { input: { main: resolve('index.html'), platform: entry } } },
  });
} finally {
  unlinkSync(entry);
}
