const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');
const {surface} = require('./helpers/popover-fixture');

const script = fs.readFileSync('internal/platform/httpapi/web/embed/access.js', 'utf8');
const serverTime = Date.parse('2026-11-27T00:00:00Z');
const passwordRequired = {error: '需要密码', code: 'password_required'};
const header = 'X-Topbase-Share-Token';

function viewer({embedded = false, denyStorage = false, savedToken = ''} = {}) {
  const {doc, element, views} = surface();
  const gate = element('main'), canvas = element('div'), form = element('form', gate);
  const password = element('input', form), submit = element('button', form), retry = element('button', gate);
  const heading = element('h1', gate), status = element('p', gate), error = element('p', form);
  status.setAttribute('role', 'status');
  error.setAttribute('role', 'alert');
  error.hidden = form.hidden = retry.hidden = canvas.hidden = true;
  password.removeAttribute = key => { delete password.attrs[key]; };
  password.select = () => { password.selected = true; };
  gate.querySelector = selector => selector === '#share-access-retry' ? retry : gate.querySelectorAll(selector)[0];
  form.querySelector = selector => form.querySelectorAll(selector)[0];
  views.set('#share-access', gate);
  views.set('#board-canvas', canvas);

  const replies = [], calls = [], timers = new Map(), stored = new Map();
  const state = {now:serverTime + 3600000, reloads:0, clears:0};
  if (savedToken) stored.set('topbase-share:fixture', savedToken);
  const storage = {
    getItem: key => stored.get(key) || null,
    setItem: (key, val) => stored.set(key, val),
    removeItem: key => stored.delete(key)
  };
  if (denyStorage) {
    for (const method of Object.keys(storage)) storage[method] = () => { throw new Error('Storage blocked'); };
  }
  const context = vm.createContext({
    document:doc, sessionStorage:storage, location:{reload: () => { state.reloads++; }},
    Date:class extends Date { static now() { return state.now; } },
    setTimeout(fn, delay) { const id = Symbol(); timers.set(id, {fn,delay}); return id; },
    clearTimeout(id) { timers.delete(id); },
    addEventListener() {},
    async fetch(url, options) {
      calls.push({url, ...options});
      assert.ok(replies.length, 'unexpected request ' + url);
      const reply = replies.shift();
      if (reply instanceof Error) throw reply;
      return {
        ok:reply.status >= 200 && reply.status < 300, status:reply.status,
        headers:{get: name => name.toLowerCase() === 'date' ? new Date(serverTime).toUTCString() : null},
        json:async () => reply.data
      };
    }
  });
  context.window = context;
  vm.runInContext(script, context);
  const api = context.TopbaseSharedDashboard.mount('fixture', embedded, {
    clearCanvas() { state.clears++; },
    openDashboard: () => api.load()
  });
  const reply = (status, data) => replies.push({status, data});
  const verify = value => { password.value = value; return form.onsubmit({preventDefault() {}}); };
  return {api, calls, replies, reply, verify, gate, canvas, form, password, submit, retry, heading, error, doc, state, stored, timers};
}

test('a stale metadata grant prompts for a password and an invalid password can be retried', async () => {
  const v = viewer({savedToken:'old-grant'});
  v.reply(401, passwordRequired);
  assert.equal(await v.api.load(), null);
  assert.equal(v.calls[0].headers[header], 'old-grant');
  assert.equal(v.stored.size, 0);
  assert.equal(v.canvas.hidden, true);
  assert.equal(v.form.hidden, false);
  assert.equal(v.doc.activeElement, v.password);

  v.reply(401, {error:'密码错误', code:'invalid_password'});
  await v.verify('incorrect');
  assert.equal(v.error.hidden, false);
  assert.equal(v.error.textContent, '密码错误');
  assert.equal(v.password.getAttribute('aria-invalid'), 'true');
  assert.equal(v.password.selected, true);
  assert.equal(v.submit.disabled, false);
  v.reply(200, {token:'fresh-grant'});
  v.reply(200, {dashboard:{name:'Protected'}});
  await v.verify('test-fixture-password');
  assert.equal(v.canvas.hidden, false);
  assert.equal(v.gate.hidden, true);
  assert.equal(v.password.value, '');
  assert.equal(v.stored.get('topbase-share:fixture'), 'fresh-grant');
  assert.equal(v.calls.at(-1).headers[header], 'fresh-grant');
  for (const req of v.calls.filter(call => call.url.endsWith('/verify'))) {
    assert.equal(req.method, 'POST');
    assert.equal(req.headers[header], undefined);
  }
});

test('an embedded dashboard with storage blocked uses an in-memory grant for metadata and card data', async () => {
  const v = viewer({embedded:true, denyStorage:true});
  v.reply(401, passwordRequired);
  await v.api.load();
  assert.notEqual(v.doc.activeElement, v.password);
  v.reply(200, {token:'frame-grant'});
  v.reply(200, {dashboard:{name:'Protected'}});
  await v.verify('test-fixture-password');
  v.reply(200, {rows:[[42]]});
  const ac = new AbortController();
  const data = await v.api.requestCard('a/b', {region:'north'}, ac.signal);
  assert.equal(data.rows[0][0], 42);
  const [challenge, verification, metadata, card] = v.calls;
  assert.equal(challenge.headers[header], undefined);
  assert.ok(verification.url.endsWith('/verify?embed=1'));
  assert.deepEqual(JSON.parse(verification.body), {password:'test-fixture-password'});
  assert.equal(metadata.headers[header], 'frame-grant');
  assert.equal(card.headers[header], 'frame-grant');
  assert.ok(card.url.endsWith('/cards/a%2Fb/dataset?embed=1'));
  assert.deepEqual(JSON.parse(card.body), {filters:{region:'north'}});
  assert.equal(card.signal, ac.signal);
  assert.ok(v.calls.every(call => call.credentials === 'omit' && call.cache === 'no-store' && call.url.endsWith('?embed=1')));
  assert.equal(v.stored.size, 0);

  v.reply(404, {error:'不可用'});
  await assert.rejects(v.api.requestCard('a/b', {}), err => v.api.handleDenial(err));
  assert.equal(v.canvas.hidden, true);
  assert.equal(v.state.clears, 1);
  assert.equal(v.heading.textContent, '分享链接不可用');
  assert.equal(v.retry.hidden, true);
  assert.equal(v.form.hidden, true);
});

test('network errors offer retry, while revoked or disabled links stay closed', async () => {
  const v = viewer();
  v.replies.push(new Error('offline'));
  assert.equal(await v.api.load(), null);
  assert.equal(v.canvas.hidden, true);
  assert.equal(v.retry.hidden, false);
  v.reply(200, {dashboard:{name:'Visible again'}});
  await v.retry.onclick();
  assert.equal(v.canvas.hidden, false);
  for (const status of [403,404,410]) {
    v.reply(status, {error:'不可用'});
    assert.equal(await v.api.load(), null);
    assert.equal(v.gate.hidden, false);
    assert.equal(v.form.hidden, true);
    assert.equal(v.retry.hidden, true);
    assert.equal(v.canvas.hidden, true);
  }
});

test('an open dashboard expires using the server clock even when the visitor clock is an hour fast', async () => {
  const v = viewer();
  v.reply(200, {dashboard:{public_expires_at:new Date(serverTime + 5000).toISOString()}});
  assert.ok(await v.api.load());
  const timer = [...v.timers.values()][0];
  assert.equal(timer.delay, 5000);
  assert.equal(v.canvas.hidden, false);
  v.state.now += 5000;
  timer.fn();
  assert.equal(v.canvas.hidden, true);
  assert.equal(v.state.reloads, 1);
});

test('metadata already expired at delivery does not expose the canvas while navigation is pending', async () => {
  const v = viewer();
  v.reply(200, {dashboard:{public_expires_at:new Date(serverTime).toISOString()}});
  assert.equal(await v.api.load(), null);
  assert.equal(v.canvas.hidden, true);
});
