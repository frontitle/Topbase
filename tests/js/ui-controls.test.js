const assert = require('node:assert/strict');
const test = require('node:test');
const vm = require('node:vm');
const {loadUI} = require('./helpers/popover-fixture');

function controls() {
  const context = vm.createContext({});
  context.window = context;
  loadUI(context);
  return context;
}

test('control helpers escape labels, options and attributes and expose native semantics', () => {
  const {TopbaseUI: ui} = controls();
  const html = ui.button({label: '<编辑>', attrs: {
    disabled: false, 'aria-expanded': false, 'data-title': '"文本"', onclick: 'bad()', style: 'display:none'
  }});
  assert.match(html, /type="button" class="tb-btn tb-btn-secondary"/);
  assert.match(html, /aria-expanded="false" data-title="&quot;文本&quot;"/);
  assert.match(html, /&lt;编辑&gt;/);
  assert.doesNotMatch(html, /disabled|onclick|style=/);
  assert.throws(() => ui.iconButton({icon: 'x'}), /名称/);
  const icon = ui.iconButton({icon: 'trash-2', label: '删除', tone: 'danger'});
  assert.match(icon, /tb-icon-danger/);
  assert.match(icon, /aria-label="删除"/);
  assert.match(icon, /lucide-nav.svg#trash-2/);
  assert.equal(ui.icon('x" onload'), '');
  assert.match(ui.input({label: '备注', type: 'textarea', value: '</textarea>'}), /&lt;\/textarea&gt;<\/textarea>/);
  assert.match(ui.input({value: 0, type: 'number', attrs: {min: 0}}), /value="0" min="0"/);
  assert.match(ui.checkbox({label: '吸附', checked: true}), /type="checkbox" checked/);
  const select = ui.select({value: [0,'<'], options: [{label: '零', value: 0}, {label: '<', value: '<', disabled: true}], attrs: {multiple: true}});
  assert.match(select, /multiple><option value="0" selected>零/);
  assert.match(select, /value="&lt;" selected disabled>&lt;/);
});

test('the shared component namespace resolves specialist scripts when loaded', () => {
  const context = controls();
  const catalog = context.TopbaseUI.components;
  assert.equal(catalog.grid, undefined);
  context.TopbaseGrid = () => 'grid';
  assert.equal(catalog.grid(), 'grid');
  assert.equal(context.TopbaseUI.tabs, context.TopbaseTabs);
  assert.equal(context.TopbaseUI.formDialog, context.formDialog);
});
