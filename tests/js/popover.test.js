const assert = require('node:assert/strict');
const test = require('node:test');
const vm = require('node:vm');
const {surface, loadUI} = require('./helpers/popover-fixture');

function fixture() {
  const f = surface();
  const context = vm.createContext({document: f.doc});
  context.window = context;
  loadUI(context);
  return {...f, popover: context.TopbaseUI.popover};
}

test('menu labels, keyboard navigation, command focus and visibility follow the same controller', () => {
  const {doc, element, popover} = fixture();
  const trigger = element(), menu = element('div'), first = element('button', menu);
  const disabled = element('button', menu), hidden = element('button', menu), link = element('a', menu);
  disabled.disabled = true;
  hidden.hidden = true;
  link.setAttribute('href','/account/');
  popover(trigger, menu);
  assert.equal(trigger.getAttribute('aria-controls'), menu.id);
  assert.equal(menu.getAttribute('aria-labelledby'), trigger.id);
  assert.equal(trigger.getAttribute('aria-haspopup'), 'menu');
  trigger.send('keydown', {key: 'ArrowUp'});
  assert.equal(doc.activeElement, link);
  assert.equal(first.getAttribute('role'), 'menuitem');
  assert.equal(link.tabIndex, -1);
  link.send('keydown', {key:'ArrowDown'});
  assert.equal(doc.activeElement, first);
  first.send('keydown', {key:'End'});
  assert.equal(doc.activeElement, link);
  let focusOnCommand;
  link.onclick = () => { focusOnCommand = doc.activeElement; };
  link.send('keydown', {key:' '});
  assert.equal(focusOnCommand, trigger);
  assert.equal(menu.hidden, true);
  trigger.click();
  assert.equal(first.send('keydown', {key:'Tab'}).defaultPrevented, undefined);
  assert.equal(doc.activeElement, trigger);
  assert.equal(menu.hidden, true);
});

test('form popovers retain native input keys; tab exits, external focus and disposal dismiss them', () => {
  const {doc, element, popover} = fixture();
  const trigger = element(), second = element(), panel = element('div'), input = element('input', panel), end = element('button', panel);
  const other = element(), menu = element('div');
  const dialog = popover(trigger, panel, {type:'dialog', initialFocus:() => input});
  const commands = popover(other, menu);
  dialog.open({from:second});
  assert.equal(doc.activeElement, input);
  assert.equal(input.send('keydown', {key:'ArrowDown'}).defaultPrevented, undefined);
  assert.equal(input.send('keydown', {key:'Escape',isComposing:true}).defaultPrevented, undefined);
  assert.equal(panel.hidden, false);
  assert.equal(input.send('keydown', {key:'Tab'}).defaultPrevented, undefined);
  assert.equal(panel.hidden, false);
  input.send('keydown', {key:'Tab',shiftKey:true});
  assert.equal(doc.activeElement, second);
  assert.equal(second.getAttribute('aria-expanded'), 'false');
  dialog.open();
  end.focus();
  end.send('keydown', {key:'Tab'});
  assert.equal(panel.hidden, true);
  dialog.open();
  commands.open();
  assert.equal(panel.hidden, true);
  assert.equal(menu.hidden, false);
  element().send('pointerdown');
  assert.equal(menu.hidden, true);
  dialog.open();
  element().focus();
  assert.equal(panel.hidden, true);
  dialog.open();
  dialog.dispose();
  trigger.click();
  assert.equal(panel.hidden, true);
});
