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

test('grid can render without the toolbar for data-only browsing surfaces', () => {
  const element = host();
  TopbaseGrid(element, { columns: ['status'], rows: [['open']], hideToolbar: true, filtersEnabled: false });
  assert.doesNotMatch(element.innerHTML, /tb-grid-bar/);
  assert.match(element.innerHTML, /<table>/);
});

test('dashboard table mode is a presentation-only data table', () => {
  const element = host();
  TopbaseGrid(element, {
    columns: ['status'],
    rows: [['open'], ['closed']],
    compact: true,
    dashboardOnly: true,
    filtersEnabled: false
  });

  assert.match(element.innerHTML, /dashboard-only/);
  assert.doesNotMatch(element.innerHTML, /data-sort=/);
  assert.doesNotMatch(element.innerHTML, /tb-row-number/);
});

test('grid exposes collaborative table controls for grouping and row density', () => {
  const element = host();
  TopbaseGrid(element, { columns: ['owner', 'status'], rows: [['Ada', 'open']] });
  assert.match(element.innerHTML, /data-group/);
  assert.match(element.innerHTML, /data-row-height/);
  assert.match(element.innerHTML, /data-clear-sort/);
  assert.match(element.innerHTML, /表格视图/);
});

test('field configuration supports ordering and visibility changes without changing data columns', () => {
  const element = host();
  TopbaseGrid(element, { columns: ['owner', 'status'], rows: [['Ada', 'open']] });
  assert.match(element.innerHTML, /draggable="true" data-field="owner"/);
  assert.match(element.innerHTML, /data-move-field="up"/);
  assert.match(element.innerHTML, /data-move-field="down"/);
  assert.match(element.innerHTML, /data-toggle-column="owner"/);
  assert.match(element.innerHTML, /隐藏/);
});
