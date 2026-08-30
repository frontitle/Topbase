const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function read(file) {
  return fs.readFileSync(path.resolve(file), 'utf8');
}

test('shared API supports aborting stale requests', () => {
  const ui = read('internal/platform/httpapi/web/ui.js');
  assert.match(ui, /async function api\(path, method, body, options\)/);
  assert.match(ui, /signal: options\.signal/);
  assert.match(ui, /global\.queryLimitMessage = queryLimitMessage/);
});

test('dashboard bounds parallel card loading and releases hidden data', () => {
  const dashboard = read('internal/platform/httpapi/web/dashboard-view.js');
  assert.match(dashboard, /new AbortController\(\)/);
  assert.match(dashboard, /Math\.min\(4, cards\.length\)/);
  assert.match(dashboard, /delete cardData\[id\]/);
  assert.match(dashboard, /disposeCardVisuals\(\)/);
  assert.match(dashboard, /element\._tbChart\.dispose\(\)/);
  assert.match(dashboard, /addEventListener\('pagehide'/);
});
