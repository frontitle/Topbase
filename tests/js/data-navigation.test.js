const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync('internal/platform/httpapi/web/data/data.js', 'utf8');
const tables = ['orders', 'customers'].map(name => ({
  schema: 'public', name, columns: [{ name: 'id', data_type: 'integer' }]
}));
function deferred() {
  let resolve, reject;
  const promise = new Promise((ok, fail) => { resolve = ok; reject = fail; });
  return { promise, resolve, reject };
}
async function page(respond, user = {}) {
  const elements = new Map(), grids = [], messages = [], requests = [];
  function element(selector) {
    if (selector.endsWith('.table-item.active')) return null;
    if (!elements.has(selector)) elements.set(selector, { value: '', setAttribute() {} });
    return elements.get(selector);
  }
  const context = vm.createContext({
    $: element, $$: () => [], esc: String, AbortController, URLSearchParams,
    location: { search: '' }, history: { replaceState() {} },
    document: { addEventListener() {} }, toast: text => messages.push(text),
    topbasePickDatabase: () => '', topbaseRememberDatabase() {},
    TopbaseQueryEditor: { mount: () => ({ setMode() {}, setSQL() {}, setGeneratedSQL() {} }) },
    TopbaseFilter: () => ({ filters: () => [], destroy() {} }),
    TopbaseGrid: (_, options) => grids.push(options), queryLimitMessage: () => '',
    api: async (path, method, body, { signal } = {}) => {
      if (path === '/api/user/current') return user;
      if (path === '/api/databases') return [{ id: 'db' }];
      requests.push({ path, body, signal });
      // Replies can arrive even after cancellation. Exercise the UI guard too.
      const response = respond(path, body);
      if (response !== undefined) return response;
      if (path.endsWith('/tables')) return tables;
      if (path.endsWith('/annotation')) return {};
      if (path.endsWith('/fields')) return [];
      throw Error('Unexpected request ' + path);
    }
  });
  await vm.runInContext(script, context);
  return { context, element, grids, messages, requests };
}
const result = name => ({ columns: ['id'], rows: [[name]] });

test('source browser gates warehouse management and keeps materialized tables queryable', async () => {
  const saved = { schema: 'warehouse', name: 'wh_orders', warehouse: true, columns: tables[0].columns };
  for (const isAdmin of [false, true]) {
    const p = await page(path => {
      if (path.endsWith('/tables')) return [saved];
      if (path === '/api/dataset') return result('saved');
    }, { is_admin: isAdmin });
    assert.equal(p.element('#warehouse-link').hidden, !isAdmin);
    await p.context.openDatabase('db');
    await p.context.openTable(saved);
    assert.deepEqual(p.grids.map(grid => grid.rows[0][0]), ['saved']);
    assert.equal(p.requests.some(request => request.path.startsWith('/api/schedules')), false);
  }
});

test('switching tables aborts old metadata and preview and ignores late failures', async () => {
  const note = deferred(), preview = deferred();
  const p = await page((path, body) => {
    if (path.endsWith('/orders/annotation')) return note.promise;
    if (path === '/api/dataset') return body.source.table.name === 'orders' ? preview.promise : result('customers');
  });
  await p.context.openDatabase('db');
  const first = p.context.openTable(tables[0]);
  const oldRequests = p.requests.slice(-3);
  await p.context.openTable(tables[1]);
  note.resolve({ display_name: 'Old orders' });
  preview.reject(Error('delayed failure'));
  await first;
  assert.ok(oldRequests.every(request => request.signal.aborted));
  assert.equal(p.element('#table-name').textContent, 'customers');
  assert.match(p.element('#grid-status').textContent, /返回 1 行/);
  assert.deepEqual(p.grids.map(grid => grid.rows[0][0]), ['customers']);
  assert.deepEqual(p.messages, []);
});

test('only the latest filter query updates the grid when replies arrive out of order', async () => {
  const first = deferred();
  const p = await page((path, body) => {
    if (path === '/api/dataset') return body.filters?.[0]?.value === 'old' ? first.promise : result(body.filters?.[0]?.value || 'preview');
  });
  await p.context.openDatabase('db');
  await p.context.openTable(tables[0]);
  const oldFilter = p.context.runWithFilters([{ field: 'id', value: 'old' }]);
  assert.equal(p.element('#save-question').disabled, true);
  const old = p.requests.at(-1);
  await p.context.runWithFilters([{ field: 'id', value: 'new' }]);
  first.resolve(result('old'));
  await oldFilter;
  assert.equal(p.element('#save-question').disabled, false);
  assert.equal(old.signal.aborted, true);
  assert.deepEqual(p.grids.map(grid => grid.rows[0][0]), ['preview', 'new']);
});

test('opening a saved visual query runs the saved configuration exactly once', async () => {
  const p = await page(path => path === '/api/dataset' ? result('analysis') : undefined);
  await p.context.openDatabase('db');
  const query = { version: 1, source: { database_id: 'db', table: tables[0] }, limit: 17 };
  await p.context.openTable(tables[0], { query_type: 'queryir', queryir: query });
  assert.deepEqual(p.requests.filter(r => r.path === '/api/dataset').map(r => r.body), [query]);
  assert.deepEqual(p.grids.map(grid => grid.rows[0][0]), ['analysis']);
});

test('cached tables remain selectable when live schema loading fails', async () => {
  const p = await page(path => {
    if (path.endsWith('/tables')) throw Error('source offline');
    if (path.endsWith('/tables?cached=1')) return tables;
  });
  assert.equal(await p.context.openDatabase('db'), true);
  assert.match(p.element('#table-list').innerHTML, /class="table-item [^"]*" data-key="public.orders"/);
  assert.match(p.element('#schema-status').textContent, /缓存/);
  assert.equal(p.element('#schema-status').hidden, false);
});

test('navigating back cancels the directory request without starting an obsolete cache read', async () => {
  const directory = deferred();
  const p = await page(path => path.endsWith('/tables') ? directory.promise : undefined);
  const opening = p.context.openDatabase('db');
  p.context.showDatabases();
  directory.reject(Error('source offline'));
  assert.equal(await opening, false);
  assert.equal(p.requests.length, 1);
  assert.equal(p.requests[0].signal.aborted, true);
  assert.equal(p.element('#db-view').hidden, false);
  assert.deepEqual(p.messages, []);
});

test('offline without a cached schema shows the cause and only offers management to administrators', async () => {
  const p = await page(path => {
    if (path.endsWith('/tables')) throw Error('connection timeout');
    if (path.endsWith('/tables?cached=1')) return [];
  });
  assert.equal(await p.context.openDatabase('db'), false);
  assert.match(p.element('#table-list').innerHTML, /connection timeout/);
  assert.doesNotMatch(p.element('#table-list').innerHTML, /href="\/admin/);
  assert.equal(p.element('#schema-status').hidden, true);
});

test('building the filter picker indexes wide-table metadata once and preserves field labels and visibility', async () => {
  const p = await page(() => undefined);
  const count = 2000;
  let reads = 0;
  const columns = Array.from({ length: count }, (_, i) => ({ name: 'column_' + i }));
  p.context.wideColumns = columns;
  p.context.wideMetadata = columns.map(({ name }, i) => ({
    get name() { reads++; return name; },
    display_name: 'Field ' + i,
    visibility: i === 0 ? 'hidden' : 'normal'
  }));
  const fields = vm.runInContext('active={columns:wideColumns};fieldMeta=wideMetadata;columnModels()', p.context);
  assert.equal(fields.length, count - 1);
  assert.equal(fields[0].display_name, 'Field 1');
  assert.ok(reads <= count * 2, `wide-table metadata required ${reads} name reads`);
});
