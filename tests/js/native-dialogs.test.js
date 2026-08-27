const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function filesUnder(root) {
  return fs.readdirSync(root, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(root, entry.name);
    if (entry.isDirectory()) return filesUnder(full);
    return /\.(?:js|html)$/.test(entry.name) && entry.name !== 'echarts.min.js' ? [full] : [];
  });
}

test('application pages never use native browser alert, confirm or prompt dialogs', () => {
  const root = path.resolve('internal/platform/httpapi/web');
  const nativeDialog = /\b(?:window\s*\.\s*)?(?:alert|confirm|prompt)\s*\(/g;
  const failures = filesUnder(root).flatMap(file => {
    const source = fs.readFileSync(file, 'utf8');
    return [...source.matchAll(nativeDialog)].map(match => `${path.relative(root, file)}:${source.slice(0, match.index).split('\n').length}`);
  });

  assert.deepEqual(failures, [], `Found native browser dialogs:\n${failures.join('\n')}`);
});
