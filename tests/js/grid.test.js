const assert = require('node:assert/strict');
const path = require('node:path');
const test = require('node:test');

global.window = global;
const {surface} = require('./helpers/popover-fixture');
const {doc, element: mockElement} = surface();
global.document = doc;
require(path.resolve('internal/platform/httpapi/web/ui.js'));
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
  assert.match(element.innerHTML, /data-row-height aria-label="行高"/);
  assert.match(element.innerHTML, /data-clear-sort/);
  assert.match(element.innerHTML, /表格视图/);
});

test('grouping combines nonadjacent values, keeps the existing sort within groups and uses stable source row numbers', () => {
  const element = host();
  const duplicate = ['B', 4];
  const rows = [['A', 3], duplicate, ['A', 1], duplicate, [null, 8], ['', 9]];
  const columns = ['team', 'score'];
  columns.indexOf = rows.indexOf = () => { throw Error('source array scanned during rendering'); };
  TopbaseGrid(element, {columns, rows, group:'team', sort:'score'});
  const headers = Array.from(element.innerHTML.matchAll(/class="tb-group-row".*?<b>(.*?)<\/b>/g), match => match[1]);
  const numbers = Array.from(element.innerHTML.matchAll(/<td class="tb-row-number">(\d+)<\/td>/g), match => Number(match[1]));
  assert.deepEqual(headers, ['A', 'B', '（空）']);
  assert.deepEqual(numbers, [3, 1, 2, 4, 5, 6]);
});

test('setData keeps the configured order of surviving fields and appends new fields', () => {
  const element = host();
  const grid = TopbaseGrid(element, {columns:['x', 'y', 'z'], order:['z', 'x', 'y'], hidden:{x:true}});
  grid.setData({columns:['y', 'z', 'a', 'x'], rows:[[1, 2, 3, 4]]});
  assert.deepEqual(Array.from(element.innerHTML.matchAll(/data-sort="([^"]*)"/g), match => match[1]), ['z', 'y', 'a']);
  grid.setData({columns:['a', 'y'], rows:[]});
  assert.deepEqual(Array.from(element.innerHTML.matchAll(/data-sort="([^"]*)"/g), match => match[1]), ['y', 'a']);
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

test('consecutive field actions retain the popup, focused field, and scroll until Escape or an outside click', () => {
  const element = host(), order = [];
  let panel, items, html;
  const originalSelect = element.querySelector;
  const control = (dataset, parent) => Object.assign(mockElement('button', parent), {dataset});
  Object.defineProperty(element, 'innerHTML', {
    get() {return html;},
    set(value) {
      html = value;
      const list = Object.assign(mockElement('div'), {scrollTop:0, getBoundingClientRect:() => ({top:0, bottom:160})});
      const summary = control({});
      summary.nextElementSibling = list;
      items = Array.from(value.matchAll(/draggable="true" data-field="([^"]+)"/g), ([, field], index) => {
        const toggle = control({toggleColumn:field}, list);
        const up = control({moveField:'up', col:field}, list);
        return {dataset:{field}, toggle, up, getBoundingClientRect:() => ({top:30, bottom:60}),
          querySelector(selector) {return selector === '[data-toggle-column]' ? toggle : index === 0 ? null : up;}};
      });
      panel = {open:value.includes('class="tb-cols" open'), summary, list,
        querySelector(selector) {return selector === 'summary' ? summary : list;},
        querySelectorAll() {return items;},
        contains(target) {return target === summary || items.some(item => target === item.toggle || target === item.up);}
      };
    }
  });
  element.contains = target => panel?.contains(target);
  element.querySelector = selector => {
    if (selector === '.tb-cols[open]') return panel?.open ? panel : null;
    if (selector === '.tb-cols') return panel;
    return originalSelect(selector);
  };
  element.querySelectorAll = selector => {
    if (selector === '.tb-grid-bar details') return [panel];
    if (selector === '[data-move-field]') return items.map(item => item.up);
    if (selector === '[data-toggle-column]') return items.map(item => item.toggle);
    return [];
  };
  const grid = TopbaseGrid(element, {columns:['x','y','z'], rows:[[1,2,3]], onChange:state => order.push(state)});
  document.querySelectorAll = () => panel.open ? [panel] : [];
  const y = () => items.find(item => item.dataset.field === 'y');
  try {
    panel.summary.click();
    panel.list.scrollTop = 41;
    y().up.onclick();
    assert.deepEqual(order[0].order, ['y', 'x', 'z']);
    assert.equal(panel.open, true);
    assert.equal(document.activeElement, y().toggle);
    y().toggle.onclick();
    assert.equal(order[1].hidden.y, true);
    assert.equal(panel.open, true);
    assert.equal(document.activeElement, y().toggle);
    assert.equal(panel.list.scrollTop, 41);
    y().toggle.send('focusin');
    assert.equal(panel.open, true);
    y().toggle.send('keydown', {key:'Escape'});
    assert.equal(panel.open, false);
    assert.equal(document.activeElement, panel.summary);
    panel.summary.click();
    mockElement().send('pointerdown');
    assert.equal(panel.open, false);
  } finally {
    document.activeElement = null;
    document.querySelectorAll = () => [];
    grid.destroy();
  }
});
