const assert = require('node:assert/strict');
const path = require('node:path');
const test = require('node:test');

global.window = global;
global.document = { activeElement: null };
require(path.resolve('internal/platform/httpapi/web/grid.js'));

function host() {
  const search = {};
  return {
    innerHTML: '',
    contains() { return false; },
    querySelector(selector) { return selector === '.tb-search' ? search : null; },
    querySelectorAll() { return []; }
  };
}

test('grid can disable its local filter controls and ignores legacy filters', () => {
  const element = host();

  TopbaseGrid(element, {
    columns: ['status'],
    rows: [['open'], ['closed']],
    filters: { status: '=open' },
    filtersEnabled: false
  });

  assert.doesNotMatch(element.innerHTML, /data-toggle-filters/);
  assert.doesNotMatch(element.innerHTML, /tb-filters/);
  assert.match(element.innerHTML, /显示 2 \/ 2 行/);
});
