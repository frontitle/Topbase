const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync('internal/platform/httpapi/web/dashboard-view.js', 'utf8');

function eventTarget() {
  const listeners = new Map();
  return {
    addEventListener(name, callback) { listeners.set(name, callback); },
    removeEventListener(name, callback) { if (listeners.get(name) === callback) listeners.delete(name); },
    emit(name, event) { listeners.get(name)?.(event); },
  };
}

function canvas() {
  const window = eventTarget();
  const hint = { hidden: true };
  const classes = new Set();
  const el = Object.assign(eventTarget(), {
    isConnected: true,
    querySelector: () => hint,
    classList: { add: name => classes.add(name), remove: (...names) => names.forEach(name => classes.delete(name)) },
    setPointerCapture(id) { this.capture = id; },
    hasPointerCapture(id) { return this.capture === id; },
    releasePointerCapture() { this.capture = null; },
  });
  let ghost;
  const document = Object.assign(eventTarget(), {
    createElement: () => ({ style: {}, remove() { ghost = null; } })
  });
  const grid = {
    ...eventTarget(), clientWidth: 1194,
    getBoundingClientRect: () => ({ left: 0, top: 0, width: 1194 }),
    appendChild(node) { ghost = node; }, contains(node) { return node === grid; }
  };
  const context = vm.createContext({
    location: { pathname: '/dashboard/test/' }, window, document,
    $: key => ({ '#grid': grid, '#ghost': ghost })[key], $$: () => [], api: () => new Promise(() => {}),
    TopbaseUI: { iconButton: ({ label }) => '<button>' + label + '</button>' },
    componentName: () => '单行标题', esc: String,
  });
  const run = code => vm.runInContext(code, context);
  run(script);
  run(`
    editing = true;
    board = { cards: [{ id: 'x', type: 'heading', title: '单行标题', layout: { x: 0, y: 0, w: 8, h: 1 } }] };
    let saves = 0, overlaps = 0;
    scheduleSave = () => { saves++; }; resolveOverlap = () => { overlaps++; };
    selectCard = id => { selectedCard = id; }; layoutEls = scheduleCanvasLayout = () => {};
  `);
  const point = (clientX, clientY, pointerId = 4) => ({ clientX, clientY, pointerId });
  const start = (mode, interactive = false) => {
    context.press = {
      ...point(100, 100), button: 0, preventDefault() {},
      target: { closest: () => interactive ? {} : null }, currentTarget: { closest: () => el }
    };
    run("startDrag(press, 'x', " + JSON.stringify(mode) + ")");
  };
  const box = () => JSON.parse(run('JSON.stringify(payload().cards[0].layout)'));
  return { start, point, window, document, grid, ghost: () => ghost, el, hint, run, box };
}

test('card drags are transient until pointerup; inactive and foreign pointers cannot change the board', () => {
  const { start, point, window, hint, el, run, box } = canvas();
  start('move', true);
  assert.equal(run('drag'), null);
  start('move');
  window.emit('pointermove', point(102, 101));
  assert.equal(hint.hidden, true);
  window.emit('pointerup', point(102, 101));
  assert.equal(run('saves'), 0);

  start('move');
  window.emit('pointermove', point(200, 188));
  assert.equal(hint.textContent, 'X 2 · Y 2');
  assert.deepEqual(box(), { x: 0, y: 0, w: 8, h: 1 });
  window.emit('pointerup', point(100, 100, 5));
  assert.equal(el.capture, 4);
  window.emit('pointerup', point(200, 188));
  assert.deepEqual(box(), { x: 2, y: 2, w: 8, h: 1 });
  assert.equal(hint.hidden, true);
  assert.equal(el.capture, null);
  assert.equal(run('overlaps'), 1);
  assert.equal(run('saves'), 1);
});

test('Escape, pointercancel and capture loss discard the resize without saving or displacing other cards', () => {
  for (const signal of ['keydown','pointercancel','lostpointercapture','blur']) {
    const { start, point, window, el, hint, run, box } = canvas();
    start('resize');
    window.emit('pointermove', point(150, 144));
    assert.equal(hint.textContent, '9 × 2 格');
    if (signal === 'keydown') window.emit(signal, { key: 'Escape', preventDefault() {}, stopPropagation() {} });
    else (signal === 'lostpointercapture' ? el : window).emit(signal, point(150, 144));
    window.emit('pointerup', point(150, 144));
    assert.deepEqual(box(), { x: 0, y: 0, w: 8, h: 1 });
    assert.equal(hint.hidden, true);
    assert.equal(el.capture, null);
    assert.equal(run('saves'), 0);
    assert.equal(run('overlaps'), 0);
  }
});

test('resizing uses fractional grid units with snap off and skips a drop back at its origin', () => {
  const { start, point, window, run, box } = canvas();
  run('board.appearance = { snap: false }');
  start('resize');
  window.emit('pointermove', point(125, 111));
  window.emit('pointerup', point(125, 111));
  assert.deepEqual(box(), { x: 0, y: 0, w: 8.5, h: 1.25 });
  start('move');
  window.emit('pointermove', point(150, 144));
  window.emit('pointerup', point(100, 100));
  assert.equal(run('saves'), 1);
});

test('card controls are separate from the editable title and absent in the presentation', () => {
  const { run } = canvas();
  const editor = run('cardHTML(board.cards[0])');
  assert.match(editor, /class="card-tools"/);
  assert.match(editor, /位置与尺寸/);
  assert.match(editor, /tabindex="0"/);
  assert.match(editor, /contenteditable="true" data-edit="title"/);
  assert.doesNotMatch(editor, /card-head-tools-only|复制组件|打开组件/);
  const viewer = run('editing = false; cardHTML(board.cards[0])');
  assert.doesNotMatch(viewer, /card-tools|resize-handle|card-drag-hint|contenteditable|tabindex/);
});

test('insert ghost shows true grid geometry, clears on leaving or cancelling, and never changes card data', () => {
  const { run, grid, ghost, document, box } = canvas();
  const initial = box();
  run("bindGridDrop(); beginPaletteDrag({ dataTransfer: { setData() {} } }, { questionId: 'q' })");
  const over = { clientX: 1100, clientY: 88, preventDefault() {}, dataTransfer: {} };
  grid.ondragover(over);
  assert.equal(ghost().title, 'X 12 · Y 2　12 × 10 格');
  assert.equal(ghost().className, 'board-ghost hint-right');
  grid.ondragleave({ relatedTarget: null });
  assert.equal(ghost(), null);
  grid.ondragover(over);
  document.emit('dragend');
  assert.equal(ghost(), null);
  grid.ondragover(over);
  assert.equal(ghost(), null);
  assert.deepEqual(box(), initial);
  assert.equal(run('saves'), 0);
});
