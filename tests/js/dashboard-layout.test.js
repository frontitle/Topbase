const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

const source = fs.readFileSync(path.resolve('internal/platform/httpapi/web/dashboard-layout.js'), 'utf8');
const editor = vm.runInNewContext(source + '\nTopbaseDashboardLayout');
const limits = { columns: 24, minW: 4, minH: 2, maxH: 48, snap: true };

function fixture() {
  const card = { id: 'active', layout: { x: 2, y: 0, w: 5, h: 3 } };
  const options = { ...limits };
  const inputs = Object.fromEntries(['x', 'y', 'w', 'h'].map(key =>
    [key, { dataset: { layoutField: key } }]));
  const buttons = ['left', 'center', 'right'].map(edge => ({ dataset: { alignTo: edge } }));
  const host = { querySelectorAll: selector => selector === '[data-layout-field]' ? Object.values(inputs) : buttons };
  let commits = 0;
  editor.bind(host, () => card, () => options, (target, next) => {
    target.layout = next;
    commits++;
  });
  function enter(field, value) {
    inputs[field].value = Number.isNaN(value) ? '' : String(value);
    inputs[field].onchange();
  }
  return { card, inputs, buttons, options, enter, count: () => commits };
}

test('inspector shows actual grid units and clamps edits at the canvas and card bounds', () => {
  const { card, enter, inputs, options, count } = fixture();
  assert.match(editor.section(card.layout, options), /data-layout-field="x" min="0" max="19"/);
  enter('x', 999);
  assert.equal(card.layout.x, 19);
  enter('w', 16);
  assert.equal(card.layout.w, 5);
  assert.equal(count(), 1);
  enter('w', 1);
  assert.equal(card.layout.w, 4);
  assert.equal(inputs.x.max, '20');
  enter('h', 99);
  assert.equal(card.layout.h, 48);
  enter('h', -3);
  assert.equal(card.layout.h, 2);
  enter('h', NaN);
  assert.equal(inputs.h.value, '2');
  assert.equal(count(), 4);
  options.minH = 1;
  enter('h', 1);
  assert.equal(card.layout.h, 1);
});

test('quarter cell edits survive serialization and exact center uses a half cell even with snap', () => {
  const { card, enter, options, buttons, inputs } = fixture();
  buttons[1].onclick();
  assert.equal(card.layout.x, 9.5);
  buttons[2].onclick();
  assert.equal(card.layout.x, 19);
  buttons[0].onclick();
  assert.equal(card.layout.x, 0);
  enter('y', 0.8);
  assert.equal(card.layout.y, 1);
  options.snap = false;
  enter('y', 1.66);
  enter('w', 5.26);
  assert.equal(inputs.y.step, '0.25');
  assert.deepEqual(JSON.parse(JSON.stringify(card.layout)), { x: 0, y: 1.75, w: 5.25, h: 3 });
});

test('typed values commit on blur or Enter without a change event and survive a following alignment', () => {
  const { inputs, buttons, card, count } = fixture();
  inputs.w.value = '7';
  inputs.w.onblur();
  buttons[1].onclick();
  assert.equal(card.layout.w, 7);
  assert.equal(card.layout.x, 8.5);
  inputs.x.onblur();
  assert.equal(card.layout.x, 8.5);
  inputs.w.value = '9';
  inputs.w.blur = () => inputs.w.onblur();
  inputs.w.onkeydown({key: 'Enter', preventDefault() {}});
  inputs.w.onchange();
  buttons[1].onclick();
  assert.equal(card.layout.x, 7.5);
  assert.equal(count(), 4);
});

test('collision cascades move only affected cards, including multiple cards hit on one row', () => {
  const card = (id, x, y, w, h) => ({ id, layout: { x, y, w, h } });
  const anchor = card('anchor', 0, 0, 12, 4);
  const left = card('left', 0, 0, 6, 1.5);
  const right = card('right', 6, 0, 6, 1.5);
  const next = card('next', 0, 4, 12, 3);
  const tail = card('tail', 0, 6, 3, 2);
  const untouched = card('other', 14, 0, 4, 2);
  const intentionalOverlap = card('other-2', 15, 0, 4, 2);
  const cards = [intentionalOverlap, next, anchor, right, left, untouched, tail];
  editor.resolveOverlap(anchor, cards);
  assert.deepEqual([anchor, left, right, next, tail, untouched, intentionalOverlap].map(c => c.layout.y),
    [0, 4, 4, 5.5, 8.5, 0, 0]);
  assert.deepEqual(cards.map(c => c.id), ['other-2', 'next', 'anchor', 'right', 'left', 'other', 'tail']);
});
