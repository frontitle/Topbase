const assert = require('node:assert/strict');
const test = require('node:test');
const vm = require('node:vm');
const {surface, loadUI} = require('./helpers/popover-fixture');

function fixture() {
  const {doc, element, views} = surface();
  const dialog = element('dialog'), opener = element(), control = element('button', dialog);
  views.set('#details', dialog);
  dialog.getBoundingClientRect = () => ({ left: 30, right: 140, top: 30, bottom: 120 });
  dialog.showModal = () => { dialog.open = true; control.focus(); };
  // Native close dispatches its event in a later task.
  dialog.close = value => { dialog.open = false; dialog.returnValue = value; };
  const context = vm.createContext({ document: doc });
  context.window = context;
  loadUI(context);
  const results = [];
  const modal = context.TopbaseUI.modal('#details', { onClose: value => results.push(value) });
  return {dialog, opener, doc, control, results, modal};
}

test('native and programmatic closing finish once and restore the opening control', () => {
  const {dialog, opener, doc, control, results, modal} = fixture();
  opener.focus();
  modal.open();
  assert.equal(doc.activeElement, control);
  modal.open(); // A render may refresh the contents without changing the opener.
  dialog.close('cancel');
  dialog.send('close');
  assert.equal(doc.activeElement, opener);
  assert.deepEqual(results, ['cancel']);
  modal.open({ from: opener });
  modal.close('done');
  dialog.send('close');
  assert.deepEqual(results, ['cancel', 'done']);
});

test('a queued close from a previous opening cannot dismiss or clear the reopened dialog', () => {
  const {dialog, opener, results, modal} = fixture();
  modal.open();
  dialog.close('');
  modal.open({from: opener});
  dialog.send('close');
  assert.equal(dialog.open, true);
  assert.deepEqual(results, []);
  modal.close('second');
  assert.deepEqual(results, ['second']);
});

test('backdrop dismissal requires the press and click to both be outside the dialog', () => {
  const {dialog, opener, modal, results} = fixture();
  const inside = {clientX: 40, clientY: 38, button: 0};
  const outside = {clientX: 4, clientY: 6, button: 0};
  modal.open({ from: opener });
  dialog.send('pointerdown', inside);
  dialog.send('click', outside);
  assert.equal(dialog.open, true);
  dialog.send('pointerdown', outside);
  dialog.send('click', inside);
  assert.equal(dialog.open, true);
  dialog.send('pointerdown', outside);
  dialog.send('click', outside);
  assert.equal(dialog.open, false);
  assert.deepEqual(results, ['']);
});
