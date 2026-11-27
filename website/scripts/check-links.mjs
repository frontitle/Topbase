import {readdir, readFile, stat} from 'node:fs/promises';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';

const dist = fileURLToPath(new URL('../dist/', import.meta.url));
const root = new URL('https://frontitle.github.io/Topbase/');
const seen = new Set();

async function checkPage(file) {
  const html = await readFile(join(dist, file), 'utf8');
  if (html.includes('__VERSION__') || html.includes('<!-- site header -->')) {
    throw new Error(`Unrendered template in ${file}`);
  }
  for (const match of html.matchAll(/<(?:a|img|link|script)\b[^>]*?\s(?:href|src)="([^"]+)"/gi)) {
    const target = new URL(match[1].replaceAll('&amp;', '&'), new URL(file, root));
    if (target.origin !== root.origin) continue;
    if (!target.pathname.startsWith(root.pathname)) {
      throw new Error(`${file} points outside GitHub Pages: ${match[1]}`);
    }
    let path = decodeURIComponent(target.pathname.slice(root.pathname.length));
    if (target.pathname.endsWith('/')) path += 'index.html';
    if (seen.has(path)) continue;
    if (!(await stat(join(dist, path)).catch(() => null))?.isFile()) {
      throw new Error(`${file}: missing ${match[1]}`);
    }
    seen.add(path);
  }
}

const pages = ['index.html', 'zh/index.html'];
for (const dir of ['docs', 'zh/docs']) {
  for (const name of await readdir(join(dist, dir))) {
    if (name.endsWith('.html')) pages.push(join(dir, name));
  }
}
for (const file of pages) await checkPage(file);
console.log(`Checked ${pages.length} HTML pages and ${seen.size} local destinations.`);
