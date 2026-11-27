const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync('internal/platform/httpapi/web/admin/integrations/integrations.js', 'utf8');

async function boot() {
  const nodes = new Map();
  const requests = [];
  let items = [];
  function node(selector) {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: '', dataset: {}, reset() {}, focus() {}, removeAttribute() {}, replaceChildren() {}
    });
    return nodes.get(selector);
  }
  const shortcuts = ['slack', 'teams', 'google_chat'].map(provider => ({dataset: {quickProvider: provider}}));
  const api = async (path, method, body) => {
    requests.push({path, method, body});
    if (path === '/api/webhooks') {
      if (method === 'PUT') items = body.map(item => ({...item, url: '', secret: ''}));
      return items;
    }
    if (path.endsWith('/test')) return {sent: true};
    return [];
  };
  const env = {
    api, $: node, $$: selector => selector.includes('[data-quick-provider]') ? shortcuts : [],
    TopbaseUI: {modal: () => ({open() {}, close() {}}), button: ({label}) => label},
    TopbaseLocale: {t: (key, fields = {}) => key.replace(/\{(\w+)\}/g, (_, name) => fields[name]), locale: () => 'zh-CN'},
    crypto: require('node:crypto').webcrypto, Option: class {}, esc: String, toast() {}
  };
  vm.runInNewContext(source, env);
  await new Promise(resolve => setImmediate(resolve));
  const save = () => node('#hook-form').onsubmit({preventDefault() {}});
  const click = data => node('#hooks').onclick({target: {closest: () => ({dataset: data})}});
  return {node, requests, shortcuts, save, click};
}

test('quick binding saves each native card preset and sends only on explicit test', async () => {
  const {node, requests, shortcuts, save, click} = await boot();
  for (const shortcut of shortcuts) {
    const provider = shortcut.dataset.quickProvider;
    shortcut.onclick();
    assert.equal(node('#hook-provider').value, provider);
    assert.equal(node('#hook-style').value, 'card');
    assert.equal(node('#hook-url').required, true);
    assert.equal(node('#hook-json-field').hidden, true);
    assert.equal(node('#hook-secret-field').hidden, true);
    assert.match(node('#hook-guide-link').href, /^https:/);
    node('#hook-url').value = 'https://example.test/private-token';
    await save();
    const hook = requests.at(-1).body.at(-1);
    assert.equal(requests.at(-1).method, 'PUT');
    assert.equal(hook.provider, provider);
    assert.equal(hook.template, '');
    assert.doesNotMatch(node('#hooks').innerHTML, /private-token/);

    await click({edit: hook.id});
    assert.equal(node('#hook-url').required, false);
    assert.equal(node('#hook-url').value, '');
    node('#hook-provider').value = 'generic';
    node('#hook-provider').onchange();
    assert.equal(node('#hook-url').required, true);
    await click({test: hook.id});
    assert.equal(requests.at(-1).path, '/api/webhooks/' + hook.id + '/test');
    assert.equal(requests.at(-1).method, 'POST');
  }
  assert.equal(requests.filter(request => request.path.endsWith('/test')).length, 3);
});

test('only custom mode requires an editable JSON object; invalid JSON keeps the draft', async () => {
  const {node, requests, shortcuts, save} = await boot();
  shortcuts[0].onclick();
  node('#hook-style').value = 'custom';
  node('#hook-style').onchange();
  assert.equal(node('#hook-json-field').hidden, false);
  assert.equal(node('#hook-json').required, true);
  assert.equal(node('#hook-content').hidden, true);
  node('#hook-url').value = 'https://example.test/private-token';
  node('#hook-json').value = '{';
  await save();
  assert.equal(requests.some(request => request.method === 'PUT'), false);
  assert.equal(node('#hook-error').hidden, false);
  node('#hook-json').value = '{"text":"{{body}}"}';
  await save();
  assert.equal(requests.at(-1).body[0].template, node('#hook-json').value);
});

test('dashboard subscribes to the selected sanitized channel by ID', async () => {
  const dashboard = fs.readFileSync('internal/platform/httpapi/web/dashboard-view.js', 'utf8');
  const subscribe = dashboard.slice(dashboard.indexOf('async function subscribeBoard()'), dashboard.indexOf("if ($('#more'))"));
  const calls = [];
  let choices;
  const env = {
    boardId: 'ops',
    api: async (path, method, body) => {
      calls.push([path, method, body]);
      if (!method) return [{id: 'first', name: '日报'}, {id: 'second', name: '周报'}];
      return {channel: body.channel};
    },
    choiceDialog: async options => { choices = options.options; return choices[1].value; },
    dashboardText: key => key, toast() {}
  };
  await vm.runInNewContext(subscribe + '\nsubscribeBoard()', env);
  assert.equal(choices[1].label, '周报');
  assert.equal(calls[1][0], '/api/dashboards/ops/subscriptions');
  assert.equal(calls[1][2].channel, 'webhook:second');
});
