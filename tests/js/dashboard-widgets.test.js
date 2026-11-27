const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');
const {surface, loadUI} = require('./helpers/popover-fixture');

function load(file, context) {
  vm.runInContext(fs.readFileSync('internal/platform/httpapi/web/' + file, 'utf8'), context, { filename: file });
}

function workspace() {
  const {doc, element, views: elements} = surface();
  doc.createElement = () => ({ style: {}, remove() { elements.delete('#ghost'); } });
  const context = vm.createContext({
    URL, Event, URLSearchParams, dispatchEvent() {}, addEventListener() {},
    document: doc, location: { pathname: '/dashboard/test/' }
  });
  context.window = context;
  loadUI(context);
  context.api = () => new Promise(() => {});
  load('dashboard-widgets.js', context);
  return { context, element, elements, catalog: context.TopbaseDashboardWidgets };
}

test('widget presets are searchable, portable and create independent plugin cards', () => {
  const { catalog } = workspace();
  const preset = { id: 'delivery', label: '交付进度', url: 'https://example.com/widget', w: 8, h: 3, config: { colors: ['blue'] } };
  catalog.registerPlugin(preset);
  preset.config.colors[0] = 'red';
  const first = catalog.create('delivery');
  assert.equal(first.type, 'plugin');
  assert.equal(first.config.url, preset.url);
  assert.equal(first.config.plugin_config.colors[0], 'blue');
  first.config.plugin_config.colors[0] = 'green';
  assert.equal(catalog.create('delivery').config.plugin_config.colors[0], 'blue');
  assert.equal(catalog.list('交付')[0].group, '嵌入与扩展');
  assert.equal(catalog.list('  DELiVery  ')[0].id, 'delivery');
  assert.equal(catalog.list('unknown-widget').length, 0);
  assert.throws(() => catalog.registerPlugin(preset), /唯一/);
  assert.throws(() => catalog.registerPlugin({ ...preset, id: 'new-one', w: 25 }), /宽度/);
  assert.throws(() => catalog.registerPlugin({ ...preset, id: 'new-one', url: 'data:text/plain,abc' }), /HTTPS/);
});

test('widget drop preview and saved position share the configured size and canvas bounds', async () => {
  const { context, elements, element } = workspace();
  const grid = {
    addEventListener() {},
    getBoundingClientRect: () => ({ width: 1194, left: 0, top: 0 }),
    appendChild(el) { elements.set('#ghost', el); }
  };
  elements.set('#grid', grid);
  const panel = element('section');
  elements.set('#palette', panel);
  elements.set('#insert-toggle', element());
  elements.set('#qsearch', element('input', panel));
  load('dashboard-layout.js', context);
  load('dashboard-view.js', context);
  vm.runInContext('board = { cards: [], appearance: { grid_version: 2 } }; editing = true; render = () => {}; saveBoard = async () => {}; bindGridDrop(); insertPopover.open()', context);
  let drawDragFrame;
  context.requestAnimationFrame = fn => { drawDragFrame = fn; };
  const event = { clientX: 1193, clientY: 90, preventDefault() {}, dataTransfer: { setData() {} } };
  context.beginPaletteDrag(event, { widgetId: 'time' });
  drawDragFrame();
  assert.equal(panel.classList.contains('palette-in-flight'), true);
  assert.equal(panel.hidden, false);
  grid.ondragover(event);
  const ghost = elements.get('#ghost');
  assert.equal(ghost.style.width, '294px');
  assert.equal(ghost.style.height, '166px');
  grid.ondrop(event);
  assert.deepEqual(JSON.parse(vm.runInContext('JSON.stringify(board.cards[0].layout)', context)), { x: 18, y: 2, w: 6, h: 4 });
  assert.equal(elements.has('#ghost'), false);
  assert.equal(panel.hidden, true);
  assert.equal(panel.classList.contains('palette-in-flight'), false);
  grid.ondrop(event);
  assert.equal(vm.runInContext('board.cards.length', context), 1);
  await context.addStatic('time');
  assert.deepEqual(JSON.parse(vm.runInContext('JSON.stringify(board.cards[1].layout)', context)), { x: 0, y: 6, w: 6, h: 4 });
});

test('insert popup manages focus, outside dismissal and mode transitions', () => {
  const { context, elements, element } = workspace();
  const doc = context.document;
  const panel = element('section'), toggle = element(), search = element('input', panel);
  for (const [key, control] of [['#grid', element()], ['#insert-toggle', toggle], ['#editor-tools', element()], ['#palette', panel], ['#qsearch', search]]) elements.set(key, control);
  load('dashboard-view.js', context);
  const empty = element(), outside = element();
  empty.setAttribute('data-empty-insert', '');
  vm.runInContext('editing = true; syncEditorTools(editing)', context);
  toggle.send('keydown', {key:'ArrowDown'});
  assert.equal(panel.hidden, false);
  assert.equal(doc.activeElement, search);
  assert.equal(toggle.attrs['aria-expanded'], 'true');
  panel.send('keydown', {key:'Escape'});
  assert.equal(panel.hidden, true);
  assert.equal(doc.activeElement, toggle);
  assert.equal(toggle.attrs['aria-expanded'], 'false');

  empty.click();
  assert.equal(panel.hidden, false);
  panel.send('keydown', {key:'Escape'});
  assert.equal(doc.activeElement, empty);
  toggle.click();
  outside.click();
  assert.equal(panel.hidden, true);
  toggle.click();
  vm.runInContext('editing = false; syncEditorTools(editing)', context);
  assert.equal(elements.get('#editor-tools').hidden, true);
  assert.equal(panel.hidden, true);
  toggle.click();
  assert.equal(panel.hidden, true);
});
