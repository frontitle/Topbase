const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const source = fs.readFileSync('internal/platform/httpapi/web/question-view.js', 'utf8');

function page(fetch, formDialog) {
  const controls = new Map();
  function control(id) {
    if (!controls.has(id)) controls.set(id, {});
    return controls.get(id);
  }
  const downloads = [], messages = [];
  const scope = vm.createContext({
    location: {pathname:'/questions/q1/'}, fetch, formDialog, confirmDialog: async () => true,
    $: selector => control(selector.slice(1)), api: () => new Promise(() => {}),
    toast: text => messages.push(text), setTimeout: fn => fn(),
    URL: {createObjectURL: () => 'blob:csv', revokeObjectURL: () => {}},
    document: {
      cookie: 'topbase_csrf=test-token', getElementById: control, body: {append() {}},
      createElement: () => ({click() { downloads.push(this.download); }, remove() {}})
    }
  });
  vm.runInContext(source, scope);
  return {control, downloads, messages, scope, setQuestion: value => {
    scope.fixture = value;
    vm.runInContext('question = fixture; canRun = true', scope);
  }};
}

test('saved SQL export prompts for referenced parameters and only downloads the successful response', async () => {
  let resolve, requests = 0;
  const response = new Promise(done => { resolve = done; });
  const p = page((url, options) => {
    requests++;
    assert.equal(url, '/api/questions/q1/export');
    assert.deepEqual(JSON.parse(options.body), {parameters: {tag: 'paid,online'}});
    assert.equal(options.headers['X-Topbase-CSRF'], 'test-token');
    return response;
  }, async dialog => {
    assert.deepEqual(Array.from(dialog.fields, f => [f.label, f.required]), [['tag', true], ['region', false]]);
    return {p0: 'paid,online', p1: ''};
  });
  p.setQuestion({id:'q1', name:'销售/日报', native_sql:'select {{tag}} [[where region={{region}}]]'});
  const button = p.control('export-csv');
  const first = button.onclick();
  await button.onclick();
  await new Promise(setImmediate);
  assert.equal(button.disabled, true);
  assert.equal(requests, 1);
  resolve({ok:true, blob: async () => ({type:'text/csv;charset=utf-8'})});
  await first;
  assert.deepEqual(p.downloads, ['销售_日报.csv']);
  assert.equal(button.disabled, false);
});

test('shared SQL reader can run with parameters once and export the same values without edit controls', async () => {
  const reads = [], exports = [];
  let dialogCount = 0, rendered;
  const p = page(async (url, options) => {
    exports.push(JSON.parse(options.body));
    return {ok:true, blob:async () => ({type:'text/csv'})};
  }, async dialog => {
    dialogCount++;
    assert.equal(dialog.fields[0].label, 'tag');
    assert.equal(dialog.fields[0].value, dialogCount === 1 ? '' : 'confirmed');
    return dialogCount === 1 ? {p0:'confirmed'} : null;
  });
  const saved = {id:'q1', name:'订单', native_sql:'select {{tag}} as label', query_type:'native', can_edit:false, can_run:true, can_materialize:false};
  p.scope.api = async (url, method, params) => {
    if (url === '/api/questions/q1?view=detail') return saved;
    assert.equal(url, '/api/questions/q1/native-preview');
    assert.equal(method, 'POST');
    reads.push(JSON.parse(JSON.stringify(params)));
    return {columns:['label'], rows:[[params.parameters.tag]], chartspec:{type:'table'}};
  };
  p.scope.queryLimitMessage = () => '';
  p.scope.TopbaseCode = {setCode(){}};
  p.scope.TopbaseViz = {infer:() => ({type:'table'}), merge: spec => spec, render: (el, view) => {rendered = view;}};
  p.scope.document.body.toggleAttribute = (name, value) => {assert.equal(name, 'data-can-edit'); assert.equal(value, false);};
  await vm.runInContext('boot()', p.scope);
  assert.match(p.control('meta').textContent, /只读/);
  assert.match(p.control('viz-stage').innerHTML, /等待查询参数/);
  assert.equal(p.control('native-controls').hidden, false);
  assert.equal(p.control('export-action').hidden, false);
  assert.equal(p.control('warehouse-action').hidden, true);
  assert.equal(p.control('heading').role, undefined);
  assert.equal(reads.length, 0);
  for (const action of ['edit-analysis', 'move', 'archive', 'title']) await p.control(action).onclick();
  await p.control('native-parameters').onclick();
  assert.equal(rendered.rows[0][0], 'confirmed');
  assert.equal(rendered.onViewChange, null);
  await p.control('export-csv').onclick();
  assert.deepEqual(reads, [{parameters:{tag:'confirmed'}}]);
  assert.deepEqual(exports, reads);
  assert.equal(dialogCount, 1);
  await p.control('native-parameters').onclick();
  assert.equal(reads.length, 1);
  assert.equal(dialogCount, 2);
  assert.deepEqual(p.messages, ['CSV 已开始下载']);
});

test('canceling export with temporary filters sends no request; a denied export never downloads an error', async () => {
  let requests = 0;
  const p = page(async () => {
    requests++;
    return {ok:false, status:403, text:async () => '{"error":"data permission required"}'};
  });
  p.setQuestion({id:'q1', name:'订单数', queryir:{filters:[]}});
  vm.runInContext('temporaryFilters = true;', p.scope);
  p.scope.confirmDialog = async () => false;
  await p.control('export-csv').onclick();
  assert.equal(requests, 0);
  p.scope.confirmDialog = async () => true;
  await p.control('export-csv').onclick();
  assert.equal(requests, 1);
  assert.deepEqual(p.downloads, []);
  assert.equal(p.control('export-csv').textContent, '无导出权限');
  assert.equal(p.control('export-csv').disabled, true);
});
