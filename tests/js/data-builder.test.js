const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const html = fs.readFileSync(path.resolve('internal/platform/httpapi/web/data/index.html'), 'utf8');
const script = fs.readFileSync(path.resolve('internal/platform/httpapi/web/data/data.js'), 'utf8');

test('data browser keeps one canonical filter and exposes guided query steps', () => {
  assert.match(script, /TopbaseGrid\('#grid-wrap',\{[^\n]*filtersEnabled:false/);
  assert.match(html, /id="filter-bar"/);
  assert.match(html, /id="aggregation"/);
  assert.match(html, /id="group-by-field"/);
  assert.match(html, /id="join-builder"/);
  assert.match(html, /id="sort-builder"/);
  assert.match(html, /id="limit-builder"/);
  assert.match(html, /id="expression-builder"/);
  assert.match(html, /id="run-visual"[^>]*>运行并预览/);
});
