import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const root = process.cwd();
const app = fs.existsSync(path.join(root, 'frontend/public/bassaps')) ? path.join(root, 'frontend') : root;
const folder = path.join(app, 'public/bassaps');
const manifest = JSON.parse(fs.readFileSync(path.join(folder, 'manifest.json'), 'utf8'));
for (const [file, expected] of Object.entries(manifest.files)) {
  const full = path.resolve(folder, file);
  if (!full.startsWith(folder + path.sep)) throw Error('Invalid manifest path');
  const actual = crypto.createHash('sha256').update(fs.readFileSync(full)).digest('hex');
  if (actual !== expected) throw Error('Design snapshot changed: ' + file + '. Regenerate from the canonical source.');
}
function walk(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(directory, entry.name);
    return entry.isDirectory() ? walk(full) : /\.(css|tsx|vue)$/.test(entry.name) ? [full] : [];
  });
}
for (const file of [...walk(path.join(app, 'src')), ...walk(path.join(app, '.vitepress/theme'))]) {
  const text = fs.readFileSync(file, 'utf8');
  if (/#[0-9a-fA-F]{3,8}\b|rgba?\(\s*\d/.test(text)) throw Error('Raw UI color outside canonical tokens: ' + file);
}
console.log('Design snapshot verified: ' + manifest.revision + ' (' + Object.keys(manifest.files).length + ' assets).');
